package session

import (
	"log/slog"
	"net/http"

	"github.com/garystansbury/tightship/internal/authz"
)

// IdentityResolver turns the session cookie into the identity the route table decides with. It is
// the production counterpart to httpapi.DebugHeaderIdentity, and it is what makes the debug
// resolver unnecessary outside development.
//
// Grants are empty until the identity module lands. That is the correct failure direction and the
// same one the bindings stub already takes: a signed-in user with no grants can reach routes that
// need only a signed-in identity, and is refused by every route that names a capability. A
// resolver that invented grants to make screens work would be the one mistake this architecture
// is arranged to prevent.
func IdentityResolver(store *Store, cookies Cookies, log *slog.Logger) func(*http.Request) (authz.Identity, bool) {
	return func(r *http.Request) (authz.Identity, bool) {
		token := cookies.Token(r)
		if token == "" {
			return authz.Identity{}, false
		}
		sess, err := store.Lookup(r.Context(), token)
		if err != nil {
			if err != ErrNotFound {
				// A database problem is not the same as a bad cookie, and treating it as one
				// would sign everybody out during a blip. It still refuses the request — but it
				// is logged as the operational event it is.
				log.Error("session lookup failed", "err", err)
			}
			return authz.Identity{}, false
		}
		// Real and Effective are the same until impersonation exists. Every audit line names
		// Real, so the column is populated correctly from the first session onwards.
		return authz.Identity{Real: sess.Subject, Effective: sess.Subject}, true
	}
}
