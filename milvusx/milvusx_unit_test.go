package milvusx

import (
	"context"
	"errors"
	"testing"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/observability"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// fakeMilvusClient embeds the milvus client.Client interface (nil) and
// overrides only the methods the wrapped adapter methods call. This
// avoids stubbing the entire ~80-method client.Client surface.
type fakeMilvusClient struct {
	client.Client

	hasCollection    func(ctx context.Context, collName string) (bool, error)
	insert           func(ctx context.Context, collName string, partitionName string, columns ...entity.Column) (entity.Column, error)
	flush            func(ctx context.Context, collName string, async bool, opts ...client.FlushOption) error
	createCollection func(ctx context.Context, schema *entity.Schema, shardsNum int32, opts ...client.CreateCollectionOption) error
	dropCollection   func(ctx context.Context, collName string, opts ...client.DropCollectionOption) error
	describe         func(ctx context.Context, collName string) (*entity.Collection, error)
}

func (f *fakeMilvusClient) HasCollection(ctx context.Context, collName string) (bool, error) {
	return f.hasCollection(ctx, collName)
}
func (f *fakeMilvusClient) Insert(ctx context.Context, collName string, partitionName string, columns ...entity.Column) (entity.Column, error) {
	return f.insert(ctx, collName, partitionName, columns...)
}
func (f *fakeMilvusClient) Flush(ctx context.Context, collName string, async bool, opts ...client.FlushOption) error {
	return f.flush(ctx, collName, async, opts...)
}
func (f *fakeMilvusClient) CreateCollection(ctx context.Context, schema *entity.Schema, shardsNum int32, opts ...client.CreateCollectionOption) error {
	return f.createCollection(ctx, schema, shardsNum, opts...)
}
func (f *fakeMilvusClient) DropCollection(ctx context.Context, collName string, opts ...client.DropCollectionOption) error {
	return f.dropCollection(ctx, collName, opts...)
}
func (f *fakeMilvusClient) DescribeCollection(ctx context.Context, collName string) (*entity.Collection, error) {
	return f.describe(ctx, collName)
}

func withTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	prev := otel.GetTracerProvider()
	exp := tracetest.NewInMemoryExporter()
	res, _ := resource.New(context.Background(), resource.WithAttributes(semconv.ServiceName("milvusx-unit-test")))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSyncer(exp),
	)
	otel.SetTracerProvider(tp)
	observability.SetEnabledForTesting(t, true)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	return exp
}

// TestMilvusx_HasCollection_SpanAndErrorRecording verifies that the
// HasCollection adapter path starts a span named "milvusx.HasCollection",
// carries the collection name attribute, and records the underlying
// error on the span.
func TestMilvusx_HasCollection_SpanAndErrorRecording(t *testing.T) {
	exp := withTestTracer(t)
	wantErr := errors.New("milvus down")
	m := &Milvusx{
		cfg: &config.MilvusConfig{Name: "unit", Address: "localhost:19530"},
		Client: &fakeMilvusClient{
			hasCollection: func(ctx context.Context, collName string) (bool, error) {
				return false, wantErr
			},
		},
	}
	_, err := m.HasCollection(context.Background(), "embeddings")
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if got := spans[0].Name(); got != "milvusx.HasCollection" {
		t.Errorf("span.Name = %q, want milvusx.HasCollection", got)
	}
	if !hasAttr(spans[0].Attributes(), "collection", "embeddings") {
		t.Errorf("expected collection=embeddings, got %v", spans[0].Attributes())
	}
	if len(spans[0].Events()) == 0 {
		t.Errorf("expected error event, got none")
	}
}

// TestMilvusx_HasCollection_Success verifies the happy path produces
// no error event on the span. Mirrors the qdrantx success-path test.
func TestMilvusx_HasCollection_Success(t *testing.T) {
	exp := withTestTracer(t)
	m := &Milvusx{
		cfg: &config.MilvusConfig{Name: "unit", Address: "localhost:19530"},
		Client: &fakeMilvusClient{
			hasCollection: func(ctx context.Context, collName string) (bool, error) {
				if collName != "wanted" {
					t.Errorf("fake got collName = %q, want wanted", collName)
				}
				return true, nil
			},
		},
	}
	ok, err := m.HasCollection(context.Background(), "wanted")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v, want true nil", ok, err)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 || len(spans[0].Events()) != 0 {
		t.Errorf("expected 1 span with no events, got %+v", spans)
	}
}

