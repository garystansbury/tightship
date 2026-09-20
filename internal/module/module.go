// Package module is the contract every feature area satisfies. A deployment turns modules on in
// its config; a module whose integration is absent does not register. The core binary knows
// nothing about tickets or Chromebooks — it knows how to host a Module.
package module

import (
	"embed"

	"github.com/garystansbury/tightship/internal/authz"
	"github.com/garystansbury/tightship/internal/httpapi"
)

// Module is one feature area: help desk, chromebooks, identity, network, ...
type Module interface {
	// Name is the key used in config `modules:` and in log lines.
	Name() string
	// Capabilities this module enforces. Registered with authz at init; listed here so the
	// Roles screen can group the catalogue by module.
	Capabilities() []authz.Capability
	// Routes registers the module's endpoints, each with the capability it demands.
	Routes(r *httpapi.Router)
	// Migrations is the module's embedded SQL, applied in order by the core migrator. A module
	// owns its tables; no other module writes them.
	Migrations() embed.FS
	// Nav declares the entries this module contributes, each gated by a capability. The UI shows
	// an entry only when GET /api/v1/me lists the capability.
	Nav() []NavEntry
}

// NavEntry is one place in the shell's navigation.
type NavEntry struct {
	Label      string           `json:"label"`
	Path       string           `json:"path"`
	Capability authz.Capability `json:"capability"`
	Group      string           `json:"group"` // Devices, People, Tickets, Network, Identity, Admin
}
