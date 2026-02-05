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
// - Authorization: Bearer <api-token>  (for CLI / programmatic access),
// - X-Bunbase-Client-Key header (user API token, for SDK / demo app), or
// - session_token cookie (for browser sessions).
// If none yields a valid user, it aborts with 401.
func AuthAnyMiddleware(authService *auth.Auth, tokenService *services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) Try API token from Authorization header
		if user, ok := authenticateWithAPIToken(c, authService, tokenService); ok {
			c.Set(userContextName, user)
			c.Next()
			return
		}

		// 2) Try X-Bunbase-Client-Key as user API token (SDK / demo app)
		if key := c.GetHeader("X-Bunbase-Client-Key"); key != "" {
			if user, ok := resolveUserFromTokenValue(c, key, authService, tokenService, false); ok {
				c.Set(userContextName, user)
				c.Next()
				return
			}
		}

		// 3) Fall back to cookie-based session auth
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
	return resolveUserFromTokenValue(c, raw, authService, tokenService, true)
}

// resolveUserFromTokenValue validates a raw token string against api_tokens and returns (user, true) on success.
// When abortOnInvalid is true (e.g. Bearer), invalid/expired token results in 401 and (nil, true).
// When false (e.g. X-Bunbase-Client-Key), invalid token returns (nil, false) so other auth can run.
func resolveUserFromTokenValue(
	c *gin.Context,
	raw string,
	authService *auth.Auth,
	tokenService *services.TokenService,
	abortOnInvalid bool,
) (*models.User, bool) {
	token, err := tokenService.GetTokenByValue(raw)
	if err != nil {
		if abortOnInvalid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return nil, true
		}
		return nil, false
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		_ = tokenService.RevokeToken(token.ID)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		return nil, true
	}
	user, err := authService.GetUserByID(token.UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token user"})
		return nil, true
	}
	_ = tokenService.MarkTokenUsed(token.ID)
	return user, true
}

const projectKeyProjectIDContextKey = "project_key_project_id"

// ProjectKeyOrUserAuthMiddleware authenticates using either user auth (Bearer, X-Bunbase-Client-Key as user token, cookie)
// or project API key (X-Bunbase-Client-Key as project key when key's project matches route :id).
// Use for /v1/projects/:id/... routes so SDK can use a single project API key.
func ProjectKeyOrUserAuthMiddleware(
	authService *auth.Auth,
	tokenService *services.TokenService,
	projectService *services.ProjectService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) Try user auth (same as AuthAnyMiddleware)
		if user, ok := authenticateWithAPIToken(c, authService, tokenService); ok {
			c.Set(userContextName, user)
			c.Next()
			return
		}
		if key := c.GetHeader("X-Bunbase-Client-Key"); key != "" {
			if user, ok := resolveUserFromTokenValue(c, key, authService, tokenService, false); ok {
				c.Set(userContextName, user)
				c.Next()
				return
			}
		}
		token := GetSessionTokenFromContext(c)
		if token != "" {
			if user, err := authService.ValidateSession(token); err == nil {
				c.Set(userContextName, user)
				c.Next()
				return
			}
		}

		// 2) No user: try X-Bunbase-Client-Key as project API key (header or query param for SSE)
		key := c.GetHeader("X-Bunbase-Client-Key")
		if key == "" {
			key = c.Query("key")
		}
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		projectID, err := projectService.GetProjectIDByPublicKey(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}
		routeProjectID := c.Param("id")
		if routeProjectID == "" || projectID != routeProjectID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "api key does not match project"})
			return
		}
		c.Set(projectKeyProjectIDContextKey, projectID)
		c.Next()
	}
}

// GetProjectKeyProjectID returns the project ID set when authorized via project API key, or "" if not set.
func GetProjectKeyProjectID(c *gin.Context) string {
	v, ok := c.Get(projectKeyProjectIDContextKey)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
