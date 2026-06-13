package logging

import (
    "io"
    "log/slog"
    "os"

    "gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
    FilePath    string
    MaxSizeMB   int
    MaxBackups  int
    MaxAgeDays  int
    Compress    bool
    Level       slog.Level
    AddSource   bool
}

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
    return slog.New(handler)
}