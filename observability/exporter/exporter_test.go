package exporter

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gospacex/vectorx/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

func TestStreamExporter_ExportSpans(t *testing.T) {
	var published []struct {
		dest    string
		payload []byte
	}
	pub := &fakePublisher{
		fn: func(ctx context.Context, dest string, payload []byte) error {
			published = append(published, struct {
				dest    string
				payload []byte
			}{dest, payload})
			return nil
		},
	}

	exp := &streamExporter{stream: "tracing.span", pub: pub, kind: kindRedis}
	spans := buildTestSpans(t, 2)
	if err := exp.ExportSpans(context.Background(), spans); err != nil {
		t.Fatal(err)
	}
	if len(published) != 2 {
		t.Fatalf("expected 2 published records, got %d", len(published))
	}
	if published[0].dest != "tracing.span" {
		t.Fatalf("dest = %q", published[0].dest)
	}
}

func TestStreamExporter_KafkaUsesTopic(t *testing.T) {
	var lastDest string
	pub := &fakePublisher{
		fn: func(ctx context.Context, dest string, payload []byte) error {
			lastDest = dest
			return nil
		},
	}

	exp := &streamExporter{topic: "tracing.milvus", pub: pub, kind: kindKafka}
	spans := buildTestSpans(t, 1)
	if err := exp.ExportSpans(context.Background(), spans); err != nil {
		t.Fatal(err)
	}
	if lastDest != "tracing.milvus" {
		t.Fatalf("kafka dest = %q, want tracing.milvus", lastDest)
	}
}

func TestStreamExporter_PublisherError(t *testing.T) {
	want := errors.New("publish failed")
	pub := &fakePublisher{
		fn: func(ctx context.Context, dest string, payload []byte) error {
			return want
		},
	}

	exp := &streamExporter{stream: "s", pub: pub, kind: kindRedis}
	spans := buildTestSpans(t, 1)
	if err := exp.ExportSpans(context.Background(), spans); err != want {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestBuild_OTLP(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:  true,
		Exporter: "otlp",
		Endpoint: "localhost:4317",
	}
	exp, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
	_ = exp.Shutdown(context.Background())
}

func TestBuild_Redis_MissingPublisher(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:  true,
		Exporter: "redis",
	}
	_, err := Build(cfg)
	if err == nil {
		t.Fatal("expected error for missing redis publisher")
	}
}

func TestBuild_Kafka_MissingPublisher(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:  true,
		Exporter: "kafka",
	}
	_, err := Build(cfg)
	if err == nil {
		t.Fatal("expected error for missing kafka publisher")
	}
}

func TestBuild_UnknownExporter(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:  true,
		Exporter: "nonexistent",
	}
	_, err := Build(cfg)
	if err == nil {
		t.Fatal("expected error for unknown exporter")
	}
}

func TestSetRedisPublisher_BuildOK(t *testing.T) {
	SetRedisPublisher(&fakePublisher{fn: func(ctx context.Context, dest string, payload []byte) error { return nil }})
	defer func() { redisPublisher = nil }()

	cfg := &config.TracingConfig{
		Enabled:  true,
		Exporter: "redis",
		Stream:   "tracing.span",
	}
	exp, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_ = exp.Shutdown(context.Background())
}

func TestSetKafkaPublisher_BuildOK(t *testing.T) {
	SetKafkaPublisher(&fakePublisher{fn: func(ctx context.Context, dest string, payload []byte) error { return nil }})
	defer func() { kafkaPublisher = nil }()

	cfg := &config.TracingConfig{
		Enabled:  true,
		Exporter: "kafka",
		Topic:    "tracing.span",
	}
	exp, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_ = exp.Shutdown(context.Background())
}

func TestPublisherError_ErrorMessage(t *testing.T) {
	e := &publisherError{exporter: "redis"}
	want := "redis exporter requires SetredisPublisher to be called first"
	if got := e.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

type fakePublisher struct {
	fn func(ctx context.Context, dest string, payload []byte) error
}

func (f *fakePublisher) PublishSpan(ctx context.Context, dest string, payload []byte) error {
	return f.fn(ctx, dest, payload)
}

func buildTestSpans(t *testing.T, n int) []sdktrace.ReadOnlySpan {
	t.Helper()
	res, _ := resource.New(context.Background(), resource.WithAttributes(
		semconv.ServiceName("test"),
	))
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSyncer(exp),
	)
	tracer := tp.Tracer("test")
	ctx := context.Background()

	var spans []sdktrace.ReadOnlySpan
	for i := 0; i < n; i++ {
		_, s := tracer.Start(ctx, "test-span")
		s.SetAttributes(attribute.Int("i", i))
		s.End()
	}
	spans = exp.GetSpans().Snapshots()
	if len(spans) != n {
		t.Fatalf("got %d spans, want %d", len(spans), n)
	}
	return spans
}

// recordingPublisher captures every PublishSpan call so the end-to-end
// tests below can assert on the actual wire format that stream consumers
// (Redis Stream subscriber, Kafka topic consumer) would see.
type recordingPublisher struct {
	mu   sync.Mutex
	calls []recordingCall
}

