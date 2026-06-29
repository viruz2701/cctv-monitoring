package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	apimw "gb-telemetry-collector/internal/api/middleware"
	"gb-telemetry-collector/internal/db"

	"golang.org/x/crypto/bcrypt"
)

// API key rate limiter (INT-13.2.3, P1-RATE)
// Redis-based rate limiter, инициализируется в APIKeyMiddleware.
// Лимит: 100 req/min per API key.
const (
	apiKeyRateLimit  = 100             // 100 requests
	apiKeyRateWindow = 1 * time.Minute // per minute
)

// APIKeyMiddleware validates API keys from X-API-Key header
// и применяет Redis-based rate limiting per key (P1-RATE).
//
// Соответствует:
//   - INT-13.2.3: API key rate limiting
//   - P1-RATE: Redis-based distributed rate limiting
//   - OWASP ASVS V2.2.1 (Rate limiting)
func (s *Server) APIKeyMiddleware(next http.Handler) http.Handler {
	// P1-RATE: Инициализируем Redis-based rate limiter для API keys
	// Если Redis недоступен — используем fail-open (пропускаем запросы)
	var apiKeyLimiter *apimw.RateLimiter
	if s.rateLimitRedis != nil {
		apiKeyLimiter = apimw.NewRateLimiter(s.rateLimitRedis, apiKeyRateLimit, apiKeyRateLimit, apiKeyRateWindow)
	}

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

		// P1-RATE: Redis-based rate limiting per API key
		if apiKeyLimiter != nil {
			allowed, current, limit, err := apiKeyLimiter.Allow(r.Context(), "apikey:"+matchedKey.ID, r.Method)
			if err != nil {
				// Fail-open: при ошибке Redis пропускаем запрос
				s.logger.Warn("API key rate limiter Redis error, allowing request",
					"key_id", matchedKey.ID,
					"error", err,
				)
			} else if !allowed {
				RespondError(w, r, NewRateLimitError(
					"API key rate limit exceeded (100 req/min)"))
				return
			} else {
				// X-RateLimit headers для API key
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, limit-current)))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(apiKeyRateWindow).Unix(), 10))
			}
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
