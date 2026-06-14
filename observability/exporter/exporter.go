package exporter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gospacex/vectorx/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SpanPublisher is the seam for publishing span payloads to non-OTLP backends.
// Implementations live in application code; observability must not import the
// underlying client SDKs (redis / kafka) so a static dependency check passes.
type SpanPublisher interface {
	// PublishSpan is called with the destination (stream name for redis, topic
	// for kafka) and the JSON-encoded span record.
	PublishSpan(ctx context.Context, destination string, payload []byte) error
}

var (
	redisPublisher SpanPublisher
	kafkaPublisher SpanPublisher
)

// SetRedisPublisher injects the publisher used by the "redis" exporter.
// Call this once at application startup, before InitTracing.
func SetRedisPublisher(p SpanPublisher) { redisPublisher = p }

// SetKafkaPublisher injects the publisher used by the "kafka" exporter.
func SetKafkaPublisher(p SpanPublisher) { kafkaPublisher = p }

// Build selects and constructs the exporter per cfg.Exporter.
// For "otlp" returns an *otlptrace.Exporter.
// For "redis" / "kafka" returns a stream-backed exporter implementing
// sdktrace.SpanExporter, ready to wire into a TracerProvider.
func Build(cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	ctx := context.Background()
	switch cfg.Exporter {
	case "otlp":
		return buildOTLP(ctx, cfg)
	case "redis":
		return buildRedisExporter(cfg)
	case "kafka":
		return buildKafkaExporter(cfg)
	}
	return nil, fmt.Errorf("unknown exporter %q", cfg.Exporter)
}

// spanRecord is the wire format published to redis/kafka backends.
type spanRecord struct {
	TraceID  string `json:"trace_id"`
	SpanID   string `json:"span_id"`
	Name     string `json:"name"`
	StartNS  int64  `json:"start_ns"`
	Duration int64  `json:"duration_ns"`
}

type streamExporter struct {
	stream string
	topic  string
	pub    SpanPublisher
	kind   string
}

// ExportSpans implements sdktrace.SpanExporter by adapting the OTel ReadOnlySpan
// list into the local spanRecord type and publishing each via the seam.
func (e *streamExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, sp := range spans {
		rec := spanRecord{
			TraceID:  sp.SpanContext().TraceID().String(),
			SpanID:   sp.SpanContext().SpanID().String(),
			Name:     sp.Name(),
			StartNS:  sp.StartTime().UnixNano(),
			Duration: sp.EndTime().UnixNano() - sp.StartTime().UnixNano(),
		}
		payload, _ := json.Marshal(rec)
		dest := e.stream
		if e.kind == kindKafka {
			dest = e.topic
		}
		if err := e.pub.PublishSpan(ctx, dest, payload); err != nil {
			return err
		}
	}
	return nil
}

func (e *streamExporter) Shutdown(_ context.Context) error { return nil }
