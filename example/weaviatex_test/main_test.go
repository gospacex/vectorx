//go:build integration
// +build integration

package weaviatex_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/observability"
	"github.com/gospacex/vectorx/observability/exporter"
	"github.com/gospacex/vectorx/weaviatex"
)

func TestWeaviateE2E_OTLP(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	cfg, err := config.Load("mq.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.VectorX.Trace.Exporter = "otlp"
	if err := observability.InitTracing(&cfg.VectorX.Trace); err != nil {
		t.Fatal(err)
	}

	weaviatex.SetConfigPath("mq.yaml")
	_, err = weaviatex.GetWeaviate("primary")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("weaviate client ready (trace via OTLP)")
}

func TestWeaviateE2E_RedisStream(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	var spansPublished int
	exporter.SetRedisPublisher(&publisher{fn: func(dest string, payload []byte) {
		spansPublished++
		t.Logf("redis stream %s: span %d", dest, spansPublished)
	}})
	defer func() { exporter.SetRedisPublisher(nil) }()

	cfg, _ := config.Load("mq.yaml")
	cfg.VectorX.Trace.Exporter = "redis"
	cfg.VectorX.Trace.Stream = "tracing.span"
	observability.InitTracing(&cfg.VectorX.Trace)

	weaviatex.SetConfigPath("mq.yaml")
	c, _ := weaviatex.GetWeaviate("primary")

	_, err := c.IsLive(context.Background())
	t.Logf("redis stream test done, spans published: %d (live err: %v)", spansPublished, err)
}

func TestWeaviateE2E_KafkaTopic(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	var spansPublished int
	exporter.SetKafkaPublisher(&publisher{fn: func(dest string, payload []byte) {
		spansPublished++
		t.Logf("kafka topic %s: span %d", dest, spansPublished)
	}})
	defer func() { exporter.SetKafkaPublisher(nil) }()

	cfg, _ := config.Load("mq.yaml")
	cfg.VectorX.Trace.Exporter = "kafka"
	cfg.VectorX.Trace.Topic = "tracing.span"
	observability.InitTracing(&cfg.VectorX.Trace)

	weaviatex.SetConfigPath("mq.yaml")
	c, _ := weaviatex.GetWeaviate("primary")

	_, err := c.IsLive(context.Background())
	t.Logf("kafka topic test done, spans published: %d (live err: %v)", spansPublished, err)
}

type publisher struct {
	fn func(dest string, payload []byte)
}

func (p *publisher) PublishSpan(ctx context.Context, dest string, payload []byte) error {
	p.fn(dest, payload)
	return nil
}

func Example_weaviatex_otlp() {
	cfg, err := config.Load("mq.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := observability.InitTracing(&cfg.VectorX.Trace); err != nil {
		log.Fatal(err)
	}

	weaviatex.SetConfigPath("mq.yaml")
	_, err = weaviatex.GetWeaviate("primary")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("weaviate client ready (trace → OTLP)\n")
}

func Example_weaviatex_redis() {
	exporter.SetRedisPublisher(&publisher{fn: func(dest string, payload []byte) {
		fmt.Printf("redis stream %s: %s\n", dest, string(payload))
	}})
	defer func() { exporter.SetRedisPublisher(nil) }()

	cfg, _ := config.Load("mq.yaml")
	cfg.VectorX.Trace.Exporter = "redis"
	cfg.VectorX.Trace.Stream = "tracing.span"
	if err := observability.InitTracing(&cfg.VectorX.Trace); err != nil {
		log.Fatal(err)
	}

	weaviatex.SetConfigPath("mq.yaml")
	c, _ := weaviatex.GetWeaviate("primary")

	c.IsLive(context.Background())
	fmt.Println("trace exported via redis stream")
}

func Example_weaviatex_kafka() {
	exporter.SetKafkaPublisher(&publisher{fn: func(dest string, payload []byte) {
		fmt.Printf("kafka topic %s: %s\n", dest, string(payload))
	}})
	defer func() { exporter.SetKafkaPublisher(nil) }()

	cfg, _ := config.Load("mq.yaml")
	cfg.VectorX.Trace.Exporter = "kafka"
	cfg.VectorX.Trace.Topic = "tracing.span"
	if err := observability.InitTracing(&cfg.VectorX.Trace); err != nil {
		log.Fatal(err)
	}

	weaviatex.SetConfigPath("mq.yaml")
	c, _ := weaviatex.GetWeaviate("primary")

	c.IsLive(context.Background())
	fmt.Println("trace exported via kafka topic")
}
