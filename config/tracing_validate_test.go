package config

import "testing"

func TestValidate_DefaultEndpointForOTLPGRPC(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "otlp", Protocol: "grpc"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Endpoint != "localhost:4317" {
		t.Fatalf("default endpoint = %q", c.Endpoint)
	}
}

func TestValidate_DefaultEndpointForOTLPHTTP(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "otlp", Protocol: "http"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Endpoint != "localhost:4318" {
		t.Fatalf("default endpoint = %q", c.Endpoint)
	}
}

func TestValidate_DefaultEndpointForRedis(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "redis"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Endpoint != "localhost:6379" {
		t.Fatalf("default endpoint = %q", c.Endpoint)
	}
	if c.Stream != "tracing.span" {
		t.Fatalf("default stream = %q", c.Stream)
	}
}

func TestValidate_DefaultEndpointForKafka(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "kafka"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Endpoint != "localhost:9092" {
		t.Fatalf("default endpoint = %q", c.Endpoint)
	}
	if c.Topic != "tracing.span" {
		t.Fatalf("default topic = %q", c.Topic)
	}
}

func TestValidate_UnknownExporter(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "unknown"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for unknown exporter")
	}
}

func TestValidate_Disabled_NoValidation(t *testing.T) {
	c := TracingConfig{Enabled: false, Exporter: ""}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_CustomEndpointPreserved(t *testing.T) {
	c := TracingConfig{Enabled: true, Exporter: "otlp", Endpoint: "my-collector:4317"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Endpoint != "my-collector:4317" {
		t.Fatalf("custom endpoint was overwritten: %q", c.Endpoint)
	}
}
