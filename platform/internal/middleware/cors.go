package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const clientKeyHeader = "X-Bunbase-Client-Key"

// CORSMiddleware returns a Gin middleware that sets CORS headers.
// When the request includes X-Bunbase-Client-Key (header or query param) or preflight requests it, the request's Origin is allowed.
// Otherwise only allowedOrigin (e.g. dashboard) is allowed.
func CORSMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		clientKey := c.Request.Header.Get(clientKeyHeader)
		if clientKey == "" {
			// Also check query param (for EventSource which can't send custom headers)
			clientKey = c.Query("key")
		}
		requestedHeaders := c.Request.Header.Get("Access-Control-Request-Headers")

		hasClientKey := clientKey != "" || strings.Contains(strings.ToLower(requestedHeaders), strings.ToLower(clientKeyHeader))

		if hasClientKey && origin != "" {
			// Allow the requesting origin when a client key is present or will be sent (any origin with a client key)
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, "+clientKeyHeader)
		c.Header("Access-Control-Allow-Credentials", "true")

		// Handle preflight
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
