package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/storage"
)

// StorageHandler handles project-scoped file storage (one bucket per project).
type StorageHandler struct {
	client *storage.Client
}

// NewStorageHandler creates a StorageHandler.
func NewStorageHandler(client *storage.Client) *StorageHandler {
	return &StorageHandler{client: client}
}

// safeKey returns the object key from the path param and false if invalid.
func safeKey(pathParam string) (string, bool) {
	key := strings.TrimPrefix(strings.TrimSpace(pathParam), "/")
	if key == "" {
		return "", false
	}
	if strings.Contains(key, "..") {
		return "", false
	}
	return key, true
}

// List returns objects in the project bucket. GET /v1/storage?prefix=...
func (h *StorageHandler) List(c *gin.Context) {
	if h.client == nil || !h.client.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not configured"})
		return
	}
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}
	prefix := c.Query("prefix")
	objects, err := h.client.ListObjects(c.Request.Context(), projectID, prefix)
	if err != nil {
		if err == storage.ErrDisabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"objects": objects})
}

// Get streams an object. GET /v1/storage/*path
func (h *StorageHandler) Get(c *gin.Context) {
	if h.client == nil || !h.client.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not configured"})
		return
	}
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}
	key, ok := safeKey(c.Param("path"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing object path"})
		return
	}
	result, err := h.client.GetObject(c.Request.Context(), projectID, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "object not found"})
		return
	}
	defer result.Reader.Close()
	contentType := result.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	size := result.Size
	if size < 0 {
		size = 0
	}
	c.DataFromReader(http.StatusOK, size, contentType, result.Reader, nil)
}

// Put uploads an object. PUT /v1/storage/*path — body is the file bytes.
func (h *StorageHandler) Put(c *gin.Context) {
	if h.client == nil || !h.client.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not configured"})
		return
	}
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}
	key, ok := safeKey(c.Param("path"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing object path"})
		return
	}
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	body := c.Request.Body
	if body == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body required"})
		return
	}
	err := h.client.PutObject(c.Request.Context(), projectID, key, body, c.Request.ContentLength, contentType)
	if err != nil {
		if err == storage.ErrDisabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key})
}

// Delete removes an object. DELETE /v1/storage/*path
func (h *StorageHandler) Delete(c *gin.Context) {
	if h.client == nil || !h.client.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not configured"})
		return
	}
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}
	key, ok := safeKey(c.Param("path"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing object path"})
		return
	}
	err := h.client.DeleteObject(c.Request.Context(), projectID, key)
	if err != nil {
		if err == storage.ErrDisabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": key})
}
