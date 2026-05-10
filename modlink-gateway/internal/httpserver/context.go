package httpserver

import (
	"context"

	"github.com/modlinkcloud/modlink-gateway/internal/auth"
)

type ctxKey int

const claimsCtxKey ctxKey = 1

func WithClaims(ctx context.Context, c *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsCtxKey, c)
}

func ClaimsFrom(ctx context.Context) (*auth.Claims, bool) {
	v := ctx.Value(claimsCtxKey)
	if v == nil {
		return nil, false
	}
	c, ok := v.(*auth.Claims)
	return c, ok
}
