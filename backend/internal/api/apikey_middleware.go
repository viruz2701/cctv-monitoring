package api

import (
	"sync"

	"gb-telemetry-collector/internal/db"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// API key rate limiter (INT-13.2.3)
var apiKeyRateLimiters sync.Map // map[string]*rateLimiter (key_id → limiter)

const (
	apiKeyRateLimit    = 100  // 100 requests
	apiKeyRateWindow   = 1 * time.Minute // per minute
)

// getAPIKeyRateLimiter возвращает rate limiter для API key.
func getAPIKeyRateLimiter(keyID string) *rateLimiter {
	if limiter, ok := apiKeyRateLimiters.Load(keyID); ok {
		return limiter.(*rateLimiter)
	}
	limiter := newRateLimiter(apiKeyRateLimit, apiKeyRateWindow)
	apiKeyRateLimiters.Store(keyID, limiter)
	return limiter
}

// APIKeyMiddleware validates API keys from X-API-Key header
// и применяет rate limiting per key (INT-13.2.3).
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
			RespondError(w, r, NewUnauthorizedError("API key required"))
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
			RespondError(w, r, NewUnauthorizedError("Invalid API key"))
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
			RespondError(w, r, NewUnauthorizedError("Invalid API key"))
			return
		}

		// Check if key is expired
		if matchedKey.ExpiresAt != nil && matchedKey.ExpiresAt.Before(time.Now()) {
			RespondError(w, r, NewUnauthorizedError("API key expired"))
			return
		}

		// INT-13.2.3: Rate limiting per API key
		limiter := getAPIKeyRateLimiter(matchedKey.ID)
		if !limiter.allow(matchedKey.ID) {
			RespondError(w, r, NewRateLimitError("API key rate limit exceeded (100 req/min)"))
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
