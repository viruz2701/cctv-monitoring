package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

// APIKeyMiddleware validates API keys from X-API-Key header
func (s *Server) APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Try Authorization header as fallback
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			http.Error(w, "API key required", http.StatusUnauthorized)
			return
		}

		// Hash the provided key
		hash := sha256.Sum256([]byte(apiKey))
		keyHash := hex.EncodeToString(hash[:])

		// Look up the key in database
		key, err := s.db.GetAPIKeyByHash(keyHash)
		if err != nil {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Check if key is expired
		if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
			http.Error(w, "API key expired", http.StatusUnauthorized)
			return
		}

		// Update last used timestamp (async to not block request)
		go func() {
			_ = s.db.UpdateAPIKeyLastUsed(key.ID)
		}()

		// Add key info to context for handlers
		ctx := r.Context()
		ctx = setAPIKeyContext(ctx, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
