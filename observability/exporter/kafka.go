package exporter

import "github.com/gospacex/vectorx/config"

const (
	kindRedis = "redis"
	kindKafka = "kafka"
)

// buildKafkaExporter returns a SpanPublisher-backed exporter for kafka topics.
func buildKafkaExporter(cfg *config.TracingConfig) (*streamExporter, error) {
	if kafkaPublisher == nil {
		return nil, errKafkaPublisherNotSet
	}
	return &streamExporter{topic: cfg.Topic, pub: kafkaPublisher, kind: kindKafka}, nil
}