// TestMilvusx_Insert_SpanName verifies the Insert span name and that
// partitionName is passed through to the underlying client. This
// protects the "Insert writes to the right partition" contract.
func TestMilvusx_Insert_SpanName(t *testing.T) {
	exp := withTestTracer(t)
	m := &Milvusx{
		cfg: &config.MilvusConfig{Name: "unit", Address: "localhost:19530"},
		Client: &fakeMilvusClient{
			insert: func(ctx context.Context, collName, partitionName string, columns ...entity.Column) (entity.Column, error) {
				if collName != "docs" {
					t.Errorf("collName = %q, want docs", collName)
				}
				if partitionName != "p1" {
					t.Errorf("partitionName = %q, want p1", partitionName)
				}
				return nil, nil
			},
		},
	}
	if _, err := m.Insert(context.Background(), "docs", "p1"); err != nil {
		t.Fatal(err)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 || spans[0].Name() != "milvusx.Insert" {
		t.Fatalf("spans = %+v", spans)
	}
	if !hasAttr(spans[0].Attributes(), "collection", "docs") {
		t.Errorf("missing collection=docs attribute, got %v", spans[0].Attributes())
	}
}

// TestMilvusx_Flush_PassesAsyncFlag verifies Flush forwards the async
// boolean to the vendor SDK unchanged. The async flag changes the
// semantics of the call (sync vs background) — getting it wrong
// silently would cause the caller to block on flushes they expected
// to be fire-and-forget.
func TestMilvusx_Flush_PassesAsyncFlag(t *testing.T) {
	withTestTracer(t) // register a tracer so spans don't panic
	got := false
	m := &Milvusx{
		cfg: &config.MilvusConfig{Name: "unit", Address: "localhost:19530"},
		Client: &fakeMilvusClient{
			flush: func(ctx context.Context, collName string, async bool, opts ...client.FlushOption) error {
				got = async
				return nil
			},
		},
	}
	if err := m.Flush(context.Background(), "c", true); err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Errorf("async = false, want true (flag not forwarded)")
	}
}

