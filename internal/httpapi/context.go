package httpapi

import (
	"context"

	"github.com/garystansbury/tightship/internal/authz"
)

type ctxKey struct{}

func withIdentity(ctx context.Context, id authz.Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IdentityFrom returns the identity the middleware attached, or the zero Identity for a public
// route. Handlers use it for attribution; they never re-decide permission.
func IdentityFrom(ctx context.Context) authz.Identity {
	id, _ := ctx.Value(ctxKey{}).(authz.Identity)
	return id
}
