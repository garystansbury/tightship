// Package authz is the one place a permission is decided.
//
// The model, in four sentences. Capabilities are code: a fixed catalogue of strings such as
// "ticket.comment", each meaningful only because a route enforces it. Bindings are data: which
// roles hold which capabilities, editable per deployment. Scope (school, room, queue) is context a
// grant is bounded by, not a capability. Read-only is an overlay that refuses every mutation
// regardless of what is held — impersonation, and roles that exist to observe.
//
// Every route declares the capability it needs when it is registered (see package httpapi), and
// the UI renders from the same capability set returned by GET /api/v1/me. There is therefore no
// second list to keep in step, and a control that renders but is refused cannot exist.
package authz

import (
	"fmt"
	"sort"
	"strings"
)

// Capability names one enforceable action. New capabilities ship with code; there is no runtime
// "create a permission", because a capability nothing enforces is meaningless.
type Capability string

// Scope bounds a grant or a request. An empty field means "anywhere" for a grant and "no
// particular one" for a request. A school-scoped grant never satisfies a request that names no
// school: district-wide actions need district-wide grants.
type Scope struct {
	School string
	Room   string
	Queue  string
}

// Grant is one role held by an identity, possibly bounded to a school, room or queue.
type Grant struct {
	Role  string
	Scope Scope
}

// Identity is the request-scoped result of authentication. Real is who signed in; Effective is
// who they are acting as (the same string unless impersonating). Every capability question is
// asked of Effective; every audit line names Real.
type Identity struct {
	Real      string
	Effective string
	Grants    []Grant
	ReadOnly  bool
}

// Impersonating reports whether Real and Effective differ.
func (id Identity) Impersonating() bool { return id.Real != "" && id.Real != id.Effective }

// Bindings maps a role to the capabilities it holds. Loaded once per request from the database;
// seeded by migration from the deployment's chosen defaults; edited in the Roles screen.
type Bindings map[string][]Capability

// Decision is the answer and the reason, so a refusal can be logged and explained.
type Decision struct {
	Allowed bool
	Reason  string
}

// Catalog is the fixed set of capabilities this build enforces. Modules register theirs at init;
// Register refuses duplicates so two modules cannot mean different things by one string.
var catalog = map[Capability]string{}

// Register adds a capability and its one-line description to the catalogue. Call from a module's
// init(). Panics on a duplicate: that is a programming error, not a runtime condition.
func Register(c Capability, description string) Capability {
	if !validName(string(c)) {
		panic(fmt.Sprintf("authz: capability %q must look like module.action", c))
	}
	if _, dup := catalog[c]; dup {
		panic(fmt.Sprintf("authz: capability %q registered twice", c))
	}
	catalog[c] = description
	return c
}

// Known reports whether the capability is in the catalogue.
func Known(c Capability) bool { _, ok := catalog[c]; return ok }

// Catalog returns every registered capability, sorted, for the Roles screen and for tests.
func Catalog() []Capability {
	out := make([]Capability, 0, len(catalog))
	for c := range catalog {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Describe returns the registered description, or "" for an unknown capability.
func Describe(c Capability) string { return catalog[c] }

func validName(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
				return false
			}
		}
	}
	return true
}

// Can decides whether the identity may perform cap within scope. mutating says whether the
// action writes; the read-only overlay refuses every mutation before grants are consulted.
//
// Unknown capabilities are refused: a typo in a route registration must fail closed, loudly, in
// the test that walks the route table — never quietly allow.
func Can(id Identity, b Bindings, cap Capability, scope Scope, mutating bool) Decision {
	if !Known(cap) {
		return Decision{false, fmt.Sprintf("unknown capability %q", cap)}
	}
	if mutating && id.ReadOnly {
		return Decision{false, "read-only session"}
	}
	for _, g := range id.Grants {
		if !holds(b[g.Role], cap) {
			continue
		}
		if !covers(g.Scope, scope) {
			continue
		}
		return Decision{true, "role " + g.Role + scopeNote(g.Scope)}
	}
	return Decision{false, "no grant holds " + string(cap) + scopeNote(scope)}
}

// Capabilities returns the union of capabilities the identity's grants hold, with the scope each
// applies in. This is what GET /api/v1/me returns and what the UI renders from.
func Capabilities(id Identity, b Bindings) []Held {
	seen := map[string]bool{}
	var out []Held
	for _, g := range id.Grants {
		for _, c := range b[g.Role] {
			if !Known(c) {
				continue
			}
			k := string(c) + "|" + g.Scope.School + "|" + g.Scope.Room + "|" + g.Scope.Queue
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, Held{Capability: c, Scope: g.Scope})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Capability != out[j].Capability {
			return out[i].Capability < out[j].Capability
		}
		return out[i].Scope.School < out[j].Scope.School
	})
	return out
}

// Held is one capability and the scope it is held in.
type Held struct {
	Capability Capability `json:"capability"`
	Scope      Scope      `json:"scope"`
}

func holds(caps []Capability, c Capability) bool {
	for _, x := range caps {
		if x == c {
			return true
		}
	}
	return false
}

// covers reports whether a grant's scope admits a request's scope. An empty grant field is
// "anywhere". A set grant field must equal the request's field — and a request that leaves the
// field empty is asking for the district-wide version, which a bounded grant does not cover.
func covers(grant, req Scope) bool {
	return field(grant.School, req.School) && field(grant.Room, req.Room) && field(grant.Queue, req.Queue)
}

func field(grant, req string) bool {
	if grant == "" {
		return true
	}
	return grant == req
}

func scopeNote(s Scope) string {
	var parts []string
	if s.School != "" {
		parts = append(parts, "school="+s.School)
	}
	if s.Room != "" {
		parts = append(parts, "room="+s.Room)
	}
	if s.Queue != "" {
		parts = append(parts, "queue="+s.Queue)
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
