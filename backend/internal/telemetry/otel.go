// Package telemetry — OpenTelemetry integration.
//
// P3-1.3: OpenTelemetry Integration
//   - Trace context propagation
//   - Jaeger/Zipkin compatible exporter (OTLP HTTP)
//   - Performance impact <5%
//
// Compliance: ISO 27001 A.12.4 (audit trail), OWASP ASVS V7 (logging)
package telemetry

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config для OpenTelemetry.
type Config struct {
	Enabled     bool   `mapstructure:"enabled"`
	Endpoint    string `mapstructure:"endpoint"`
	ServiceName string `mapstructure:"service_name"`
	Environment string `mapstructure:"environment"`
}

// InitTracer инициализирует TracerProvider с OTLP HTTP exporter.
// Возвращает nil, если отключено (cfg.Enabled == false).
func InitTracer(cfg Config, logger *slog.Logger) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		logger.Info("OpenTelemetry disabled")
		return nil, nil
	}

	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.DeploymentEnvironment(cfg.Environment),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	logger.Info("OpenTelemetry initialized",
		"endpoint", cfg.Endpoint,
		"service", cfg.ServiceName,
		"env", cfg.Environment,
	)
	return tp, nil
}

// StartSpan создаёт span с указанным именем в контексте.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer("cctv-health-monitor")
	return tracer.Start(ctx, name)
}
