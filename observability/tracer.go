package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace/noop"
)

const TracerName = "github.com/gospacex/vectorx/observability"

var noopTracer = noop.NewTracerProvider().Tracer(TracerName)

// GetGlobalPropagator returns the W3C TraceContext + Baggage propagator.
func GetGlobalPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

// SetGlobalPropagator wires W3C TraceContext + Baggage into OTel global.
func SetGlobalPropagator() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}
