package auth

import (
	"net/http"
	"strings"
)

// Middleware returns an HTTP middleware that validates JWT Bearer tokens.
// When secret is nil or empty, the middleware does not require auth (pass-through).
func Middleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(secret) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			if auth == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing authorization")
				return
			}
			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				writeAuthError(w, http.StatusUnauthorized, "invalid authorization")
				return
			}
			tokenString := strings.TrimPrefix(auth, prefix)
			claims, err := ValidateToken(tokenString, secret)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

// RequireRole returns a middleware that checks the request context has at least the given role.
// Must be used after Middleware so context has claims.
func RequireRole(required Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeAuthError(w, http.StatusUnauthorized, "missing claims")
				return
			}
			if !claims.HasRole(required) {
				writeAuthError(w, http.StatusForbidden, "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
