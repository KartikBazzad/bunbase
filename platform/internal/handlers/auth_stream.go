package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
)

// AuthStream provides a simple SSE endpoint that periodically validates the
// current session and notifies the client when it expires.
func (h *AuthHandler) AuthStream(c *gin.Context) {
	// Require an authenticated user via cookie
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	// Initial event
	fmt.Fprintf(c.Writer, "event: auth\n")
	fmt.Fprintf(c.Writer, "data: {\"status\":\"ok\",\"userId\":\"%s\"}\n\n", user.ID)
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Keep the connection open until client disconnects or context is done
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			// Re-check authentication; if invalid, tell client and close
			if _, ok := middleware.RequireAuth(c); !ok {
				fmt.Fprintf(c.Writer, "event: auth\n")
				fmt.Fprintf(c.Writer, "data: {\"status\":\"expired\"}\n\n")
				flusher.Flush()
				return
			}

			// Still valid; send heartbeat
			fmt.Fprintf(c.Writer, "event: auth\n")
			fmt.Fprintf(c.Writer, "data: {\"status\":\"ok\"}\n\n")
			flusher.Flush()
		}
	}
}

