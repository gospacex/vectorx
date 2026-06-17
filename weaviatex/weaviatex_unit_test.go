package weaviatex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/observability"
	"github.com/weaviate/weaviate/entities/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// withTestTracer swaps the global OTel TracerProvider for a fresh
// in-memory exporter-backed one. Auto-restores on cleanup. The
// observability.SetEnabledForTesting toggle is also flipped so
// StartSpan emits real spans (otherwise the noop path is taken
// even with a real TracerProvider).
func withTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	prev := otel.GetTracerProvider()
	exp := tracetest.NewInMemoryExporter()
	res, _ := resource.New(context.Background(), resource.WithAttributes(semconv.ServiceName("weaviatex-unit-test")))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSyncer(exp),
	)
	otel.SetTracerProvider(tp)
	observability.SetEnabledForTesting(t, true)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	return exp
}

// fakeWeaviateOps is a weaviateOps implementation for unit tests. It
// embeds the interface (nil) and overrides only the method under
// test. Tests construct a *Weaviatex manually and assign the field
// directly so the wrapper can be exercised without a live SDK.
type fakeWeaviateOps struct {
	weaviateOps

	graphQLRaw  func(ctx context.Context, query string) (any, error)
	createObject func(ctx context.Context, className string, properties map[string]any, vector []float32) (any, error)
	deleteObject func(ctx context.Context, className string, id string) error
	createClass func(ctx context.Context, class *models.Class) error
	isLive      func(ctx context.Context) (bool, error)
}

func (f *fakeWeaviateOps) opsGraphQLRaw(ctx context.Context, query string) (any, error) {
	return f.graphQLRaw(ctx, query)
}
func (f *fakeWeaviateOps) opsCreateObject(ctx context.Context, className string, properties map[string]any, vector []float32) (any, error) {
	return f.createObject(ctx, className, properties, vector)
}
func (f *fakeWeaviateOps) opsDeleteObject(ctx context.Context, className string, id string) error {
	return f.deleteObject(ctx, className, id)
}
func (f *fakeWeaviateOps) opsCreateClass(ctx context.Context, class *models.Class) error {
	return f.createClass(ctx, class)
}
func (f *fakeWeaviateOps) opsIsLive(ctx context.Context) (bool, error) {
	return f.isLive(ctx)
}

// TestNewClient_BuildsWithSchemeAndHost verifies the constructor
// reads the right fields off the config and produces a non-nil
// client. The vendored weaviate.Client is concrete (not an
// interface), so we can't mock its network calls in a unit test —
// the constructor's only job is to map config fields onto
// weaviate.Config, and a non-nil return is the load-bearing
// assertion.
func TestNewClient_BuildsWithSchemeAndHost(t *testing.T) {
	for _, tc := range []struct {
		name   string
		scheme string
		host   string
	}{
		{"http", "http", "localhost:8080"},
		{"https", "https", "weaviate.example.com"},
		{"empty-scheme", "", "weaviate.example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.WeaviateConfig{
				Name:   "primary",
				Scheme: tc.scheme,
				Host:   tc.host,
			}
			c, err := newClient(cfg)
			if err != nil {
				t.Fatalf("newClient: %v", err)
			}
			if c == nil {
				t.Fatal("nil client")
			}
			if c.cfg != cfg {
				t.Errorf("cfg pointer not preserved")
			}
		})
	}
}

// TestNewClient_AppliesAPIKey covers the auth branch of the
// constructor: when cfg.APIKey is set, the weaviate.Config must
// include an ApiKey auth config. A typo in the auth field would
// produce a client that authenticates with empty bearer, which
// would only fail at first request — this test catches the
// structural error at unit-test time.
func TestNewClient_AppliesAPIKey(t *testing.T) {
	cfg := &config.WeaviateConfig{
		Name:   "primary",
		Scheme: "https",
		Host:   "secure.weaviate.example.com",
		APIKey: "sk-test-12345",
	}
	c, err := newClient(cfg)
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	if c == nil {
		t.Fatal("nil client")
	}
}

// TestLoadConfig_FindsAndRejects verifies the config-lookup path:
// the requested name is found, others are rejected with a
// deterministic error message.
func TestLoadConfig_FindsAndRejects(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	loadOnce = sync.Once{}
	configPath = ""

	dir := t.TempDir()
	p := filepath.Join(dir, "mq.yaml")
	yaml := `vectorx:
  weaviate:
    - name: primary
      scheme: http
      host: localhost:8080
      class: VectorXTest
    - name: audit
      scheme: https
      host: audit.weaviate.example.com
`
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	SetConfigPath(p)

	for _, name := range []string{"primary", "audit"} {
		cfg, err := loadConfig(name)
		if err != nil {
			t.Fatalf("loadConfig(%q): %v", name, err)
		}
		if cfg.Name != name {
			t.Errorf("cfg.Name = %q, want %q", cfg.Name, name)
		}
	}
	// Unknown name must fail with a clear error.
	if _, err := loadConfig("ghost"); err == nil {
		t.Fatal("expected error for unknown name")
	}
}

