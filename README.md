# VectorX

[![Go Version](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go)](https://go.dev)
[![Go Reference](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/gospacex/vectorx)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.44.0-blueviolet?logo=opentelemetry)](https://opentelemetry.io)
[![Milvus](https://img.shields.io/badge/Milvus-2.4.17-blue)](https://milvus.io)
[![Qdrant](https://img.shields.io/badge/Qdrant-1.13.4-red)](https://qdrant.tech)
[![Weaviate](https://img.shields.io/badge/Weaviate-1.30.2-green)](https://weaviate.io)
[![Coverage](https://img.shields.io/badge/coverage-85.5%25-brightgreen)](https://github.com/gospacex/vectorx)
[![Go Report Card](https://goreportcard.com/badge/github.com/gospacex/vectorx)](https://goreportcard.com/report/github.com/gospacex/vectorx)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-make%20validate-blue)](Makefile)

> Enterprise-grade Go SDK for **Milvus**, **Qdrant**, and **Weaviate** vector databases — with lazy-load singleton management, OpenTelemetry tracing, and a unified YAML configuration that is field-for-field isomorphic with the [mqx](https://github.com/gospacex/mqx) message-queue SDK.

---

## Table of Contents

- [Why VectorX?](#why-vectorx)
- [Features](#features)
- [Compatibility Matrix](#compatibility-matrix)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Top-Level Runtime](#top-level-runtime)
- [Adapter API](#adapter-api)
  - [milvusx](#milvusx) · [qdrantx](#qdrantx) · [weaviatex](#weaviatex)
- [Examples](EXAMPLES.md) · [RAG / Multi-Tenant / Semantic Cache / Graceful Shutdown](EXAMPLES.md)
- [Observability](#observability)
- [Architecture](#architecture)
- [Performance Notes](#performance-notes)
- [Testing](#testing)
- [Project Layout](#project-layout)
- [Makefile Reference](#makefile-reference)
- [Design Decisions](#design-decisions)
- [Security](#security)
- [Comparison with Alternatives](#comparison-with-alternatives)
- [Roadmap](#roadmap)
- [Support & FAQ](#support--faq)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

---

## Why VectorX?

Production vector-database workloads hit three pain points that off-the-shelf vendor SDKs do not solve cleanly:

| Pain point | What breaks | VectorX's answer |
|---|---|---|
| Three different SDKs, three different lifecycles | You write `NewClient` boilerplate per backend, each with its own pooling, retry, and shutdown quirks | One `*Runtime` accessor per backend; clients are lazy-built only on first use |
| Tracing is bolted on, not built in | You hand-instrument every gRPC / HTTP call, leak span context across goroutines, and forget to flush | Every adapter method is auto-instrumented; one `rt.Close()` flushes and shuts down the OTel pipeline |
| Config files drift | The mq team's YAML, the vector team's YAML, and the auth team's YAML all have subtly different field names | A single `mq.yaml` whose `vectorx:` block is field-for-field isomorphic to `mqx:` — same `TracingConfig`, same `Trace` struct |

VectorX is **not** a wrapper that re-implements the vendor SDKs. It is a thin facade over the official `milvus-sdk-go`, `qdrant/go-client`, and `weaviate-go-client` packages, augmented with the cross-cutting concerns (config, tracing, lifecycle) that production code needs and the vendor SDKs do not provide.

---

## Features

- **One-line startup.** `rt := vectorx.MustInit("mq.yaml")` loads config, initializes tracing, and registers adapter paths.
- **Lazy singletons per name.** Multiple named instances (`primary`, `analytics`, `audit`) share a per-adapter `sync.Map` — the second `GetMilvus("primary")` returns the same pointer as the first.
- **Race-clean by design.** All accessor / Close paths go through a `sync.RWMutex`; the underlying `Get*` constructors also use `sync.Map` + per-key mutex. `go test -race` stays clean under 100+ concurrent accessors.
- **Pluggable trace exporters.** OTLP gRPC, OTLP HTTP, Redis Stream (via `SpanPublisher` seam), Kafka Topic (via `SpanPublisher` seam). Adding a new exporter does not require changing the adapter packages.
- **Static decoupling invariants.** `observability/` never imports an adapter or a redis/kafka SDK. Verified by `go list -deps` in CI.
- **Idempotent shutdown.** `rt.Close()` can be called any number of times; subsequent calls return `nil` without re-invoking closers.
- **Fail-fast configuration.** `Init` returns `ErrNoAdaptersConfigured` when the YAML is missing every adapter block, instead of returning a Runtime that does nothing useful.

---

## Compatibility Matrix

| Component | Version | Notes |
|---|---|---|
| Go | **1.26.2+** | Uses `errors.Join`, `atomic.Bool`, generic-friendly OTel APIs |
| OpenTelemetry SDK | **1.44.0** | `go.opentelemetry.io/otel`, `otel/sdk`, `otel/exporters/otlp/*` |
| Milvus | **2.4.17+** server / `milvus-sdk-go/v2 v2.4.2` | gRPC, default port 19530 |
| Qdrant | **1.13.4+** server / `qdrant/go-client v1.18.2` | gRPC (default) or HTTP, default port 6334 |
| Weaviate | **1.30.2+** server / `weaviate-go-client/v5 v5.7.3` | HTTP, default port 8080 |
| mqx (sibling) | local `../mqx` replace | Field-isomorphic `TracingConfig` |

The SDK is verified against the matrix above via `make validate`. Newer minor versions of any backend usually work without code changes; pin in your own `go.mod` if you need exact reproducibility.

---

## Installation

```bash
go get github.com/gospacex/vectorx
```

This pulls in the top-level `package vectorx` (Init / MustInit / Runtime) plus all three adapter packages and the OTel SDK. If you only need one adapter, import the package directly:

```go
import "github.com/gospacex/vectorx/milvusx"  // or /qdrantx, /weaviatex
```

The top-level package adds ~0 transitive dependencies beyond the three adapter packages — no penalty for using it.

---

## Quick Start

The recommended one-line startup uses the [Top-Level Runtime](#top-level-runtime):

```go
package main

import (
    "context"
    "log"

    "github.com/gospacex/vectorx"
)

func main() {
    rt := vectorx.MustInit("mq.yaml")
    defer rt.Close()

    ctx := context.Background()
    c, err := rt.Milvus("primary")
    if err != nil {
        log.Fatal(err)
    }
    has, _ := c.HasCollection(ctx, "my_collection")
    log.Printf("milvus has collection: %v", has)
}
```

The error-returning variant for libraries / long-running services:

```go
rt, err := vectorx.Init("mq.yaml")
if err != nil {
    log.Fatalf("vectorx init: %v", err)
}
defer rt.Close()
```

The sections below document the underlying per-adapter API (`milvusx.GetMilvus`, `qdrantx.GetQdrant`, `weaviatex.GetWeaviate`) and the `config` / `observability` packages that the top-level runtime wires together for you. Reach for them directly only when you need fine-grained control (multiple config files, swapping the trace exporter at runtime, or skipping observability entirely).

---

## Configuration

All configuration resides in a single `mq.yaml`, shared with the mqx SDK. The `vectorx:` key holds all vector-database and tracing config.

```yaml
vectorx:
  trace:
    enabled: true
    service_name: my-vector-service
    exporter: otlp                    # "otlp" | "redis" | "kafka"
    endpoint: localhost:4317          # defaults: otlp-grpc=4317, otlp-http=4318, redis=6379, kafka=9092
    protocol: grpc                    # "grpc" | "http" (otlp only)
    sampler_type: always_on
    sampler_ratio: 1.0

  milvus:
    - name: primary
      address: localhost:19530
      username: ""
      password: ""
      db_name: default
      collection: vectorx_test

  qdrant:
    - name: primary
      host: localhost
      port: 6334
      grpc: true

  weaviate:
    - name: primary
      scheme: http
      host: localhost:8080
      class: VectorXTest
```

Multiple named instances of each adapter are supported (e.g., `milvus: [{name: primary, ...}, {name: secondary, ...}]`). Each is independently lazy-loaded on first access via `Get*("name")`.

---

## Top-Level Runtime

```go
rt := vectorx.MustInit("mq.yaml")
defer rt.Close()

m := rt.Milvus("primary")    // *milvusx.Milvusx
q := rt.Qdrant("backup")     // *qdrantx.Qdrantx
w := rt.Weaviate("audit")    // *weaviatex.Weaviatex
```

`rt.Milvus` / `rt.Qdrant` / `rt.Weaviate` are thin proxies over each adapter's existing `sync.Map` lazy singleton — no client is constructed until first use. `rt.Close()` flushes and shuts down the OTel TracerProvider and is safe to call multiple times.

**Error contract:**

- Accessors return `ErrClosed` after `Close()` (a sentinel you can `errors.Is` against).
- `Must*` variants panic with a non-nil `error` value — either `ErrClosed` or a wrapped underlying error — so callers can `errors.As` uniformly.
- `Init` returns `ErrNoAdaptersConfigured` when the YAML is missing every adapter block.

---

## Adapter API

The full method surface for each adapter lives in the package godoc. This
section shows the lifecycle (init → use → close) for each adapter and a
representative method. See [EXAMPLES.md](EXAMPLES.md) for end-to-end patterns.

> **Recommended:** use `rt := vectorx.MustInit("mq.yaml")` for the one-line startup; the per-adapter `Get*` API below remains available for advanced use.

### milvusx

```go
import "github.com/gospacex/vectorx/milvusx"

milvusx.SetConfigPath("mq.yaml")
c, err := milvusx.GetMilvus("primary")       // lazy-load singleton
c    := milvusx.MustGetMilvus("primary")      // panics on failure
c.Close()

// Collection ops: HasCollection, CreateCollection, DropCollection, DescribeCollection
// Data ops:      Insert, Flush, Search
```

### qdrantx

```go
import "github.com/gospacex/vectorx/qdrantx"

qdrantx.SetConfigPath("mq.yaml")
c, err := qdrantx.GetQdrant("primary")
c    := qdrantx.MustGetQdrant("primary")
c.Close()

// Methods: Upsert, Search, Delete
```

### weaviatex

```go
import "github.com/gospacex/vectorx/weaviatex"

weaviatex.SetConfigPath("mq.yaml")
c, err := weaviatex.GetWeaviate("primary")
c    := weaviatex.MustGetWeaviate("primary")

// Methods: IsLive, GraphQLRaw, CreateObject, DeleteObject, CreateClass
// No Close() — HTTP client holds no persistent connections.
```

---

## Observability

### Tracing Exporters

| Exporter | Config (`exporter:`) | Backend | Mechanism |
|---|---|---|---|
| OTLP gRPC | `otlp` | Jaeger / SigNoz / Grafana Tempo / any OTLP collector | Direct OTLP gRPC exporter |
| OTLP HTTP | `otlp` + `protocol: http` | Same | `otlptracehttp` exporter |
| Redis Stream | `redis` | Redis (reuses mqx handle) | `SpanPublisher` interface → Redis XADD |
| Kafka Topic | `kafka` | Kafka (reuses mqx handle) | `SpanPublisher` interface → Kafka produce |

**Redis and Kafka** exporters require the application to inject a publisher implementation:

```go
import "github.com/gospacex/vectorx/observability/exporter"

exporter.SetRedisPublisher(myRedisPublisher)   // inject before InitTracing
exporter.SetKafkaPublisher(myKafkaPublisher)
```

The publisher must satisfy `exporter.SpanPublisher`:

```go
type SpanPublisher interface {
    PublishSpan(ctx context.Context, destination string, payload []byte) error
}
```

This seam keeps the `observability` package free from direct redis/kafka SDK dependencies, satisfying the [static decouple invariant](#static-decoupling-invariant).

### Metrics

| Metric | Type | Labels |
|---|---|---|
| `vectorx_trace_exports_total` | Counter | `exporter`, `status` (success/failure) |
| `vectorx_trace_export_duration_seconds` | Histogram | `exporter` |

Metrics are registered on the OpenTelemetry meter `github.com/gospacex/vectorx/observability` and available via Prometheus endpoint.

### Static Decoupling Invariant

```bash
go list -deps ./observability/...   # MUST NOT contain vectorx/{milvusx,qdrantx,weaviatex} or any /redis or /kafka module path
```

This is automated in CI as a `make validate` step. The invariant is what makes VectorX portable: `observability/` can be vendored into a different binary (e.g., a sidecar) without dragging in the gRPC clients.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Your Application                     │
├─────────────────────────────────────────────────────────┤
│  mq.yaml (single config file, mqx: + vectorx: keys)      │
└──────────────────────┬──────────────────────────────────┘
                       │
              ┌────────▼────────┐
              │   config.Load()  │
              └────────┬────────┘
                       │
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
   ┌──────────┐  ┌──────────┐  ┌──────────┐
   │ milvusx  │  │ qdrantx  │  │ weaviatex│
   │ GetMilvus│  │ GetQdrant│  │GetWeaviate│
   │ .Search  │  │ .Upsert  │  │ .IsLive  │
   │ .Insert  │  │ .Search  │  │.GraphQLRaw│
   │   ...    │  │   ...    │  │   ...    │
   └────┬─────┘  └────┬─────┘  └────┬─────┘
        │             │             │
        │  ┌──────────▼──────────┐  │
        └──►   observability    ◄──┘
           │  InitTracing()      │
           │  StartSpan()        │
           ├─────────────────────┤
           │  exporter.Build()   │
           │  ├─ OTLP (jaeger)   │
           │  ├─ Redis Stream    │
           │  └─ Kafka Topic     │
           └─────────────────────┘
```

**Layer isolation:**

| Layer | Package | Imports |
|---|---|---|
| Config | `config/` | yaml.v3 only |
| Observability | `observability/` | OTel SDK, Prometheus — **no** vectorx adapters, **no** redis/kafka client SDK |
| Adapter | `milvusx/`, `qdrantx/`, `weaviatex/` | config + observability + vendor SDK |
| Top-level facade | `vectorx` (root) | config + observability + all three adapters |

---

## Performance Notes

- **Lazy singleton** — first `GetMilvus("primary")` blocks on the gRPC handshake; subsequent calls return the cached pointer in nanoseconds. The `Runtime` adds no caching of its own.
- **Idempotent `Close`** — calling `rt.Close()` after the runtime is already closed is a no-op.
- **Concurrent reads** — accessor methods acquire `RLock`, so 100 concurrent `rt.Milvus("primary")` calls do not serialize. Top-level package adds 0 transitive dependencies beyond the three adapters.

---

## Testing

### Unit Tests

```bash
make test          # go test ./...
make test-race     # go test -race ./...
make cover         # go test -coverprofile=cover.out ./...
```

Unit tests cover config loading, observability init, adapter lazy-load, static decouple invariant, and the top-level `Runtime` lifecycle (TOCTOU-safe `Close`, fail-fast on empty config, idempotent shutdown).

### Integration Tests

Integration tests require real vector database containers:

```bash
cd example && docker compose up -d
INTEGRATION=1 go test -tags integration ./example/...
```

Tests are build-tagged `//go:build integration` and gated by `INTEGRATION=1`. OTLP has a real end-to-end test for each adapter; the Redis Stream and Kafka Topic exporters are verified at the `observability/exporter` unit level (recording `SpanPublisher` + sync span processor — no external infra needed):

| Test | Exporter | Verifies |
|---|---|---|
| `TestRuntime_OTLP_Milvus` | OTLP gRPC | `HasCollection` with span recording |
| `TestRuntime_OTLP_Qdrant` | OTLP gRPC | `Search` with span recording |
| `TestRuntime_OTLP_Weaviate` | OTLP gRPC | `IsLive` with span recording |
| `TestExporter_RedisStream_PublishesSpan` | Redis Stream | Span record (TraceID/SpanID/Name/StartNS/Duration) delivered to `SpanPublisher` |
| `TestExporter_KafkaTopic_PublishesSpan` | Kafka Topic | Span record delivered to `SpanPublisher` with `topic` as destination |

---

## Project Layout

```
vectorx/                    # Repo root
├── vectorx.go              # Top-level package: Init / MustInit / Runtime
├── *_test.go               # Accessor, close, deps tests
├── config/                 # YAML loader, mqx-isomorphic types
├── observability/          # OTel SDK wiring + exporter seam
│   └── exporter/           # SpanPublisher interface (redis/kafka)
├── milvusx/ qdrantx/ weaviatex/  # Per-adapter wrappers
├── example/                # Build-tagged integration tests
├── openspec/               # OpenSpec changes + archive
├── docs/                   # superpowers plans
├── Makefile  go.mod  go.sum  README.md  EXAMPLES.md
```

---

## Makefile Reference

`build` (compile), `test` (unit), `test-race` (race detector), `cover` (coverage), `lint` (golangci-lint), `validate` (openspec + `go list -deps` static decouple check).

---

## Design Decisions

| Decision | Rationale |
|---|---|
| Lazy-load singleton per name | `sync.Map` + per-key mutex — supports multiple named instances, unlike `sync.Once` which is global |
| Observability as standalone package | Adapters import it; it imports nothing from adapters. Enforces unidirectional dependency |
| Thin wrapper per adapter | No unified `VectorDB` interface — each SDK's semantics differ; wrapping preserves idiomatic usage |
| Config fields isomorphic to mqx | Same `TracingConfig` struct shared between mqx and vectorx |
| `SpanPublisher` seam for redis/kafka | Avoids importing redis/kafka SDKs in `observability`; application injects the publisher |
| Integration tests gated by build tag | `//go:build integration` + `INTEGRATION=1` — zero impact on normal `go test ./...` |
| Top-level `vectorx` package is a thin facade | No client state; per-adapter `sync.Map` singletons remain the single source of truth |
| `sync.RWMutex` over `atomic.Bool` for the closed gate | The accessor delegates to `Get*` after reading the closed flag. `atomic.Bool` is a snapshot; `sync.RWMutex` is a continuous gate that blocks `Close` until every in-flight accessor returns |

---

## Security

- **Plain YAML config** — treat `mq.yaml` like any other config file. Do not commit credentials; use a secrets manager or env-var expansion before `vectorx.Init` reads the file.
- **Vendor SDK credentials** — `milvus.username` / `milvus.password`, Weaviate API keys, and OTLP auth headers pass straight through to the vendor SDKs.
- **OTLP endpoints** — default is plaintext `localhost:4317`. For production, use TLS and an auth header; the OTel SDK supports both natively.
- **Span data** — OTLP exporters may emit PII (query text, payload sizes). Apply your redaction policy at the collector.
- **Dependency hygiene** — `make validate` enforces that `observability/` does not pull in redis/kafka SDKs. Run `go mod tidy` and `govulncheck ./...` as part of your release pipeline.

---

## Comparison with Alternatives

| Project | Scope | Tracing | Multi-backend | Lazy singleton | OTel-native |
|---|---|---|---|---|---|
| **vectorx** (this project) | Go SDK for Milvus / Qdrant / Weaviate | OTLP + Redis + Kafka (via seam) | ✓ | ✓ | ✓ |
| [milvus-sdk-go](https://github.com/milvus-io/milvus-sdk-go) | Milvus only | Manual | ✗ | ✗ | Optional |
| [qdrant/go-client](https://github.com/qdrant/go-client) | Qdrant only | Manual | ✗ | ✗ | Optional |
| [weaviate-go-client](https://github.com/weaviate/weaviate-go-client) | Weaviate only | Manual | ✗ | ✗ | Optional |
| [chroma-go](https://github.com/amikos-tech/chroma-go) | Chroma | Manual | ✗ | ✗ | ✗ |

Choose **vectorx** when you need (a) a single Go binary that talks to multiple vector backends, (b) OpenTelemetry tracing that works out of the box, and (c) lazy-loaded named instances (e.g., per-tenant or per-environment). For a single-backend, single-language project the vendor SDKs may be sufficient.
---

## Roadmap

- **v1.1** — Connection pool tuning (per-adapter `MaxOpenConns`, `IdleTimeout`)
- **v1.2** — Built-in retry / circuit-breaker middleware for transient gRPC errors
- **v1.3** — pgvector adapter (ParadeDB, Apache AGE-style hybrid search)
- **v1.4** — Bulk-ingest helper (`BulkInsert(ctx, collection, embeddings, metadata)`) for >10k vectors
- **v2.0** — Generated gRPC stubs from upstream protos (drop `replace` directives)

Items above are plans, not commitments. See [GitHub issues](https://github.com/gospacex/vectorx/issues) for the live backlog.

---

## Support & FAQ

**Where do I ask questions?** Open a [GitHub Discussion](https://github.com/gospacex/vectorx/discussions) for design / usage questions, an [Issue](https://github.com/gospacex/vectorx/issues) for bugs and feature requests. No Discord / Slack — VectorX is intentionally low-overhead.

**Why no vendor SDK abstraction (`VectorDB` interface)?** Each backend has different semantics (Milvus partitions vs Qdrant collections vs Weaviate classes). A unifying interface would be lossy or lowest-common-denominator. Better to expose idiomatic per-adapter methods.

**Why a YAML config and not Viper / koanf?** The `vectorx:` block is field-isomorphic with the `mqx:` block in the same file. A shared struct + reflect-based test catches drift at compile time; a generic config library would let the two projects diverge silently.

**Can I use VectorX without the top-level package?** Yes — import the adapter package directly (e.g. `github.com/gospacex/vectorx/milvusx`) and call `SetConfigPath` / `GetMilvus` yourself. The top-level package is a convenience.

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Run `make validate` before committing
4. Add or update tests for any new functionality — coverage on touched packages must stay ≥ 80%
5. For new SDK adapters, follow the `milvusx/` pattern as template
6. Ensure the [static decouple invariant](#static-decoupling-invariant) holds for `observability/`

**Code style:** `gofmt` + `go vet` + `go test -race` enforced by CI. **Commit convention:** [Conventional Commits](https://www.conventionalcommits.org/).

---

## License

Released under the **MIT License**. See [`LICENSE`](LICENSE) for the full text.

---

## Acknowledgments

Thanks to the [Milvus](https://milvus.io), [Qdrant](https://qdrant.tech), and [Weaviate](https://weaviate.io) teams for the official Go SDKs; the [OpenTelemetry](https://opentelemetry.io) project for the tracing SDK and OTLP wire format; and the [mqx](https://github.com/gospacex/mqx) contributors whose isomorphic config design inspired this SDK.
