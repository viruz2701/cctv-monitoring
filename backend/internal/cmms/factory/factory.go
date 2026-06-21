// Package factory содержит фабрику для создания CMMSRouter на основе конфигурации.
// Вынесен в отдельный пакет для избежания циклических импортов между cmms и адаптерами.
package factory

import (
	"log/slog"

	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/cmms/jira"
	"gb-telemetry-collector/internal/cmms/servicenow"
	"gb-telemetry-collector/internal/cmms/toir"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/db"
)

// NewCMMSRouterFromConfig создаёт CMMSRouter на основе конфигурации.
// Поддерживаемые адаптеры: atlas, servicenow, toir, jira, internal (по умолчанию).
func NewCMMSRouterFromConfig(cfg *config.Config, database *db.DB) *cmms.CMMSRouter {
	switch cfg.CMMSAdapter {
	case "atlas":
		adapter, err := cmms.NewAtlasAdapter(cmms.AtlasAdapterConfig{
			BaseURL:      cfg.AtlasURL,
			ClientID:     cfg.AtlasClientID,
			ClientSecret: cfg.AtlasClientSecret,
			TokenURL:     cfg.AtlasTokenURL,
			APIKey:       cfg.AtlasAPIKey,
			FallbackDir:  cfg.AtlasFallbackDir,
		})
		if err != nil {
			slog.Error("failed to create Atlas adapter, falling back to internal", "error", err)
			return cmms.NewCMMSRouter(cmms.NewInternalAdapter(database))
		}
		return cmms.NewCMMSRouter(adapter)
	case "servicenow":
		adapter, err := servicenow.NewAdapter(servicenow.AdapterConfig{
			InstanceURL:  cfg.ServiceNowInstanceURL,
			ClientID:     cfg.ServiceNowClientID,
			ClientSecret: cfg.ServiceNowClientSecret,
			TokenURL:     cfg.ServiceNowTokenURL,
			Username:     cfg.ServiceNowUsername,
			Password:     cfg.ServiceNowPassword,
			FallbackDir:  cfg.ServiceNowFallbackDir,
		})
		if err != nil {
			slog.Error("failed to create ServiceNow adapter, falling back to internal", "error", err)
			return cmms.NewCMMSRouter(cmms.NewInternalAdapter(database))
		}
		return cmms.NewCMMSRouter(adapter)
	case "toir":
		adapter, err := toir.NewAdapter(toir.AdapterConfig{
			BaseURL:     cfg.TOIRBaseURL,
			Username:    cfg.TOIRUsername,
			Password:    cfg.TOIRPassword,
			FallbackDir: cfg.TOIRFallbackDir,
		})
		if err != nil {
			slog.Error("failed to create TOIR adapter, falling back to internal", "error", err)
			return cmms.NewCMMSRouter(cmms.NewInternalAdapter(database))
		}
		return cmms.NewCMMSRouter(adapter)
	case "jira":
		adapter, err := jira.NewAdapter(jira.AdapterConfig{
			BaseURL:     cfg.JiraBaseURL,
			Email:       cfg.JiraEmail,
			APIToken:    cfg.JiraAPIToken,
			FallbackDir: cfg.JiraFallbackDir,
		})
		if err != nil {
			slog.Error("failed to create Jira adapter, falling back to internal", "error", err)
			return cmms.NewCMMSRouter(cmms.NewInternalAdapter(database))
		}
		return cmms.NewCMMSRouter(adapter)
	default:
		return cmms.NewCMMSRouter(cmms.NewInternalAdapter(database))
	}
}
