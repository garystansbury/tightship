package identity

import (
	"embed"
	"io/fs"

	"github.com/garystansbury/tightship/internal/authz"
	"github.com/garystansbury/tightship/internal/module"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Capabilities this module enforces. They are registered at init so the catalogue is complete
// before any route table is built; a route naming an unregistered capability panics at
// registration, which is the point.
var (
	// Changing how people sign in is the most consequential setting in the product: it is the one
	// that can lock everybody out, and the one an attacker would most like to reach.
	CapAuthSettingsRead  = authz.Register("auth.settings.read", "View sign-in settings")
	CapAuthSettingsWrite = authz.Register("auth.settings.write", "Change sign-in settings, including whether local passwords are accepted")

	CapAccountRead    = authz.Register("account.read", "View accounts")
	CapAccountCreate  = authz.Register("account.create", "Create accounts")
	CapAccountUpdate  = authz.Register("account.update", "Change an account's details")
	CapAccountDisable = authz.Register("account.disable", "Disable an account and end its sessions")

	// Setting somebody else's password is a distinct power from editing their name, because it is
	// the power to become them.
	CapPasswordSetOwn   = authz.Register("password.set_own", "Change your own password")
	CapPasswordSetOther = authz.Register("password.set_other", "Set another account's local password")
)

// Module is the identity module: accounts, the credentials that prove them, and the settings that
// govern which credentials are accepted.
type Module struct {
	Service *Service
}

func (Module) Name() string { return "identity" }

func (Module) Capabilities() []authz.Capability {
	return []authz.Capability{
		CapAuthSettingsRead, CapAuthSettingsWrite,
		CapAccountRead, CapAccountCreate, CapAccountUpdate, CapAccountDisable,
		CapPasswordSetOwn, CapPasswordSetOther,
	}
}

func (Module) Migrations() fs.FS { return migrations }

func (Module) Nav() []module.NavEntry {
	return []module.NavEntry{
		{Label: "Accounts", Path: "/admin/accounts", Capability: CapAccountRead, Group: "Admin"},
		{Label: "Sign-in", Path: "/admin/sign-in", Capability: CapAuthSettingsRead, Group: "Admin"},
	}
}

// Routes lives in routes.go, with the handlers.

// Compile-time proof the module satisfies the contract it claims, so a missing method is a build
// failure here rather than a wiring failure in cmd.
var _ module.Module = Module{}
