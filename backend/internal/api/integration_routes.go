// Package api — Integration routes: Atlas CMMS, ITSM webhooks (ServiceNow, Jira, 1C:TOIR).
package api

import (
	"time"

	"github.com/go-chi/chi/v5"
)

// mountIntegrationRoutes регистрирует Atlas CMMS и ITSM webhook маршруты.
func (s *Server) mountIntegrationRoutes(r chi.Router) {
	// Atlas CMMS Integration
	r.Get("/api/v1/atlas/health", s.atlasHealthCheck)
	r.Get("/api/v1/atlas/fallback/status", s.atlasFallbackStatus)
	r.Post("/api/v1/atlas/fallback/retry", s.atlasRetryFallback)
	r.Post("/api/v1/atlas/sync-asset/{deviceId}", s.atlasSyncAsset)
}

// mountWebhookRoutes регистрирует ITSM webhook-эндпоинты (rate-limited, HMAC).
func (s *Server) mountWebhookRoutes(r chi.Router) {
	if s.syncEngine == nil {
		return
	}

	r.Group(func(r chi.Router) {
		r.Use(s.newRateLimiterMiddleware(30, time.Minute))
		r.Post("/api/v1/webhooks/servicenow", s.syncEngine.ServiceNowWebhookHandler().ServeHTTP)
		r.Post("/api/v1/webhooks/jira", s.syncEngine.JiraWebhookHandler().ServeHTTP)
		r.Post("/api/v1/webhooks/toir", s.syncEngine.TOIRWebhookHandler().ServeHTTP)
	})
}
