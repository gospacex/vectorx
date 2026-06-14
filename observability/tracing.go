package observability

import (
	"context"
	"sync"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/observability/exporter"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	initOnce sync.Once
	enabled  bool
)

// InitTracing initializes the global OTel TracerProvider per cfg.
// Calling InitTracing more than once is a no-op.
//
// For the "redis" and "kafka" exporters, a SpanPublisher must be injected
// via exporter.SetRedisPublisher / exporter.SetKafkaPublisher before this call.
func InitTracing(cfg *config.TracingConfig) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	var initErr error
	initOnce.Do(func() {
		SetGlobalPropagator()
		exp, err := exporter.Build(cfg)
		if err != nil {
			initErr = err
			return
		}
		res, _ := resource.New(context.Background(), resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		))
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(samplerFromConfig(cfg)),
		)
		otel.SetTracerProvider(tp)
		enabled = true
	})
	return initErr
}

func samplerFromConfig(cfg *config.TracingConfig) sdktrace.Sampler {
	if cfg.SamplerType == "" {
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplerRatio))
	}
	switch cfg.SamplerType {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplerRatio))
	}
}

// StartSpan returns a span scoped to name. When tracing is disabled, returns
// a non-recording span on the original context.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if !enabled {
		return noopTracer.Start(ctx, name)
	}
	ctx, span := otel.Tracer(TracerName).Start(ctx, name, trace.WithAttributes(attrs...))
	return ctx, span
}
