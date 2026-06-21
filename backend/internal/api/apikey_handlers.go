package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/auth"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

// handleCreateAPIKey creates a new API key
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	var req struct {
		Name        string     `json:"name"`
		Permissions []string   `json:"permissions"`
		ExpiresAt   *time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	if req.Name == "" {
		respondError(w, r, NewBadRequestError("name is required"))
		return
	}

	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		s.logger.Error("failed to generate API key", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	apiKey := "sk_live_" + hex.EncodeToString(keyBytes)

	// Hash the key with bcrypt (cost=12)
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), 12)
	if err != nil {
		s.logger.Error("failed to hash API key", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	keyHash := string(hash)

	// Generate ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		s.logger.Error("failed to generate ID", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	id := hex.EncodeToString(idBytes)

	// Extract prefix for lookup
	keyPrefix := apiKey[:8]

	// Save to database
	if err := s.db.CreateAPIKey(id, req.Name, keyHash, keyPrefix, req.Permissions, req.ExpiresAt, claims.UserID); err != nil {
		s.logger.Error("failed to create API key", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
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
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	keys, err := s.db.GetAPIKeys(claims.UserID)
	if err != nil {
		s.logger.Error("failed to get API keys", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	jsonResponse(w, http.StatusOK, keys)
}

// handleRevokeAPIKey revokes an API key
func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	if err := s.db.RevokeAPIKey(id, claims.UserID); err != nil {
		s.logger.Error("failed to revoke API key", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "revoked"})
}
