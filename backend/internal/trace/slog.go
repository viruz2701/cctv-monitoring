// Package trace — slog.Handler обёртка для автоматического добавления trace_id.
package trace

import (
	"context"
	"log/slog"
)

// ── LogHandler ───────────────────────────────────────────────────────

// LogHandler оборачивает slog.Handler и автоматически добавляет trace_id
// из context.Context во все записи лога.
//
// Использование:
//
//	handler := trace.NewLogHandler(slog.Default().Handler())
//	slog.SetDefault(slog.New(handler))
//
// Теперь все логи автоматически содержат trace_id (если он есть в контексте).
type LogHandler struct {
	next slog.Handler
}

// NewLogHandler создаёт LogHandler, оборачивающий существующий handler.
func NewLogHandler(next slog.Handler) *LogHandler {
	return &LogHandler{next: next}
}

// Enabled reports whether the handler handles records at the given level.
func (h *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle добавляет trace_id к записи лога если он есть в контексте.
func (h *LogHandler) Handle(ctx context.Context, record slog.Record) error {
	if traceID := FromContext(ctx); traceID != "" {
		record.AddAttrs(slog.String("trace_id", traceID))
	}
	return h.next.Handle(ctx, record)
}

// WithAttrs creates a new handler with the given attributes.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{next: h.next.WithAttrs(attrs)}
}

// WithGroup creates a new handler with the given group.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{next: h.next.WithGroup(name)}
}
