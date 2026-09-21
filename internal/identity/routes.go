package identity

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/garystansbury/tightship/internal/httpapi"
	"github.com/garystansbury/tightship/internal/session"
)

// Routes registers the module's endpoints.
//
// Three of them are public, and each is public for a reason that has to be stated rather than
// assumed. Sign-in cannot require a session because issuing one is its job. The setup routes
// cannot require a session because no account exists yet to have one. Everything else declares a
// capability, and the mutating-route rule refuses to register it otherwise.
func (m Module) Routes(r *httpapi.Router) {
	s := m.Service

	// --- public: the ways in ------------------------------------------------
	r.Public(http.MethodGet, "/api/v1/auth/methods", s.handleMethods)
	r.Public(http.MethodPost, "/api/v1/auth/sign-in", s.handleSignIn)
	r.Public(http.MethodPost, "/api/v1/auth/sign-out", s.handleSignOut)
	r.Public(http.MethodGet, "/api/v1/setup", s.handleSetupStatus)
	r.Public(http.MethodPost, "/api/v1/setup", s.handleCompleteSetup)

	// --- the knobs ----------------------------------------------------------
	r.Handle(http.MethodGet, "/api/v1/auth/settings", CapAuthSettingsRead, nil, s.handleReadSettings)
	r.Handle(http.MethodPut, "/api/v1/auth/settings", CapAuthSettingsWrite, nil, s.handleWriteSettings)

	// --- your own password --------------------------------------------------
	r.Handle(http.MethodPost, "/api/v1/auth/password", CapPasswordSetOwn, nil, s.handleChangeOwnPassword)
}

// Dependencies the handlers need but the module contract does not carry. Set by cmd/tightship
// when it wires the module; the router is built after, so nothing can serve before this is done.
type Deps struct {
	Sessions *session.Store
	Cookies  session.Cookies
}

// Wire attaches the request-path dependencies.
func (s *Service) Wire(d Deps) { s.deps = d }

// ---------------------------------------------------------------------------

