package identity

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/garystansbury/tightship/internal/authz"
	"github.com/garystansbury/tightship/internal/httpapi"
)

// actorCapabilities is the set the requester actually holds, derived from the same identity and
// bindings the middleware decided with. It is the input to every escalation check.
//
// Scope is deliberately ignored here. A school-scoped grant of a capability still proves the
// actor has been trusted with that capability, and role editing is a district-wide act — so the
// alternative, letting a school-scoped holder bind it district-wide, is the escalation. Binding
// is gated by roles.bind, which a district hands out district-wide or not at all.
func (s *Service) actorCapabilities(r *http.Request) map[authz.Capability]bool {
	id := httpapi.IdentityFrom(r.Context())
	bindings, err := s.Bindings(r.Context())
	if err != nil {
		s.log.Error("could not read bindings for an escalation check", "err", err)
		return map[authz.Capability]bool{} // fail closed: hold nothing, grant nothing
	}
	held := map[authz.Capability]bool{}
	for _, h := range authz.Capabilities(id, bindings) {
		held[h.Capability] = true
	}
	return held
}

func (m Module) roleRoutes(r *httpapi.Router) {
	s := m.Service
	r.Handle(http.MethodGet, "/api/v1/capabilities", CapRoleRead, nil, s.handleCatalog)
	r.Handle(http.MethodGet, "/api/v1/roles", CapRoleRead, nil, s.handleListRoles)
	r.Handle(http.MethodPost, "/api/v1/roles", CapRoleBind, nil, s.handleCreateRole)
	r.Handle(http.MethodPut, "/api/v1/roles/{name}/capabilities", CapRoleBind, nil, s.handleSetCapabilities)
	r.Handle(http.MethodDelete, "/api/v1/roles/{name}", CapRoleBind, nil, s.handleDeleteRole)

	r.Handle(http.MethodGet, "/api/v1/grants", CapRoleRead, nil, s.handleListGrants)
	r.Handle(http.MethodPost, "/api/v1/grants", CapRoleGrant, nil, s.handleGrant)
	r.Handle(http.MethodDelete, "/api/v1/grants/{id}", CapRoleGrant, nil, s.handleRevokeGrant)
}

// handleCatalog returns every capability this build enforces, with its description, so the roles
// screen can offer the real set rather than a list someone typed into the frontend.
func (s *Service) handleCatalog(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Capability  string `json:"capability"`
		Description string `json:"description"`
		Held        bool   `json:"held_by_you"`
	}
	held := s.actorCapabilities(r)
	var out []entry
	for _, c := range authz.Catalog() {
		// held_by_you is what lets the screen grey out what this person cannot delegate, so the
		// escalation refusal is visible before it is hit rather than after.
		out = append(out, entry{string(c), authz.Describe(c), held[c]})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Service) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := s.Roles(r.Context())
	if err != nil {
		s.log.Error("list roles", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, roles)
}

func (s *Service) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	role, err := s.CreateRole(r.Context(), req.Name, req.Description)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, role)
}

func (s *Service) handleSetCapabilities(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	want := make([]authz.Capability, 0, len(req.Capabilities))
	for _, c := range req.Capabilities {
		want = append(want, authz.Capability(c))
	}
	id := httpapi.IdentityFrom(r.Context())
	err := s.SetRoleCapabilities(r.Context(), r.PathValue("name"), want, s.actorCapabilities(r), id.Real)
	if err != nil {
		writeJSON(w, statusFor(err), map[string]string{"error": err.Error()})
		return
	}
	role, _ := s.RoleByName(r.Context(), r.PathValue("name"))
	writeJSON(w, http.StatusOK, role)
}

func (s *Service) handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	id := httpapi.IdentityFrom(r.Context())
	if err := s.DeleteRole(r.Context(), r.PathValue("name"), id.Real); err != nil {
		writeJSON(w, statusFor(err), map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Service) handleListGrants(w http.ResponseWriter, r *http.Request) {
	grants, err := s.ListGrants(r.Context())
	if err != nil {
		s.log.Error("list grants", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, grants)
}

func (s *Service) handleGrant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email  string `json:"email"`
		Role   string `json:"role"`
		School string `json:"school"`
		Room   string `json:"room"`
		Queue  string `json:"queue"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	id := httpapi.IdentityFrom(r.Context())
	scope := authz.Scope{School: req.School, Room: req.Room, Queue: req.Queue}
	err := s.GrantRole(r.Context(), req.Email, req.Role, scope, s.actorCapabilities(r), id.Real)
	if err != nil {
		writeJSON(w, statusFor(err), map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "granted"})
}

func (s *Service) handleRevokeGrant(w http.ResponseWriter, r *http.Request) {
	grantID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "that is not a grant id"})
		return
	}
	id := httpapi.IdentityFrom(r.Context())
	if err := s.RevokeGrant(r.Context(), grantID, id.Real); err != nil {
		writeJSON(w, statusFor(err), map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// statusFor maps the refusals to codes a client can act on. An escalation attempt is 403 rather
// than 400: the request was well formed and the answer is that this person may not do it.
func statusFor(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errorIs(err, ErrWouldEscalate), errorIs(err, ErrRoleProtected):
		return http.StatusForbidden
	case errorIs(err, ErrLastAdministrator):
		return http.StatusConflict
	case errorIs(err, ErrRoleNotFound), errorIs(err, ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}

func errorIs(err, target error) bool { return errors.Is(err, target) }
