package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

const userContextName = "user"

// AuthMiddleware validates session tokens and sets user in Gin context
func AuthMiddleware(authService *auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from cookie
		token := GetSessionTokenFromContext(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Validate session
		user, err := authService.ValidateSession(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Set user in context
		c.Set(userContextName, user)
		c.Next()
	}
}

// OptionalAuthMiddleware validates session if present but doesn't require it
func OptionalAuthMiddleware(authService *auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := GetSessionTokenFromContext(c)
		if token != "" {
			if user, err := authService.ValidateSession(token); err == nil {
				c.Set(userContextName, user)
			}
		}
		c.Next()
	}
}

// GetUserFromContext retrieves the user from the Gin context
func GetUserFromContext(c *gin.Context) (*models.User, bool) {
	val, ok := c.Get(userContextName)
	if !ok {
		return nil, false
	}
	user, ok := val.(*models.User)
	return user, ok
}

// RequireAuth is a helper that checks if user is authenticated, writing error response if not
func RequireAuth(c *gin.Context) (*models.User, bool) {
	user, ok := GetUserFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return nil, false
	}
	return user, true
}

// GetSessionTokenFromContext extracts session token from cookie or Authorization header
func GetSessionTokenFromContext(c *gin.Context) string {
	// Try cookie first
	if cookie, err := c.Cookie("session_token"); err == nil && cookie != "" {
		return cookie
	}

	// Try Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}
