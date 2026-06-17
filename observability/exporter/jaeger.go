package exporter

import (
	"context"
	"fmt"

	"github.com/gospacex/vectorx/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

// buildOTLP returns an OTLP gRPC or HTTP exporter per cfg.Protocol.
//
// TLS posture is driven by cfg.Insecure (from the mqx TracingConfig):
//
//   - Insecure == true  → use plaintext gRPC / HTTP  (dev / localhost only)
//   - Insecure == false → use default TLS credentials (production)
//
// Custom headers (typically "Authorization: Bearer …") flow through
// cfg.Headers and are injected as gRPC / HTTP metadata on every call.
// mqx's Validate() auto-fills an "Authorization: Basic …" header when
// Username/Password are set, so callers that want bearer auth can just
// set cfg.Headers["Authorization"] directly and skip Username/Password.
func buildOTLP(ctx context.Context, cfg *config.TracingConfig) (*otlptrace.Exporter, error) {
	if cfg.Endpoint == "" {
		// mqx's Validate() normally fills this, but Build() can be called
		// before Validate (e.g. from a test that constructs the config
		// struct directly). Fail loud instead of silently using
		// localhost:4317 — that would send production spans to a local
		// collector with no warning.
		return nil, fmt.Errorf("exporter.Build: cfg.Endpoint is empty")
	}

	if cfg.Protocol == "http" {
		return buildOTLPHTTP(ctx, cfg)
	}
	return buildOTLPGRPC(ctx, cfg)
}

func buildOTLPGRPC(ctx context.Context, cfg *config.TracingConfig) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
	}
	return otlptracegrpc.New(ctx, opts...)
}

func buildOTLPHTTP(ctx context.Context, cfg *config.TracingConfig) (*otlptrace.Exporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}
	return otlptracehttp.New(ctx, opts...)
}
