package session

import (
	"net/http"
	"time"
)

// The cookie carries the token and nothing else. Every attribute below is load-bearing, so they
// are set in one place rather than at each call site where one could be forgotten.

// Cookies writes and clears the session cookie.
type Cookies struct {
	Name string
}

// Set attaches the session cookie to a response.
//
//   - HttpOnly: script cannot read the token, so an XSS bug cannot exfiltrate the session.
//   - Secure: never sent over plain HTTP. Browsers treat http://localhost as a secure context,
//     so development still works without weakening this.
//   - SameSite=Lax: the cookie is withheld from cross-site POSTs, which is most of CSRF, but is
//     still sent on a top-level navigation — which is exactly what the return leg of an OpenID
//     Connect sign-in is. SameSite=Strict would drop the cookie on the way back from the identity
//     provider and land the user on a signed-out page immediately after signing in.
//   - Path=/ and no Domain: required by the __Host- prefix, and narrower than the alternative —
//     a host-only cookie is not sent to any sibling subdomain.
//
// MaxAge is the absolute lifetime, not the idle window. The browser's copy expiring early would
// sign someone out while the server still considered the session live; the server is the
// authority on the idle window, and it checks it on every lookup.
func (c Cookies) Set(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear removes the cookie. It must repeat the attributes used to set it — a browser matches a
// deletion to an existing cookie by name, path and domain, so a Clear that omits Path=/ leaves
// the original in place.
func (c Cookies) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Token reads the token out of a request, or "" if there is no cookie.
func (c Cookies) Token(r *http.Request) string {
	ck, err := r.Cookie(c.Name)
	if err != nil {
		return ""
	}
	return ck.Value
}