// handleMethods tells the sign-in screen what to offer. It is public and unauthenticated by
// necessity, so it says only which methods exist — never whether a given address has one.
func (s *Service) handleMethods(w http.ResponseWriter, r *http.Request) {
	settings, err := s.AuthSettings(r.Context())
	if err != nil {
		s.log.Error("read auth settings", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	needSetup, err := s.SetupRequired(r.Context())
	if err != nil {
		s.log.Error("setup check", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"local":          settings.LocalEnabled,
		"sso":            settings.SSOEnabled,
		"setup_required": needSetup,
	})
}

type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleSignIn verifies a local password and issues a session.
func (s *Service) handleSignIn(w http.ResponseWriter, r *http.Request) {
	var req signInRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}

	acct, mustChange, err := s.Authenticate(r.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, ErrLocalDisabled):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "local sign-in is not enabled"})
		return
	case errors.Is(err, ErrLockedOut):
		// Saying "locked" is safe: it is only ever reached for an address that has a local
		// credential AND has already failed repeatedly, so it reveals nothing a successful
		// attacker would not already know, and a person who is locked out needs to be told.
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "too many failed attempts; try again later"})
		return
	case err != nil:
		if !errors.Is(err, ErrAuth) {
			s.log.Error("sign-in failed unexpectedly", "err", err)
		}
		// One message for every reason. The form must not become a way to learn which addresses
		// are real or which accounts are disabled.
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "that email or password is not right"})
		return
	}

	token, sess, err := s.deps.Sessions.Create(r.Context(), acct.Email, session.KindStaff, clientIP(r), r.UserAgent())
	if err != nil {
		s.log.Error("could not create a session", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	s.deps.Cookies.Set(w, token, sess.ExpiresAt)
	s.log.Info("signed in", "subject", acct.Email, "method", "local")

	writeJSON(w, http.StatusOK, map[string]any{
		"email":                acct.Email,
		"display_name":         acct.DisplayName,
		"must_change_password": mustChange,
	})
}

// handleSignOut revokes the session and clears the cookie. It is public because a caller whose
// session has already expired must still be able to clear it — requiring a valid session to sign
// out leaves a stale cookie in place exactly when it is least wanted.
func (s *Service) handleSignOut(w http.ResponseWriter, r *http.Request) {
	if token := s.deps.Cookies.Token(r); token != "" {
		if err := s.deps.Sessions.Revoke(r.Context(), token); err != nil {
			s.log.Error("could not revoke a session on sign-out", "err", err)
		}
	}
	s.deps.Cookies.Clear(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed out"})
}

// handleSetupStatus says whether first-run setup is still needed, so the web app can route
// straight to the setup screen instead of showing a sign-in form nobody can use. It reveals only
// "this deployment has no accounts", which is already obvious to anyone who can reach it.
func (s *Service) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	need, err := s.SetupRequired(r.Context())
	if err != nil {
		s.log.Error("setup check", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"setup_required": need})
}

type setupRequest struct {
	Token       string `json:"token"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// handleCompleteSetup creates the first administrator.
func (s *Service) handleCompleteSetup(w http.ResponseWriter, r *http.Request) {
	var req setupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	acct, err := s.CompleteSetup(r.Context(), req.Token, req.Email, req.DisplayName, req.Password)
	if err != nil {
		if errors.Is(err, ErrWeakPassword) || strings.Contains(err.Error(), ErrWeakPassword.Error()) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.log.Warn("setup attempt refused", "err", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "that setup link is not valid"})
		return
	}
	// Sign the new administrator straight in: they have just proved possession of the setup token
	// and chosen the password, so a sign-in form here would only be a second chance to mistype it.
	token, sess, err := s.deps.Sessions.Create(r.Context(), acct.Email, session.KindStaff, clientIP(r), r.UserAgent())
	if err != nil {
		s.log.Error("could not create a session after setup", "err", err)
		writeJSON(w, http.StatusOK, map[string]any{"email": acct.Email, "signed_in": false})
		return
	}
	s.deps.Cookies.Set(w, token, sess.ExpiresAt)
	writeJSON(w, http.StatusCreated, map[string]any{"email": acct.Email, "signed_in": true})
}

func (s *Service) handleReadSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.AuthSettings(r.Context())
	if err != nil {
		s.log.Error("read auth settings", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// handleWriteSettings changes the knobs. The refusals live in the service, not here, so that the
// same rules apply to any other caller — a CLI, a future installer — rather than only to requests
// that happen to arrive through this handler.
func (s *Service) handleWriteSettings(w http.ResponseWriter, r *http.Request) {
	var want AuthSettings
	if err := decodeJSON(w, r, &want); err != nil {
		return
	}
	id := httpapi.IdentityFrom(r.Context())
	if err := s.SetAuthSettings(r.Context(), want, id.Real); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	settings, _ := s.AuthSettings(r.Context())
	writeJSON(w, http.StatusOK, settings)
}

type changePasswordRequest struct {
	Current string `json:"current_password"`
	New     string `json:"new_password"`
}

// handleChangeOwnPassword changes the signed-in account's password. The current password is
// required even though the caller is already signed in: a stolen session should not be enough to
// take permanent ownership of an account.
func (s *Service) handleChangeOwnPassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	id := httpapi.IdentityFrom(r.Context())
	if id.Impersonating() {
		// Changing a password while impersonating would be an account takeover with somebody
		// else's name on it.
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "a password cannot be changed while impersonating"})
		return
	}

	acct, _, err := s.Authenticate(r.Context(), id.Effective, req.Current)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "that password is not right"})
		return
	}
	if err := s.SetPassword(r.Context(), acct.ID, req.New, false); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	// Every other session for this account ends. A password change is what somebody does after
	// suspecting their account was used, and it has to actually evict whoever was using it.
	if n, err := s.deps.Sessions.RevokeAllFor(r.Context(), acct.Email); err != nil {
		s.log.Error("could not revoke sessions after a password change", "err", err)
	} else {
		s.log.Info("password changed; sessions revoked", "subject", acct.Email, "revoked", n)
	}
	token, sess, err := s.deps.Sessions.Create(r.Context(), acct.Email, session.KindStaff, clientIP(r), r.UserAgent())
	if err != nil {
		s.deps.Cookies.Clear(w)
		writeJSON(w, http.StatusOK, map[string]string{"status": "password changed; please sign in again"})
		return
	}
	s.deps.Cookies.Set(w, token, sess.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}

// ---------------------------------------------------------------------------

// decodeJSON reads a JSON body with a size limit and strict field checking. A body without a
// limit is a memory-exhaustion surface on an unauthenticated route, and an unknown field that is
// silently ignored is a setting the caller believes they sent.
func decodeJSON(w http.ResponseWriter, r *http.Request, into any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "that request body is not valid"})
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// clientIP returns the peer address. X-Forwarded-For is deliberately NOT consulted here: it is
// caller-supplied unless a trusted proxy list is applied to it, and server.trusted_proxies is not
// wired into the request path yet. Recording the proxy's address is wrong but harmless; recording
// an attacker's chosen address in an audit trail is worse.
func clientIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}
