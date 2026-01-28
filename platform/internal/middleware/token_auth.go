package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// AuthAnyMiddleware authenticates using either:
// - Authorization: Bearer <api-token>  (for CLI / programmatic access), or
// - session_token cookie (for browser sessions).
// If neither mechanism yields a valid user, it aborts with 401.
func AuthAnyMiddleware(authService *auth.Auth, tokenService *services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) Try API token from Authorization header
		if user, ok := authenticateWithAPIToken(c, authService, tokenService); ok {
			c.Set(userContextName, user)
			c.Next()
			return
		}

		// 2) Fall back to cookie-based session auth
		token := GetSessionTokenFromContext(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, err := authService.ValidateSession(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set(userContextName, user)
		c.Next()
	}
}

// authenticateWithAPIToken validates an Authorization: Bearer <token> header
// against the api_tokens table. It returns (user, true) on success, or (nil, false)
// if there is no bearer token. If a bearer token is present but invalid/expired,
// it writes a 401 response and returns (nil, true) to signal that the request
// has been handled.
func authenticateWithAPIToken(
	c *gin.Context,
	authService *auth.Auth,
	tokenService *services.TokenService,
) (*models.User, bool) {
	authHeader := c.GetHeader("Authorization")
	const prefix = "Bearer "

	if authHeader == "" {
		return nil, false
	}
	if !strings.HasPrefix(authHeader, prefix) {
		// Malformed Authorization header; treat as no bearer token and let other
		// auth mechanisms handle it.
		return nil, false
	}

	raw := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if raw == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return nil, true
	}

	token, err := tokenService.GetTokenByValue(raw)
	if err != nil {
		// Bearer token was present but invalid -> hard 401
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return nil, true
	}

	// Check expiry if set
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		_ = tokenService.RevokeToken(token.ID)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		return nil, true
	}

	// Load associated user
	user, err := authService.GetUserByID(token.UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token user"})
		return nil, true
	}

	_ = tokenService.MarkTokenUsed(token.ID)
	return user, true
}
