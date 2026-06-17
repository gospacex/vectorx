package config

import "testing"

// mqx.TracingConfig.Validate mutates the receiver in place: it normalizes
// the exporter/protocol/sampler-type strings, applies defaults for
// ServiceName/Endpoint/Stream/Topic, and clamps SamplerRatio. It returns
// no error, so the assertions below read post-validation fields.

func TestValidate_OTLPGRPC_NormalizesProtocol(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "otlp", Protocol: "grpc"}
	c.Validate()
	if c.Protocol != "grpc" {
		t.Fatalf("Protocol = %q, want grpc", c.Protocol)
	}
}

func TestValidate_OTLPHTTP_NormalizesProtocol(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "otlp", Protocol: "http"}
	c.Validate()
	if c.Protocol != "http" {
		t.Fatalf("Protocol = %q, want http", c.Protocol)
	}
}

func TestValidate_Redis_PreservesStream(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "redis", Stream: "tracing.span"}
	c.Validate()
	if c.Stream != "tracing.span" {
		t.Fatalf("Stream = %q, want tracing.span", c.Stream)
	}
}

func TestValidate_Kafka_PreservesTopic(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "kafka", Topic: "tracing.span"}
	c.Validate()
	if c.Topic != "tracing.span" {
		t.Fatalf("Topic = %q, want tracing.span", c.Topic)
	}
}

func TestValidate_UnknownExporter_FallsBackToMQXDefault(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "unknown"}
	c.Validate()
	// mqx normalizes unknown exporters to its own default ("jaeger").
	// This is documented behavior; vectorx accepts "jaeger" as an alias
	// for the OTLP gRPC exporter so the resulting config is still usable.
	if c.Exporter != "jaeger" {
		t.Fatalf("Exporter = %q, want mqx fallback %q", c.Exporter, "jaeger")
	}
}

func TestValidate_Disabled_NoValidation(t *testing.T) {
	c := TracingConfig{Enabled: false, Exporter: ""}
	c.Validate()
	if c.Exporter != "" {
		t.Fatalf("Disabled config should not be normalized, got Exporter=%q", c.Exporter)
	}
}

func TestValidate_CustomEndpointPreserved(t *testing.T) {
	c := TracingConfig{
		Enabled:  true,
		Exporter: "otlp",
		Protocol: "grpc",
		Endpoint: "my-collector:4317",
	}
	c.Validate()
	if c.Endpoint != "my-collector:4317" {
		t.Fatalf("custom endpoint was overwritten: %q", c.Endpoint)
	}
}

func TestValidate_DefaultServiceName(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "otlp", Protocol: "grpc"}
	c.Validate()
	if c.ServiceName != "dbx" {
		t.Fatalf("ServiceName = %q, want mqx default %q", c.ServiceName, "dbx")
	}
}

func TestValidate_ClampRatioAboveOne(t *testing.T) {
	c := TracingConfig{
		Enabled:      true,
		Exporter:     "otlp",
		Protocol:     "grpc",
		SamplerType:  "always_on",
		SamplerRatio: 5.0,
	}
	c.Validate()
	if c.SamplerRatio != 1.0 {
		t.Fatalf("SamplerRatio = %v, want 1.0 (clamped)", c.SamplerRatio)
	}
}

func TestValidate_ClampRatioBelowZero(t *testing.T) {
	c := TracingConfig{
		Enabled:      true,
		Exporter:     "otlp",
		Protocol:     "grpc",
		SamplerType:  "always_on",
		SamplerRatio: -0.5,
	}
	c.Validate()
	if c.SamplerRatio != 0.0 {
		t.Fatalf("SamplerRatio = %v, want 0.0 (clamped)", c.SamplerRatio)
	}
}
