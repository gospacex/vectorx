// vectorx/pinecone/examples: Pinecone 云向量数据库示例
//
// 运行前需设置环境变量 PINECONE_API_KEY
// 运行：go run ./examples/
//
// 注意：Pinecone 为云服务，无 docker-compose。
// 当前驱动为 stub 状态，CreateCollection/Insert/Delete 尚未实现，
// 示例仅展示客户端创建 + OTel 初始化 + Search 调用。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/vectorx"
	_ "github.com/gospacex/vectorx/pinecone"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("PINECONE_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  PINECONE_API_KEY not set, skipping example")
		fmt.Println("   export PINECONE_API_KEY=your-key-here")
		return
	}

	// 1. 初始化 OTel TracerProvider (Jaeger)
	tp, err := initTracerProvider("vectorx-pinecone-example", "http://localhost:14268/api/traces")
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}
	defer func() { _ = tp.Shutdown(ctx) }()
	otel.SetTracerProvider(tp)

	// 2. 初始化 logx
	if err := logx.ReadConfig(); err != nil {
		log.Printf("logx.ReadConfig fallback: %v", err)
	}
	logger := logx.GetGlobal()
	logger.InfoContext(ctx, "example started", logx.String("driver", "pinecone"))

	// 3. 读取配置并创建客户端
	client, err := vectorx.ReadAndCreate("default", configx.ConfigOptions{
		ConfigFile: "config.yaml",
		ConfigType: "yaml",
	})
	if err != nil {
		logger.Fatal("create client failed", logx.Err(err))
	}
	defer client.Close()
	logger.InfoContext(ctx, "pinecone client created")

	// 4. Search（stub 返回空结果，验证调用链可达）
	query := vectorx.Vector([]float32{1, 0, 0, 0})
	logger.InfoContext(ctx, "searching", logx.Int("limit", 3))
	result, err := client.Search(ctx, "example-index", query, 3)
	if err != nil {
		logger.WarnContext(ctx, "search returned error (expected in stub)", logx.Err(err))
	} else {
		for i, id := range result.IDs {
			fmt.Printf("  [%d] id=%s score=%.4f\n", i, id, result.Distances[i])
		}
	}

	fmt.Println("\n✅ Pinecone example completed (stub mode).")
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
