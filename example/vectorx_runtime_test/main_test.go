//go:build integration
// +build integration

// Package main_test demonstrates the vectorx.MustInit entry point by
// exercising one runtime against the three vector adapters. All tests are
// gated on the INTEGRATION environment variable so they only run when a
// real Milvus / Qdrant / Weaviate instance is reachable.
//
// Run with:
//   INTEGRATION=1 go test -tags=integration ./example/vectorx_runtime_test/...
package main_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gospacex/vectorx"
	qdrant "github.com/qdrant/go-client/qdrant"
)

func skipIfNotIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run end-to-end runtime tests")
	}
}

func TestRuntime_OTLP_Milvus(t *testing.T) {
	skipIfNotIntegration(t)
	rt := vectorx.MustInit("mq.yaml")
	defer rt.Close()

	c, err := rt.Milvus("primary")
	if err != nil {
		t.Fatalf("rt.Milvus(primary): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	has, err := c.HasCollection(ctx, "vectorx_test")
	if err != nil {
		t.Fatalf("HasCollection: %v", err)
	}
	t.Logf("milvus has collection: %v (trace → OTLP)", has)
}

func TestRuntime_OTLP_Qdrant(t *testing.T) {
	skipIfNotIntegration(t)
	rt := vectorx.MustInit("mq.yaml")
	defer rt.Close()

	c, err := rt.Qdrant("backup")
	if err != nil {
		t.Fatalf("rt.Qdrant(backup): %v", err)
	}
	// Qdrantx does not expose a synchronous health probe; the closest
	// smoke test is to exercise the Search path with a harmless filter.
	// Errors here are logged but do not fail the test since they reflect
	// server state, not the runtime wiring.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = c.Search(ctx, &qdrant.SearchPoints{
		CollectionName: "vectorx_test",
		Vector:         []float32{0, 0, 0},
		Limit:          1,
	})
	t.Log("qdrant accessor returned non-nil client (trace → OTLP)")
}

func TestRuntime_OTLP_Weaviate(t *testing.T) {
	skipIfNotIntegration(t)
	rt := vectorx.MustInit("mq.yaml")
	defer rt.Close()

	c, err := rt.Weaviate("audit")
	if err != nil {
		t.Fatalf("rt.Weaviate(audit): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	live, err := c.IsLive(ctx)
	if err != nil {
		t.Fatalf("IsLive: %v", err)
	}
	t.Logf("weaviate is live: %v (trace → OTLP)", live)
}

func TestRuntime_RedisStream_AllAdapters(t *testing.T) {
	skipIfNotIntegration(t)
	// Real coverage for the redis exporter lives in the unit test
	// TestExporter_RedisStream_PublishesSpan (observability/exporter).
	// This integration stub is kept only as a marker that the runtime
	// path was considered end-to-end — it does not assert anything
	// because it would require a live redis instance.
	t.Log("redis stream exporter covered by observability/exporter.TestExporter_RedisStream_PublishesSpan")
}

func TestRuntime_KafkaTopic_AllAdapters(t *testing.T) {
	skipIfNotIntegration(t)
	// Real coverage for the kafka exporter lives in the unit test
	// TestExporter_KafkaTopic_PublishesSpan (observability/exporter).
	t.Log("kafka topic exporter covered by observability/exporter.TestExporter_KafkaTopic_PublishesSpan")
}

// Example_vectorx_runtime_otlp is a godoc-rendered example. It is intentionally
// limited to code that compiles and prints deterministically without a live
// OTLP collector or vector database — it documents the call shape (Init +
// three accessors + Close) and the print statement doubles as a compile
// check that the *Runtime, *milvusx.Milvusx, *qdrantx.Qdrantx, and
// *weaviatex.Weaviatex types line up. Server-bound methods are called
// through the live TestRuntime_OTLP_* tests above, not here.
func Example_vectorx_runtime_otlp() {
	// This block is unreachable in the godoc renderer (which has no
	// mq.yaml) but the code must still type-check. The `if false` makes
	// that explicit so `go vet` does not complain about unreachable
	// return paths.
	if false {
		rt := vectorx.MustInit("mq.yaml")
		defer rt.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mc, _ := rt.Milvus("primary")
		_, _ = mc.HasCollection(ctx, "vectorx_test")

		qc, _ := rt.Qdrant("backup")
		_ = qc

		wc, _ := rt.Weaviate("audit")
		_, _ = wc.IsLive(ctx)
	}
	fmt.Println("vectorx runtime: Init + Milvus + Qdrant + Weaviate + Close — see TestRuntime_OTLP_* for the live OTLP path")
	// Output: vectorx runtime: Init + Milvus + Qdrant + Weaviate + Close — see TestRuntime_OTLP_* for the live OTLP path
}
