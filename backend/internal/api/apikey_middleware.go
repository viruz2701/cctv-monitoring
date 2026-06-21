package api

import (
	"gb-telemetry-collector/internal/db"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
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
			respondError(w, r, NewUnauthorizedError("API key required"))
			return
		}

		// Extract prefix for lookup (first 8 chars of the key)
		prefix := ""
		if len(apiKey) >= 8 {
			prefix = apiKey[:8]
		}

		// Look up keys by prefix
		keys, err := s.db.GetAPIKeysByPrefix(prefix)
		if err != nil || len(keys) == 0 {
			respondError(w, r, NewUnauthorizedError("Invalid API key"))
			return
		}

		// Verify with bcrypt — try each matching key
		var matchedKey *db.APIKey
		for i := range keys {
			if err := bcrypt.CompareHashAndPassword([]byte(keys[i].KeyHash), []byte(apiKey)); err == nil {
				matchedKey = &keys[i]
				break
			}
		}

		if matchedKey == nil {
			respondError(w, r, NewUnauthorizedError("Invalid API key"))
			return
		}

		// Check if key is expired
		if matchedKey.ExpiresAt != nil && matchedKey.ExpiresAt.Before(time.Now()) {
			respondError(w, r, NewUnauthorizedError("API key expired"))
			return
		}

		// Update last used timestamp (async to not block request)
		go func() {
			_ = s.db.UpdateAPIKeyLastUsed(matchedKey.ID)
		}()

		// Add key info to context for handlers
		ctx := r.Context()
		ctx = setAPIKeyContext(ctx, matchedKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