type recordingCall struct {
	Dest    string
	Payload []byte
}

func (r *recordingPublisher) PublishSpan(_ context.Context, dest string, payload []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// copy payload so later mutations don't disturb assertions
	buf := make([]byte, len(payload))
	copy(buf, payload)
	r.calls = append(r.calls, recordingCall{Dest: dest, Payload: buf})
	return nil
}

func (r *recordingPublisher) Calls() []recordingCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordingCall, len(r.calls))
	copy(out, r.calls)
	return out
}

// TestExporter_RedisStream_PublishesSpan is the end-to-end coverage for the
// "redis" exporter: inject a recording SpanPublisher, build the exporter
// via Build(), wire it into an SDK TracerProvider with a sync span
// processor (deterministic flush), emit a real span, and assert the
// publisher received a JSON-encoded spanRecord with non-empty
// TraceID/SpanID/Name/StartNS/Duration. The destination must be the
// configured stream name.
func TestExporter_RedisStream_PublishesSpan(t *testing.T) {
	rec := &recordingPublisher{}
	SetRedisPublisher(rec)
	defer func() { redisPublisher = nil }()

	cfg := &config.TracingConfig{
		Enabled:  true,
		Exporter: "redis",
		Stream:   "tracing.vectorx.test",
	}
	exp, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}

	res, _ := resource.New(context.Background(), resource.WithAttributes(
		semconv.ServiceName("redis-stream-test"),
	))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSyncer(exp), // sync processor — no race, no Sleep
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	tracer := tp.Tracer("redis-stream-test")
	_, span := tracer.Start(context.Background(), "milvusx.Search",
		trace.WithAttributes(attribute.String("collection", "vectorx_test")),
	)
	span.End()

	// Sync processor flushes on End; calls is non-empty by now.
	calls := rec.Calls()
	if len(calls) == 0 {
		t.Fatal("expected at least one PublishSpan call, got 0")
	}
	if calls[0].Dest != "tracing.vectorx.test" {
		t.Fatalf("dest = %q, want tracing.vectorx.test", calls[0].Dest)
	}

	var rec0 spanRecord
	if err := json.Unmarshal(calls[0].Payload, &rec0); err != nil {
		t.Fatalf("payload not valid spanRecord JSON: %v\npayload=%s", err, calls[0].Payload)
	}
	if rec0.TraceID == "" || rec0.TraceID == "00000000000000000000000000000000" {
		t.Fatalf("TraceID empty/zero: %q", rec0.TraceID)
	}
	if rec0.SpanID == "" || rec0.SpanID == "0000000000000000" {
		t.Fatalf("SpanID empty/zero: %q", rec0.SpanID)
	}
	if rec0.Name != "milvusx.Search" {
		t.Fatalf("Name = %q, want milvusx.Search", rec0.Name)
	}
	if rec0.StartNS <= 0 {
		t.Fatalf("StartNS = %d, want > 0", rec0.StartNS)
	}
	if rec0.Duration <= 0 {
		t.Fatalf("Duration = %d, want > 0", rec0.Duration)
	}
}

// TestExporter_KafkaTopic_PublishesSpan mirrors the redis test for the
// kafka exporter: the destination passed to PublishSpan must be the
// configured topic (not the stream field), and the JSON payload must
// be a spanRecord.
func TestExporter_KafkaTopic_PublishesSpan(t *testing.T) {
	rec := &recordingPublisher{}
	SetKafkaPublisher(rec)
	defer func() { kafkaPublisher = nil }()

	cfg := &config.TracingConfig{
		Enabled:  true,
		Exporter: "kafka",
		Topic:    "tracing.vectorx.kafka.test",
	}
	exp, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}

	res, _ := resource.New(context.Background(), resource.WithAttributes(
		semconv.ServiceName("kafka-topic-test"),
	))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSyncer(exp),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	tracer := tp.Tracer("kafka-topic-test")
	_, span := tracer.Start(context.Background(), "qdrantx.Search",
		trace.WithAttributes(attribute.String("collection", "vectorx_qdrant")),
	)
	span.End()

	calls := rec.Calls()
	if len(calls) == 0 {
		t.Fatal("expected at least one PublishSpan call, got 0")
	}
	if calls[0].Dest != "tracing.vectorx.kafka.test" {
		t.Fatalf("dest = %q, want tracing.vectorx.kafka.test", calls[0].Dest)
	}

	var rec0 spanRecord
	if err := json.Unmarshal(calls[0].Payload, &rec0); err != nil {
		t.Fatalf("payload not valid spanRecord JSON: %v\npayload=%s", err, calls[0].Payload)
	}
	if rec0.Name != "qdrantx.Search" {
		t.Fatalf("Name = %q, want qdrantx.Search", rec0.Name)
	}
	if rec0.TraceID == "" {
		t.Fatal("TraceID empty")
	}
	if time.Duration(rec0.Duration) <= 0 {
		t.Fatalf("Duration = %d, want > 0", rec0.Duration)
	}
}
