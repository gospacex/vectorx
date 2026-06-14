package exporter

import (
	"github.com/gospacex/vectorx/config"
)

// buildRedisExporter returns a SpanPublisher-backed exporter for redis streams.
// The publisher must be set via SetRedisPublisher before this is called.
func buildRedisExporter(cfg *config.TracingConfig) (*streamExporter, error) {
	if redisPublisher == nil {
		return nil, errRedisPublisherNotSet
	}
	return &streamExporter{stream: cfg.Stream, pub: redisPublisher, kind: kindRedis}, nil
}

var errRedisPublisherNotSet = &publisherError{exporter: "redis"}
var errKafkaPublisherNotSet = &publisherError{exporter: "kafka"}

type publisherError struct{ exporter string }

func (e *publisherError) Error() string {
	return e.exporter + " exporter requires Set" + e.exporter + "Publisher to be called first"
}
