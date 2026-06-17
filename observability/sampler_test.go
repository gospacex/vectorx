package observability

import (
	"context"
	"testing"

	"github.com/gospacex/vectorx/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TestSamplerFromConfig_Matrix exercises every branch of
// samplerFromConfig so a future refactor cannot silently change
// which sampler a given YAML string maps to. The OTel sampler
// families are private types so we only assert on behavior —
// specifically the SamplingResult's Decision for a fresh trace
// context. A wrong mapping shows up immediately as a different
// RecordAndSample / Drop verdict.
func TestSamplerFromConfig_Matrix(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cfg       config.TracingConfig
		shouldKeep bool // expected Decision for a fresh trace (no parent)
	}{
		{
			name: "empty_sampler_type_ratio_one_keeps",
			cfg:  config.TracingConfig{SamplerType: "", SamplerRatio: 1.0},
			shouldKeep: true,
		},
		{
			name: "empty_sampler_type_ratio_zero_drops",
			cfg:  config.TracingConfig{SamplerType: "", SamplerRatio: 0.0},
			shouldKeep: false,
		},
		{
			name: "always_on_keeps_even_with_zero_ratio",
			cfg:  config.TracingConfig{SamplerType: "always_on", SamplerRatio: 0.0},
			shouldKeep: true,
		},
		{
			name: "always_off_drops_even_with_full_ratio",
			cfg:  config.TracingConfig{SamplerType: "always_off", SamplerRatio: 1.0},
			shouldKeep: false,
		},
		{
			name: "unknown_type_falls_back_to_ratio",
			cfg:  config.TracingConfig{SamplerType: "trace_id_ratio", SamplerRatio: 1.0},
			shouldKeep: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.cfg // value copy so the loop var is captured
			s := samplerFromConfig(&c)
			if s == nil {
				t.Fatal("samplerFromConfig returned nil")
			}
			// Pass a fresh context — ParentBased samplers with no
			// parent use the root sampler; here that is the
			// TraceIDRatioBased we configured.
			res := s.ShouldSample(sdktrace.SamplingParameters{
				ParentContext: context.Background(),
				TraceID:       [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				Name:          "test",
				Kind:          trace.SpanKindInternal,
			})
			if (res.Decision == sdktrace.RecordAndSample) != tc.shouldKeep {
				t.Errorf("Decision = %v, want RecordAndSample=%v", res.Decision, tc.shouldKeep)
			}
		})
	}
}

// TestStartSpan_DisabledReturnsSpan asserts StartSpan is safe to
// call when tracing has never been initialized (the noop path). The
// returned span must be non-nil and End()-able — these are the only
// contracts adapter code relies on.
func TestStartSpan_DisabledReturnsSpan(t *testing.T) {
	prev := enabled
	enabled = false
	t.Cleanup(func() { enabled = prev })

	ctx, span := StartSpan(context.Background(), "noop-span")
	if span == nil {
		t.Fatal("expected non-nil span from StartSpan")
	}
	if span.SpanContext().IsSampled() {
		t.Errorf("disabled span should not be sampled")
	}
	span.End()
	if ctx == nil {
		t.Fatal("expected non-nil context from StartSpan")
	}
}