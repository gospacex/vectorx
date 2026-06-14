//go:build integration
// +build integration

package milvusx_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/milvusx"
	"github.com/gospacex/vectorx/observability"
	"github.com/gospacex/vectorx/observability/exporter"
)

func TestMilvusE2E_OTLP(t *testing.T) {
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

	milvusx.SetConfigPath("mq.yaml")
	c, err := milvusx.GetMilvus("primary")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	has, err := c.HasCollection(ctx, "vectorx_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("has collection: %v (trace recorded via OTLP)", has)
}

func TestMilvusE2E_RedisStream(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	var spansPublished int
	exporter.SetRedisPublisher(&testPublisher{fn: func(dest string, payload []byte) {
		spansPublished++
		t.Logf("redis stream %s: span %d", dest, spansPublished)
	}})
	defer func() { exporter.SetRedisPublisher(nil) }()

	cfg, _ := config.Load("mq.yaml")
	cfg.VectorX.Trace.Exporter = "redis"
	cfg.VectorX.Trace.Stream = "tracing.span"
	observability.InitTracing(&cfg.VectorX.Trace)

	milvusx.SetConfigPath("mq.yaml")
	c, _ := milvusx.GetMilvus("primary")
	defer c.Close()

	ctx := context.Background()
	c.HasCollection(ctx, "vectorx_test")
	t.Logf("redis stream test done, spans published: %d", spansPublished)
}

func TestMilvusE2E_KafkaTopic(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	var spansPublished int
	exporter.SetKafkaPublisher(&testPublisher{fn: func(dest string, payload []byte) {
		spansPublished++
		t.Logf("kafka topic %s: span %d", dest, spansPublished)
	}})
	defer func() { exporter.SetKafkaPublisher(nil) }()

	cfg, _ := config.Load("mq.yaml")
	cfg.VectorX.Trace.Exporter = "kafka"
	cfg.VectorX.Trace.Topic = "tracing.span"
	observability.InitTracing(&cfg.VectorX.Trace)

	milvusx.SetConfigPath("mq.yaml")
	c, _ := milvusx.GetMilvus("primary")
	defer c.Close()

	ctx := context.Background()
	c.HasCollection(ctx, "vectorx_test")
	t.Logf("kafka topic test done, spans published: %d", spansPublished)
}

type testPublisher struct {
	fn func(dest string, payload []byte)
}

func (p *testPublisher) PublishSpan(ctx context.Context, dest string, payload []byte) error {
	p.fn(dest, payload)
	return nil
}

func Example_milvusx_otlp() {
	cfg, err := config.Load("mq.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := observability.InitTracing(&cfg.VectorX.Trace); err != nil {
		log.Fatal(err)
	}

	milvusx.SetConfigPath("mq.yaml")
	c, err := milvusx.GetMilvus("primary")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	has, _ := c.HasCollection(ctx, "example_collection")
	fmt.Printf("has collection: %v (trace → OTLP)\n", has)
}

func Example_milvusx_redis() {
	exporter.SetRedisPublisher(&testPublisher{fn: func(dest string, payload []byte) {
		fmt.Printf("redis stream %s: %s\n", dest, string(payload))
	}})
	defer func() { exporter.SetRedisPublisher(nil) }()

	cfg, _ := config.Load("mq.yaml")
	cfg.VectorX.Trace.Exporter = "redis"
	cfg.VectorX.Trace.Stream = "tracing.span"
	if err := observability.InitTracing(&cfg.VectorX.Trace); err != nil {
		log.Fatal(err)
	}

	milvusx.SetConfigPath("mq.yaml")
	c, _ := milvusx.GetMilvus("primary")
	defer c.Close()

	ctx := context.Background()
	c.HasCollection(ctx, "example_collection")
	fmt.Println("trace exported via redis stream")
}

func Example_milvusx_kafka() {
	exporter.SetKafkaPublisher(&testPublisher{fn: func(dest string, payload []byte) {
		fmt.Printf("kafka topic %s: %s\n", dest, string(payload))
	}})
	defer func() { exporter.SetKafkaPublisher(nil) }()

	cfg, _ := config.Load("mq.yaml")
	cfg.VectorX.Trace.Exporter = "kafka"
	cfg.VectorX.Trace.Topic = "tracing.span"
	if err := observability.InitTracing(&cfg.VectorX.Trace); err != nil {
		log.Fatal(err)
	}

	milvusx.SetConfigPath("mq.yaml")
	c, _ := milvusx.GetMilvus("primary")
	defer c.Close()

	ctx := context.Background()
	c.HasCollection(ctx, "example_collection")
	fmt.Println("trace exported via kafka topic")
}
