package session

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/garystansbury/tightship/internal/authz"
)

// Grants returns the roles an authenticated subject holds. It is a function rather than an
// import so that session does not depend on identity: the session store knows who somebody is,
// and something else knows what that entitles them to.
type Grants func(ctx context.Context, subject string) ([]authz.Grant, error)

// IdentityResolver turns the session cookie into the identity the route table decides with. It is
// the production counterpart to httpapi.DebugHeaderIdentity, and it is what makes the debug
// resolver unnecessary outside development.
//
// A nil grants function means nobody holds anything, which is the correct failure direction: a
// signed-in user can still reach routes needing only a signed-in identity, and is refused by
// every route naming a capability. A resolver that invented grants to make screens work would be
// the one mistake this architecture is arranged to prevent.
func IdentityResolver(store *Store, cookies Cookies, grants Grants, log *slog.Logger) func(*http.Request) (authz.Identity, bool) {
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
		id := authz.Identity{Real: sess.Subject, Effective: sess.Subject}
		if grants != nil {
			held, err := grants(r.Context(), sess.Effective())
			if err != nil {
				// Refuse rather than proceed with an empty grant set. An empty set looks exactly
				// like "this person is entitled to nothing", so a database blip would silently
				// present every screen as though the user had been demoted.
				log.Error("could not load grants", "subject", sess.Subject, "err", err)
				return authz.Identity{}, false
			}
			id.Grants = held
		}
		return id, true
	}
}
