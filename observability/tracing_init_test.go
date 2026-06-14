package observability

import (
	"context"
	"sync"
	"testing"

	"github.com/gospacex/vectorx/config"
)

func TestInitTracing_NilConfig_NoOp(t *testing.T) {
	initOnce = sync.Once{}
	enabled = false
	if err := InitTracing(nil); err != nil {
		t.Fatal(err)
	}
}

func TestInitTracing_Disabled_NoOp(t *testing.T) {
	initOnce = sync.Once{}
	enabled = false
	cfg := &config.TracingConfig{Enabled: false}
	if err := InitTracing(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestStartSpan_AfterInit(t *testing.T) {
	initOnce = sync.Once{}
	enabled = false
	cfg := &config.TracingConfig{
		Enabled:     true,
		Exporter:    "otlp",
		Endpoint:    "localhost:4317",
		ServiceName: "test",
	}
	// InitTracing will try to connect to localhost:4317
	// We just verify it handles the connection error gracefully
	_ = InitTracing(cfg)
	// even if InitTracing failed, StartSpan should work as noop
	_, span := StartSpan(context.TODO(), "test")
	if span == nil {
		t.Fatal("expected non-nil span even if init failed")
	}
	span.End()
}

func TestGetGlobalPropagator_NotNil(t *testing.T) {
	p := GetGlobalPropagator()
	if p == nil {
		t.Fatal("expected non-nil propagator")
	}
}
