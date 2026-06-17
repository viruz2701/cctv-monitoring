package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/auth"

	"github.com/go-chi/chi/v5"
)

// handleCreateAPIKey creates a new API key
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name        string     `json:"name"`
		Permissions []string   `json:"permissions"`
		ExpiresAt   *time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		s.logger.Error("failed to generate API key", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	apiKey := "sk_live_" + hex.EncodeToString(keyBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// Generate ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		s.logger.Error("failed to generate ID", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id := hex.EncodeToString(idBytes)

	// Save to database
	if err := s.db.CreateAPIKey(id, req.Name, keyHash, req.Permissions, req.ExpiresAt, claims.UserID); err != nil {
		s.logger.Error("failed to create API key", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return the key (only time it's shown in plain text)
	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"id":          id,
		"name":        req.Name,
		"api_key":     apiKey,
		"permissions": req.Permissions,
		"expires_at":  req.ExpiresAt,
		"created_at":  time.Now(),
	})
}

// handleListAPIKeys returns all API keys for the current user
func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keys, err := s.db.GetAPIKeys(claims.UserID)
	if err != nil {
		s.logger.Error("failed to get API keys", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, keys)
}

// handleRevokeAPIKey revokes an API key
func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := s.db.RevokeAPIKey(id, claims.UserID); err != nil {
		s.logger.Error("failed to revoke API key", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "revoked"})
}
