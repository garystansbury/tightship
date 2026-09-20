// Package httpapi is the route table and the one authorization middleware.
//
// A route is registered with the capability it requires. That registration IS the permission
// list: there is no allow-list elsewhere to keep in step, and the test in this package walks the
// table and fails the build if a mutating route declares none. GET /api/v1/me returns the same
// capability set the middleware decides with, so the UI renders from one truth.
package httpapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/garystansbury/tightship/internal/authz"
)

// Route is one registered endpoint and what it demands.
type Route struct {
	Method     string
	Pattern    string
	Capability authz.Capability // "" only for Public routes
	Public     bool             // no identity required (health, static assets, sign-in)
}

// Mutating reports whether the route's method writes. GET and HEAD read; everything else writes.
func (r Route) Mutating() bool { return r.Method != http.MethodGet && r.Method != http.MethodHead }

// IdentityResolver turns a request into who is asking. The real implementation reads the
// session; the development one reads a header when the deployment allows it.
type IdentityResolver func(*http.Request) (authz.Identity, bool)

// BindingsSource returns the current role->capability bindings. Loaded once per request.
type BindingsSource func(*http.Request) authz.Bindings

// ScopeExtractor derives the scope a request is about (from its path or body) so Can can bound
// a school-scoped grant. The default asks for the district-wide version, which only district-wide
// grants satisfy — the safe direction.
type ScopeExtractor func(*http.Request) authz.Scope

// Router owns the mux and the table.
type Router struct {
	mux      *http.ServeMux
	routes   []Route
	identity IdentityResolver
	bindings BindingsSource
	log      *slog.Logger
}

// New builds a router. identity and bindings are required; a nil identity resolver would make
// every protected route unreachable, which is the correct default and is what New enforces.
func New(identity IdentityResolver, bindings BindingsSource, log *slog.Logger) *Router {
	if identity == nil {
		identity = func(*http.Request) (authz.Identity, bool) { return authz.Identity{}, false }
	}
	if bindings == nil {
		bindings = func(*http.Request) authz.Bindings { return authz.Bindings{} }
	}
	if log == nil {
		log = slog.Default()
	}
	r := &Router{mux: http.NewServeMux(), identity: identity, bindings: bindings, log: log}
	r.Public(http.MethodGet, "/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle(http.MethodGet, "/api/v1/me", "", nil, r.me)
	return r
}

// Handle registers a protected route. cap is the capability it requires; a mutating route with
// no capability is refused at registration, because that is the exact mistake this design exists
// to make impossible. A GET with no capability requires only a signed-in identity.
func (r *Router) Handle(method, pattern string, cap authz.Capability, scope ScopeExtractor, h http.HandlerFunc) {
	rt := Route{Method: method, Pattern: pattern, Capability: cap}
	if rt.Mutating() && cap == "" {
		panic(fmt.Sprintf("httpapi: %s %s mutates but declares no capability", method, pattern))
	}
	if cap != "" && !authz.Known(cap) {
		panic(fmt.Sprintf("httpapi: %s %s declares unknown capability %q", method, pattern, cap))
	}
	if scope == nil {
		scope = func(*http.Request) authz.Scope { return authz.Scope{} }
	}
	r.routes = append(r.routes, rt)
	r.mux.HandleFunc(method+" "+pattern, func(w http.ResponseWriter, req *http.Request) {
		id, ok := r.identity(req)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
			return
		}
		if cap != "" {
			d := authz.Can(id, r.bindings(req), cap, scope(req), rt.Mutating())
			if !d.Allowed {
				r.log.Info("refused", "real", id.Real, "effective", id.Effective, "cap", cap, "route", method+" "+pattern, "why", d.Reason)
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "not permitted", "capability": string(cap)})
				return
			}
		}
		h(w, req.WithContext(withIdentity(req.Context(), id)))
	})
}

// Public registers a route that needs no identity: health, static assets, the sign-in flow.
func (r *Router) Public(method, pattern string, h http.HandlerFunc) {
	r.routes = append(r.routes, Route{Method: method, Pattern: pattern, Public: true})
	r.mux.HandleFunc(method+" "+pattern, h)
}

// Routes returns the table, sorted, for the test that audits it and for `tightship routes`.
func (r *Router) Routes() []Route {
	out := append([]Route(nil), r.routes...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pattern != out[j].Pattern {
			return out[i].Pattern < out[j].Pattern
		}
		return out[i].Method < out[j].Method
	})
	return out
}

// ServeHTTP makes the router an http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) { r.mux.ServeHTTP(w, req) }

// me answers GET /api/v1/me: who you are, who you are acting as, and every capability you hold
// with its scope. The UI's navigation and controls are rendered from this and nothing else.
func (r *Router) me(w http.ResponseWriter, req *http.Request) {
	id := IdentityFrom(req.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"real":          id.Real,
		"effective":     id.Effective,
		"impersonating": id.Impersonating(),
		"read_only":     id.ReadOnly,
		"capabilities":  authz.Capabilities(id, r.bindings(req)),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// DebugHeaderIdentity is the development resolver: X-Tightship-User names the effective identity
// and X-Tightship-Roles lists "role[@school]" grants. It exists so the app runs without an
// identity provider on a laptop. It must only be installed when config dev.allow_debug_identity
// is true, and main refuses to install it otherwise.
func DebugHeaderIdentity(req *http.Request) (authz.Identity, bool) {
	u := strings.TrimSpace(req.Header.Get("X-Tightship-User"))
	if u == "" {
		return authz.Identity{}, false
	}
	id := authz.Identity{Real: u, Effective: u}
	for _, g := range strings.Split(req.Header.Get("X-Tightship-Roles"), ",") {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		role, school, _ := strings.Cut(g, "@")
		id.Grants = append(id.Grants, authz.Grant{Role: role, Scope: authz.Scope{School: school}})
	}
	return id, true
}