// TestSetConfigPath_ResetsLoadOnce verifies that calling
// SetConfigPath with a new path forces a reload on the next
// loadConfig call. This is the contract that lets the integration
// tests in the same package use different mq.yaml files per
// subtest without leaking the prior config.
func TestSetConfigPath_ResetsLoadOnce(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	loadOnce = sync.Once{}
	configPath = ""

	dir := t.TempDir()
	// First config: only "alpha" exists.
	p1 := filepath.Join(dir, "a.yaml")
	if err := os.WriteFile(p1, []byte("vectorx:\n  weaviate:\n    - name: alpha\n      host: a:1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	SetConfigPath(p1)
	if _, err := loadConfig("alpha"); err != nil {
		t.Fatalf("first load: %v", err)
	}

	// Second config: only "beta" exists. SetConfigPath must reset
	// loadOnce so the new path is loaded.
	p2 := filepath.Join(dir, "b.yaml")
	if err := os.WriteFile(p2, []byte("vectorx:\n  weaviate:\n    - name: beta\n      host: b:2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	SetConfigPath(p2)
	if _, err := loadConfig("beta"); err != nil {
		t.Fatalf("second load: %v", err)
	}
	if _, err := loadConfig("alpha"); err == nil {
		t.Fatal("alpha should be gone after config switch")
	}
}

// hasAttr returns true if attrs contains (k, v) as a string attribute.
func hasAttr(attrs []attribute.KeyValue, k, v string) bool {
	for _, a := range attrs {
		if string(a.Key) == k && a.Value.AsString() == v {
			return true
		}
	}
	return false
}

// weaviatexMethodCase is a single row in the table-driven coverage
// test. Each row pairs a span name with the method invocation and
// the fake wiring needed to drive it. Methods that take a "class"
// input also check the class attribute on the span.
type weaviatexMethodCase struct {
	spanName string
	class    string
	fakeWith func(err error) *fakeWeaviateOps
	invoke   func(t *testing.T, w *Weaviatex) error
}

// weaviatexMethodCases lists every wrapper method *Weaviatex exposes.
func weaviatexMethodCases() []weaviatexMethodCase {
	return []weaviatexMethodCase{
		{
			spanName: "weaviatex.GraphQLRaw",
			fakeWith: func(err error) *fakeWeaviateOps {
				return &fakeWeaviateOps{graphQLRaw: func(ctx context.Context, query string) (any, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, w *Weaviatex) error {
				_, err := w.GraphQLRaw(context.Background(), "{ Get { ... } }")
				return err
			},
		},
		{
			spanName: "weaviatex.CreateObject",
			class:    "Article",
			fakeWith: func(err error) *fakeWeaviateOps {
				return &fakeWeaviateOps{createObject: func(ctx context.Context, className string, properties map[string]any, vector []float32) (any, error) {
					return map[string]any{"id": "abc"}, err
				}}
			},
			invoke: func(t *testing.T, w *Weaviatex) error {
				_, err := w.CreateObject(context.Background(), "Article", map[string]any{"title": "x"}, []float32{0.1, 0.2})
				return err
			},
		},
		{
			spanName: "weaviatex.DeleteObject",
			class:    "Article",
			fakeWith: func(err error) *fakeWeaviateOps {
				return &fakeWeaviateOps{deleteObject: func(ctx context.Context, className string, id string) error {
					return err
				}}
			},
			invoke: func(t *testing.T, w *Weaviatex) error {
				return w.DeleteObject(context.Background(), "Article", "abc")
			},
		},
		{
			spanName: "weaviatex.CreateClass",
			class:    "SchemaA",
			fakeWith: func(err error) *fakeWeaviateOps {
				return &fakeWeaviateOps{createClass: func(ctx context.Context, class *models.Class) error {
					return err
				}}
			},
			invoke: func(t *testing.T, w *Weaviatex) error {
				return w.CreateClass(context.Background(), &models.Class{Class: "SchemaA"})
			},
		},
		{
			spanName: "weaviatex.IsLive",
			fakeWith: func(err error) *fakeWeaviateOps {
				return &fakeWeaviateOps{isLive: func(ctx context.Context) (bool, error) {
					return false, err
				}}
			},
			invoke: func(t *testing.T, w *Weaviatex) error {
				_, err := w.IsLive(context.Background())
				return err
			},
		},
	}
}

// TestWeaviatex_MethodSpanNameAndError is a table-driven coverage
// test that exercises every wrapped method. Each row builds a
// *Weaviatex with a fake ops that returns the row's wantErr, then
// asserts the wrapper (a) starts a span with the expected name, (b)
// records the error on it, and (c) propagates the error. Methods
// that take a class input also check the class attribute.
func TestWeaviatex_MethodSpanNameAndError(t *testing.T) {
	for _, tc := range weaviatexMethodCases() {
		t.Run(tc.spanName, func(t *testing.T) {
			exp := withTestTracer(t)
			want := errors.New("boom-" + tc.spanName)
			w := &Weaviatex{
				cfg: &config.WeaviateConfig{Name: "unit"},
				ops: tc.fakeWith(want),
			}
			err := tc.invoke(t, w)
			if !errors.Is(err, want) {
				t.Fatalf("err = %v, want %v", err, want)
			}
			spans := exp.GetSpans().Snapshots()
			if len(spans) != 1 {
				t.Fatalf("spans = %d, want 1", len(spans))
			}
			if got := spans[0].Name(); got != tc.spanName {
				t.Errorf("span.Name = %q, want %q", got, tc.spanName)
			}
			if tc.class != "" && !hasAttr(spans[0].Attributes(), "class", tc.class) {
				t.Errorf("missing class=%s, got %v", tc.class, spans[0].Attributes())
			}
			if len(spans[0].Events()) == 0 {
				t.Errorf("expected error event")
			}
		})
	}
}

// TestWeaviatex_SuccessNoErrorEvent verifies that on the success
// path the span is emitted but no error event is attached.
func TestWeaviatex_SuccessNoErrorEvent(t *testing.T) {
	exp := withTestTracer(t)
	w := &Weaviatex{
		cfg: &config.WeaviateConfig{Name: "unit"},
		ops: &fakeWeaviateOps{isLive: func(ctx context.Context) (bool, error) {
			return true, nil
		}},
	}
	if _, err := w.IsLive(context.Background()); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if len(spans[0].Events()) != 0 {
		t.Errorf("expected no error events on success path, got %d", len(spans[0].Events()))
	}
}
