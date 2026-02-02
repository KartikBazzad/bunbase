package auth

import "context"

type contextKey string

const claimsKey contextKey = "claims"

// WithClaims returns a context with the given claims.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

// ClaimsFromContext returns claims from the context, or nil if not set.
func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}
