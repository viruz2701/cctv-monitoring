package protocols

import (
    "context"
   
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