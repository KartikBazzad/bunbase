package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

const userContextName = "user"

// AuthMiddleware validates session tokens and sets user in Gin context
// Uses SessionService for unified session management (routes to bun-auth or tenant-auth based on session type)
func AuthMiddleware(sessionService *services.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from cookie (platform users use "session_token", tenant users use "tenant_session_token")
		sessionToken := GetSessionTokenFromContext(c)
		if sessionToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Validate session via SessionService (routes to appropriate auth service)
		user, err := sessionService.ValidateSession(c.Request.Context(), sessionToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Set user in context
		c.Set(userContextName, user)
		c.Next()
	}
}

// AuthMiddlewareLegacy is kept for backward compatibility during migration
// TODO: Remove after full migration to SessionService
func AuthMiddlewareLegacy(authService *auth.Auth) gin.HandlerFunc {
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
func OptionalAuthMiddleware(sessionService *services.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken := GetSessionTokenFromContext(c)
		if sessionToken != "" {
			if user, err := sessionService.ValidateSession(c.Request.Context(), sessionToken); err == nil {
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
// Checks both "session_token" (platform) and "tenant_session_token" (tenant) cookies
func GetSessionTokenFromContext(c *gin.Context) string {
	// Try platform session cookie first
	if cookie, err := c.Cookie("session_token"); err == nil && cookie != "" {
		return cookie
	}

	// Try tenant session cookie
	if cookie, err := c.Cookie("tenant_session_token"); err == nil && cookie != "" {
		return cookie
	}

	// Try Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}
