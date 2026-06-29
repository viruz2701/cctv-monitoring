// Package api — Integration routes: Atlas CMMS, ITSM webhooks (ServiceNow, Jira, 1C:TOIR),
// Calendar Sync (Google, Outlook).
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

	// P1-CALENDAR: External Calendar Sync
	s.mountCalendarRoutes(r)
}

// mountCalendarRoutes регистрирует Calendar Sync маршруты.
func (s *Server) mountCalendarRoutes(r chi.Router) {
	if s.calendarHandler == nil {
		return
	}

	r.Route("/api/v1/integrations/calendar", func(r chi.Router) {
		r.Get("/providers", s.calendarHandler.handleListProviders)
		r.Post("/{provider}/connect", s.calendarHandler.handleConnect)
		r.Post("/{provider}/disconnect", s.calendarHandler.handleDisconnect)
		r.Get("/{provider}/status", s.calendarHandler.handleStatus)
		r.Post("/sync", s.calendarHandler.handleSync)

		// OAuth2 callback — без JWT (провайдер редиректит сюда)
		r.Get("/{provider}/callback", s.calendarHandler.handleCallback)
	})
}

// mountGraphQLRoute регистрирует GraphQL read-only endpoint (INT-13.2.4).
func (s *Server) mountGraphQLRoute(r chi.Router) {
	r.Post("/api/v1/graphql", s.handleGraphQL)
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
