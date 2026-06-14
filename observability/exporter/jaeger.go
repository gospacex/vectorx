package exporter

import (
	"context"

	"github.com/gospacex/vectorx/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

// buildOTLP returns an OTLP gRPC or HTTP exporter per cfg.Protocol.
func buildOTLP(ctx context.Context, cfg *config.TracingConfig) (*otlptrace.Exporter, error) {
	if cfg.Protocol == "http" {
		return otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.Endpoint),
			otlptracehttp.WithInsecure(),
		)
	}
	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
}
