// vectorx/milvus/examples: 完整 CRUD + OTel 调用链测试
//
// 前提：docker compose up -d（在 vectorx/milvus/ 目录下）
// 运行：go run ./examples/
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/vectorx"
	_ "github.com/gospacex/vectorx/milvus"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func main() {
	ctx := context.Background()

	// 1. 初始化 OTel TracerProvider (Jaeger)
	tp, err := initTracerProvider("vectorx-milvus-example", "http://localhost:14268/api/traces")
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}
	defer func() { _ = tp.Shutdown(ctx) }()
	otel.SetTracerProvider(tp)

	// 2. 初始化 logx（开启 OTel 上链）
	if err := logx.ReadConfig(); err != nil {
		log.Printf("logx.ReadConfig fallback: %v", err)
	}
	logger := logx.GetGlobal()
	logger.InfoContext(ctx, "example started", logx.String("driver", "milvus"))

	// 3. 读取配置并创建客户端
	client, err := vectorx.ReadAndCreate("default", configx.ConfigOptions{
		ConfigFile: "config.yaml",
		ConfigType: "yaml",
	})
	if err != nil {
		logger.Fatal("create client failed", logx.Err(err))
	}
	defer client.Close()

	collectionName := "example_collection"

	// 4. 健康检查
	if err := client.HealthCheck(); err != nil {
		logger.Fatal("health check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "health check passed")

	// 5. CreateCollection
	logger.InfoContext(ctx, "creating collection", logx.String("collection", collectionName))
	if err := client.CreateCollection(ctx, collectionName, 4); err != nil {
		logger.Fatal("create collection failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "collection created", logx.String("collection", collectionName))

	// 6. Insert
	embeddings := []vectorx.Embedding{
		{ID: "doc1", Vector: []float32{1, 0, 0, 0}, Fields: map[string]interface{}{"title": "hello"}},
		{ID: "doc2", Vector: []float32{0, 1, 0, 0}, Fields: map[string]interface{}{"title": "world"}},
		{ID: "doc3", Vector: []float32{1, 1, 0, 0}, Fields: map[string]interface{}{"title": "both"}},
	}
	logger.InfoContext(ctx, "inserting embeddings", logx.Int("count", len(embeddings)))
	if err := client.Insert(ctx, collectionName, embeddings); err != nil {
		logger.Fatal("insert failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "insert succeeded")

	// 7. Search
	query := vectorx.Vector([]float32{1, 0, 0, 0})
	logger.InfoContext(ctx, "searching", logx.Int("limit", 3))
	result, err := client.Search(ctx, collectionName, query, 3)
	if err != nil {
		logger.Fatal("search failed", logx.Err(err))
	}
	for i, id := range result.IDs {
		fmt.Printf("  [%d] id=%s distance=%.4f fields=%v\n", i, id, result.Distances[i], result.Fields[i])
	}

	// 8. Delete
	logger.InfoContext(ctx, "deleting doc1", logx.String("id", "doc1"))
	if err := client.Delete(ctx, collectionName, []string{"doc1"}); err != nil {
		logger.Fatal("delete failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "delete succeeded")

	// 9. DropCollection
	logger.InfoContext(ctx, "dropping collection", logx.String("collection", collectionName))
	if err := client.DropCollection(ctx, collectionName); err != nil {
		logger.Fatal("drop collection failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "collection dropped", logx.String("collection", collectionName))

	fmt.Println("\n✅ All operations completed successfully.")
	time.Sleep(100 * time.Millisecond)
}

func initTracerProvider(serviceName, endpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	if err != nil {
		return nil, fmt.Errorf("create jaeger exporter: %w", err)
	}
	res, err := resource.New(context.Background(),
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	return tp, nil
}
