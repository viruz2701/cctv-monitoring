package logging

import (
	"io"
	"log/slog"
	"os"

	"gb-telemetry-collector/internal/trace"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	FilePath   string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
	Level      slog.Level
	AddSource  bool
}

// NewLogger создаёт структурированный логгер с JSON форматом.
// Логгер автоматически обёрнут в trace.LogHandler для включения trace_id
// из context.Context во все записи.
func NewLogger(cfg Config) *slog.Logger {
	var writers []io.Writer

	// Файл с ротацией
	if cfg.FilePath != "" {
		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}
		writers = append(writers, lumberjackLogger)
	}

	// Всегда пишем в stdout (для docker/k8s)
	writers = append(writers, os.Stdout)

	multiWriter := io.MultiWriter(writers...)

	handlerOpts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	// JSON handler для структурированных логов (лучше для анализа)
	handler := slog.NewJSONHandler(multiWriter, handlerOpts)

	// Оборачиваем в trace.LogHandler для автоматического включения trace_id
	tracedHandler := trace.NewLogHandler(handler)

	return slog.New(tracedHandler)
}
