# Examples

Real-world patterns built on top of VectorX. The application code in each example
stays focused on business logic; tracing and connection pooling are handled by
the SDK.

## RAG Ingestion

Batch-ingest documents into Milvus, with per-call OTel spans emitted automatically:

```go
rt := vectorx.MustInit("mq.yaml")
defer rt.Close()

m, _ := rt.Milvus("primary")
ctx := context.Background()

for _, doc := range documents {
    vec, _ := embedder.Embed(ctx, doc.Text)
    _, err := m.Insert(ctx, "docs", "", "",
        column.NewColumnVarChar([]string{doc.ID}),
        column.NewColumnFloatVector(vec))
    if err != nil {
        log.Printf("insert %s: %v", doc.ID, err)
    }
}
```

## Multi-Tenant Search

Route each tenant to a dedicated Qdrant collection via a per-tenant collection
name. The Qdrant client itself is shared (one `*qdrantx.Qdrantx` for the
process); only the collection name varies per tenant:

```go
rt := vectorx.MustInit("mq.yaml")
defer rt.Close()

q, _ := rt.Qdrant("primary")
ctx := context.Background()

for _, t := range tenants {
    collection := "tenant_" + t.ID
    hits, _ := q.Search(ctx, collection, t.QueryVector, 10, nil, nil)
    route(t, hits)
}
```

For tenants that need fully isolated backends, configure multiple Qdrant
instances in `mq.yaml` and resolve per-tenant via `rt.Qdrant(tenant.Tier)`.

## Semantic Cache

Store query embeddings in Weaviate, look up top-K similar entries before
recomputing. The OTel span for `GraphQLRaw` makes cache hit rate visible in your
tracing backend without extra instrumentation:

```go
rt := vectorx.MustInit("mq.yaml")
defer rt.Close()

w, _ := rt.Weaviate("cache")
ctx := context.Background()

vec, _ := embedder.Embed(ctx, query)
hits, _ := w.GraphQLRaw(ctx, fmt.Sprintf(
    `{ Get { Cache(nearVector: {vector: %s}) { question answer } } }`,
    vecJSON(vec)))

if len(hits) == 0 {
    ans, _ := llm.Complete(ctx, query)
    _, _ = w.CreateObject(ctx, "Cache",
        map[string]any{"question": query, "answer": ans}, vec)
}
```

## Switching Trace Exporter at Runtime

To swap OTLP for Redis Stream (for example), edit `mq.yaml` to set
`exporter: redis`, restart, and inject a `SpanPublisher` before the first
`MustInit`:

```go
import "github.com/gospacex/vectorx/observability/exporter"

exporter.SetRedisPublisher(myRedisPublisher)   // before vectorx.MustInit
rt := vectorx.MustInit("mq.yaml")
```

The `observability/` package never imports a redis SDK directly; the
application injects the publisher through the `SpanPublisher` seam.

## Graceful Shutdown

```go
rt := vectorx.MustInit("mq.yaml")

ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

go func() {
    <-ctx.Done()
    if err := rt.Close(); err != nil {
        log.Printf("vectorx close: %v", err)
    }
}()

// ... main loop ...
```

`rt.Close()` is idempotent — the second call returns `nil` without re-running
closers, so double-shutdown from multiple signal handlers is safe.
