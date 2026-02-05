package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// HandleCollectionSubscribe provides SSE stream for collection-level document changes.
// GET /api/projects/:id/database/collections/:collection/subscribe
func (h *DatabaseHandler) HandleCollectionSubscribe(c *gin.Context) {
	projectID := c.Param("id")
	collection := c.Param("collection")

	if projectID == "" || collection == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID and collection are required"})
		return
	}

	// Verify access - support API key from query param (for EventSource) or header
	key := c.GetHeader("X-Bunbase-Client-Key")
	if key == "" {
		key = c.Query("key")
	}
	if key != "" {
		keyProjectID, err := h.projectService.GetProjectIDByPublicKey(key)
		if err == nil && keyProjectID == projectID {
			// Authorized via API key
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}
	} else {
		// Try user auth
		user, ok := middleware.RequireAuth(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID)
		if err != nil || !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	if h.subscriptionManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "realtime subscriptions not available"})
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

	// Subscribe to all documents in this collection (no predicate = match all)
	log.Printf("[SSE] Starting subscription for project=%s collection=%s", projectID, collection)
	events, cancel := h.subscriptionManager.Subscribe(c.Request.Context(), projectID, collection, nil)
	defer cancel()

	// Send initial connection event
	if _, err := c.Writer.WriteString("event: connected\n"); err != nil {
		log.Printf("[SSE] Error writing connected event: %v", err)
		return
	}
	if _, err := c.Writer.WriteString(fmt.Sprintf("data: {\"projectId\":\"%s\",\"collection\":\"%s\"}\n\n", projectID, collection)); err != nil {
		log.Printf("[SSE] Error writing connected data: %v", err)
		return
	}
	flusher.Flush()

	// Stream events
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-events:
			if !ok {
				// Channel closed, send close event and return
				_, _ = c.Writer.WriteString("event: close\n")
				_, _ = c.Writer.WriteString("data: {}\n\n")
				flusher.Flush()
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			_, _ = c.Writer.WriteString("event: change\n")
			_, _ = c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", string(data)))
			flusher.Flush()
		}
	}
}

// HandleQuerySubscribe provides SSE stream for query-based document changes.
// POST /api/projects/:id/database/collections/:collection/documents/query/subscribe
func (h *DatabaseHandler) HandleQuerySubscribe(c *gin.Context) {
	projectID := c.Param("id")
	collection := c.Param("collection")

	if projectID == "" || collection == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID and collection are required"})
		return
	}

	// Verify access - support API key from query param (for EventSource) or header
	key := c.GetHeader("X-Bunbase-Client-Key")
	if key == "" {
		key = c.Query("key")
	}
	if key != "" {
		keyProjectID, err := h.projectService.GetProjectIDByPublicKey(key)
		if err == nil && keyProjectID == projectID {
			// Authorized via API key
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}
	} else {
		// Try user auth
		user, ok := middleware.RequireAuth(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID)
		if err != nil || !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	// Parse query body (same format as regular query endpoint)
	var req struct {
		Query     map[string]interface{} `json:"query"`
		Skip      int                    `json:"skip"`
		Limit     int                    `json:"limit"`
		SortField string                 `json:"sortField"`
		SortDesc  bool                   `json:"sortDesc"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if h.subscriptionManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "realtime subscriptions not available"})
		return
	}

	// Build a simple predicate function for the query
	pred := buildQueryPredicate(req.Query)

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

	// Subscribe with predicate
	log.Printf("[SSE] Starting query subscription for project=%s collection=%s", projectID, collection)
	events, cancel := h.subscriptionManager.Subscribe(c.Request.Context(), projectID, collection, pred)
	defer cancel()

	// Send initial connection event
	if _, err := c.Writer.WriteString("event: connected\n"); err != nil {
		log.Printf("[SSE] Error writing connected event: %v", err)
		return
	}
	if _, err := c.Writer.WriteString(fmt.Sprintf("data: {\"projectId\":\"%s\",\"collection\":\"%s\"}\n\n", projectID, collection)); err != nil {
		log.Printf("[SSE] Error writing connected data: %v", err)
		return
	}
	flusher.Flush()

	// Stream events
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-events:
			if !ok {
				// Channel closed, send close event and return
				log.Printf("[SSE] Event channel closed for project=%s collection=%s", projectID, collection)
				_, _ = c.Writer.WriteString("event: close\n")
				_, _ = c.Writer.WriteString("data: {}\n\n")
				flusher.Flush()
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				log.Printf("[SSE] Error marshaling event: %v", err)
				continue
			}
			if _, err := c.Writer.WriteString("event: change\n"); err != nil {
				log.Printf("[SSE] Error writing change event: %v", err)
				return
			}
			if _, err := c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", string(data))); err != nil {
				log.Printf("[SSE] Error writing change data: %v", err)
				return
			}
			flusher.Flush()
		}
	}
}

// buildQueryPredicate creates a simple predicate function from a query map.
// For MVP, supports basic equality and field existence checks.
func buildQueryPredicate(query map[string]interface{}) services.QueryPredicate {
	if len(query) == 0 {
		return nil // nil predicate = match all
	}

	return func(doc map[string]interface{}) bool {
		for field, value := range query {
			docVal, exists := doc[field]
			if !exists {
				return false
			}
			// Simple equality check (can be extended for ranges, etc.)
			if docVal != value {
				return false
			}
		}
		return true
	}
}
