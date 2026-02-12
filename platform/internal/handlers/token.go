package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// TokenHandler handles API token management endpoints.
type TokenHandler struct {
	tokenService *services.TokenService
	limitService *services.LimitService
}

// NewTokenHandler creates a new TokenHandler.
func NewTokenHandler(tokenService *services.TokenService, limitService *services.LimitService) *TokenHandler {
	return &TokenHandler{tokenService: tokenService, limitService: limitService}
}

// CreateTokenRequest represents a request to create a new API token.
type CreateTokenRequest struct {
	Name   string        `json:"name"`
	Scopes string        `json:"scopes,omitempty"`
	TTL    time.Duration `json:"ttl,omitempty"` // optional, in seconds
}

// CreateTokenResponse includes the token metadata and the raw token value.
type CreateTokenResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Scopes     string     `json:"scopes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	Token      string     `json:"token"` // raw token value, returned once
}

// Create creates a new API token for the authenticated user.
func (h *TokenHandler) Create(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if h.limitService != nil {
		if err := h.limitService.CheckAPITokenLimit(c.Request.Context(), user.ID.String()); err != nil {
			if errors.Is(err, services.ErrAPITokenLimitReached) {
				c.JSON(http.StatusForbidden, gin.H{"error": h.limitService.LimitMessage(err)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	ttl := req.TTL
	if ttl <= 0 {
		// default to 30 days
		ttl = 30 * 24 * time.Hour
	}

	token, raw, err := h.tokenService.CreateToken(user.ID.String(), req.Name, req.Scopes, ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := CreateTokenResponse{
		ID:         token.ID,
		Name:       token.Name,
		Scopes:     token.Scopes,
		CreatedAt:  token.CreatedAt,
		ExpiresAt:  token.ExpiresAt,
		LastUsedAt: token.LastUsedAt,
		Token:      raw,
	}

	c.JSON(http.StatusCreated, resp)
}

// List returns all API tokens for the authenticated user (without raw token).
func (h *TokenHandler) List(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	tokens, err := h.tokenService.ListTokensForUser(user.ID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// Delete revokes a specific API token by ID.
func (h *TokenHandler) Delete(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token id is required"})
		return
	}

	// Optional: ensure token belongs to user
	tokens, err := h.tokenService.ListTokensForUser(user.ID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	owned := false
	for _, t := range tokens {
		if t.ID == id {
			owned = true
			break
		}
	}
	if !owned {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	if err := h.tokenService.RevokeToken(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token revoked"})
}

