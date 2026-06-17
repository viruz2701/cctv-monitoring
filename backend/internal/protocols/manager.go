package protocols

import (
	"context"
	"fmt"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/sip"

	"log/slog"
)

type ProtocolHandler interface {
	Start(ctx context.Context) error
	Stop() error
}

type Manager struct {
	handlers []ProtocolHandler
	logger   *slog.Logger
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{logger: logger}
}

func (m *Manager) Register(h ProtocolHandler) {
	m.handlers = append(m.handlers, h)
}

func (m *Manager) StartAll(ctx context.Context) error {
	for _, h := range m.handlers {
		if err := h.Start(ctx); err != nil {
			m.logger.Error("Failed to start protocol handler", "error", err)
			// не прерываем, а логируем
		}
	}
	return nil
}

func (m *Manager) StopAll() {
	for _, h := range m.handlers {
		if err := h.Stop(); err != nil {
			m.logger.Error("Failed to stop protocol handler", "error", err)
		}
	}
}

func (m *Manager) ReloadService(serviceName string, newConfig interface{}) error {
	for i, h := range m.handlers {
		switch serviceName {
		case "gb28181":
			if sipHandler, ok := h.(*sip.SIPHandler); ok {
				m.logger.Info("Reloading GB28181 handler", "new_port", newConfig.(config.GB28181Config).Port)
				sipHandler.Stop()
				newHandler := sip.NewSIPHandler(sipHandler.GetStateManager(), m.logger, newConfig.(config.GB28181Config))
				if err := newHandler.Start(context.Background()); err != nil {
					return err
				}
				m.handlers[i] = newHandler
				return nil
			}
		}
	}
	return fmt.Errorf("service %s not found", serviceName)
}