// TestMilvusx_CreateCollection_SchemaName is the regression guard for
// the span attribute being derived from schema.CollectionName (not
// from a separate "name" parameter — there is no separate parameter).
// If a future refactor breaks the attribute extraction, this catches it.
func TestMilvusx_CreateCollection_SchemaName(t *testing.T) {
	exp := withTestTracer(t)
	m := &Milvusx{
		cfg: &config.MilvusConfig{Name: "unit", Address: "localhost:19530"},
		Client: &fakeMilvusClient{
			createCollection: func(ctx context.Context, schema *entity.Schema, shardsNum int32, opts ...client.CreateCollectionOption) error {
				return nil
			},
		},
	}
	sch := &entity.Schema{CollectionName: "alpha"}
	if err := m.CreateCollection(context.Background(), sch, 1); err != nil {
		t.Fatal(err)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 || spans[0].Name() != "milvusx.CreateCollection" {
		t.Fatalf("spans = %+v", spans)
	}
	if !hasAttr(spans[0].Attributes(), "collection", "alpha") {
		t.Errorf("expected collection=alpha, got %v", spans[0].Attributes())
	}
}

// TestMilvusx_DropCollection_SpanName is a smoke test for the simpler
// (collName-only) methods. Covers the common span-name + error-attribute
// shape used by DropCollection / DescribeCollection.
func TestMilvusx_DropCollection_SpanName(t *testing.T) {
	exp := withTestTracer(t)
	wantErr := errors.New("drop refused")
	m := &Milvusx{
		cfg: &config.MilvusConfig{Name: "unit", Address: "localhost:19530"},
		Client: &fakeMilvusClient{
			dropCollection: func(ctx context.Context, collName string, opts ...client.DropCollectionOption) error {
				return wantErr
			},
		},
	}
	if err := m.DropCollection(context.Background(), "doomed"); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 || spans[0].Name() != "milvusx.DropCollection" {
		t.Fatalf("spans = %+v", spans)
	}
	if !hasAttr(spans[0].Attributes(), "collection", "doomed") {
		t.Errorf("missing collection=doomed, got %v", spans[0].Attributes())
	}
	if len(spans[0].Events()) == 0 {
		t.Errorf("expected error event")
	}
}

// TestMilvusx_Search_SpanAndErrorRecording verifies the Search wrapper
// starts a span named "milvusx.Search", carries the collection + topK
// attributes, and records the underlying error. Search is the most
// complex wrapper in the adapter (10 positional args plus options) and
// was previously uncovered — the test guards the long argument list
// against silent signature drift.
func TestMilvusx_Search_SpanAndErrorRecording(t *testing.T) {
	exp := withTestTracer(t)
	wantErr := errors.New("search timed out")
	m := &Milvusx{
		cfg: &config.MilvusConfig{Name: "unit", Address: "localhost:19530"},
		Client: &fakeMilvusClient{
			// Search isn't a method we override on fakeMilvusClient
			// directly; rely on the embedded nil interface path to
			// surface the wanted error... but that would panic. So
			// add a search override on the embedded client instead.
		},
	}
	// Drive the Search wrapper through a fake that returns the wanted
	// error. We have to construct a *fresh* fake that implements
	// Search because the package-level fakeMilvusClient doesn't.
	fake := &searchFake{
		search: func(ctx context.Context, collName string, partitions []string, expr string, outputFields []string, vectors []entity.Vector, vectorField string, metricType entity.MetricType, topK int, sp entity.SearchParam, opts ...client.SearchQueryOptionFunc) ([]client.SearchResult, error) {
			if collName != "embeddings" {
				t.Errorf("collName = %q, want embeddings", collName)
			}
			if topK != 5 {
				t.Errorf("topK = %d, want 5", topK)
			}
			return nil, wantErr
		},
	}
	m.Client = fake
	_, err := m.Search(context.Background(), "embeddings", nil, "id > 0", []string{"id"}, nil, "vec", entity.L2, 5, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 || spans[0].Name() != "milvusx.Search" {
		t.Fatalf("spans = %+v", spans)
	}
	if !hasAttr(spans[0].Attributes(), "collection", "embeddings") {
		t.Errorf("missing collection=embeddings, got %v", spans[0].Attributes())
	}
	if !hasIntAttr(spans[0].Attributes(), "topK", 5) {
		t.Errorf("missing topK=5 (int), got %v", spans[0].Attributes())
	}
	if len(spans[0].Events()) == 0 {
		t.Errorf("expected error event")
	}
}

// hasIntAttr returns true if attrs contains key=k with INT64 value=v.
// OTel's AsString() returns "" for INT64 values (the stringly field
// is only set for STRING), so we have to type-check via the Value's
// numeric representation rather than the string view.
func hasIntAttr(attrs []otelAttribute, k string, v int64) bool {
	for _, a := range attrs {
		if string(a.Key) != k {
			continue
		}
		if a.Value.Type() != attribute.INT64 {
			continue
		}
		return a.Value.AsInt64() == v
	}
	return false
}

// searchFake extends the search behavior of fakeMilvusClient to cover
// the Search method (which the shared fake doesn't override because
// it has the most complex signature in the adapter).
type searchFake struct {
	client.Client
	search func(ctx context.Context, collName string, partitions []string, expr string, outputFields []string, vectors []entity.Vector, vectorField string, metricType entity.MetricType, topK int, sp entity.SearchParam, opts ...client.SearchQueryOptionFunc) ([]client.SearchResult, error)
}

func (s *searchFake) Search(ctx context.Context, collName string, partitions []string, expr string, outputFields []string, vectors []entity.Vector, vectorField string, metricType entity.MetricType, topK int, sp entity.SearchParam, opts ...client.SearchQueryOptionFunc) ([]client.SearchResult, error) {
	return s.search(ctx, collName, partitions, expr, outputFields, vectors, vectorField, metricType, topK, sp, opts...)
}

// TestMilvusx_DescribeCollection_SpanName verifies the
// DescribeCollection wrapper starts a span with the correct name and
// collection attribute, and propagates the underlying error.
func TestMilvusx_DescribeCollection_SpanName(t *testing.T) {
	exp := withTestTracer(t)
	wantErr := errors.New("not found")
	m := &Milvusx{
		cfg: &config.MilvusConfig{Name: "unit", Address: "localhost:19530"},
		Client: &fakeMilvusClient{
			describe: func(ctx context.Context, collName string) (*entity.Collection, error) {
				return nil, wantErr
			},
		},
	}
	if _, err := m.DescribeCollection(context.Background(), "docs"); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 || spans[0].Name() != "milvusx.DescribeCollection" {
		t.Fatalf("spans = %+v", spans)
	}
	if !hasAttr(spans[0].Attributes(), "collection", "docs") {
		t.Errorf("missing collection=docs, got %v", spans[0].Attributes())
	}
	if len(spans[0].Events()) == 0 {
		t.Errorf("expected error event")
	}
}

// hasAttr returns true iff attrs contains a key=k, value=v pair.
func hasAttr(attrs []otelAttribute, k, v string) bool {
	for _, a := range attrs {
		if string(a.Key) == k && a.Value.AsString() == v {
			return true
		}
	}
	return false
}

// otelAttribute is a thin alias for the OTel attribute.KeyValue type
// so the helper above doesn't have to import the OTel attribute
// package — which would otherwise pollute every test's imports.
type otelAttribute = attribute.KeyValue
