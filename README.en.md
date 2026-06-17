# VectorX

[![Go Version](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go)](https://go.dev)
[![Go Reference](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/gospacex/vectorx)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.44.0-blueviolet?logo=opentelemetry)](https://opentelemetry.io)
[![Milvus](https://img.shields.io/badge/Milvus-2.4.17-blue)](https://milvus.io)
[![Qdrant](https://img.shields.io/badge/Qdrant-1.13.4-red)](https://qdrant.tech)
[![Weaviate](https://img.shields.io/badge/Weaviate-1.30.2-green)](https://weaviate.io)
[![Race Clean](https://img.shields.io/badge/race%20detector-clean-success)](Makefile)
[![Govulncheck](https://img.shields.io/badge/vulncheck-passing-success)](https://go.dev/security/vuln)
[![Go Report Card](https://goreportcard.com/badge/github.com/gospacex/vectorx)](https://goreportcard.com/report/github.com/gospacex/vectorx)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![FOSSA Status](https://img.shields.io/badge/FOSSA-passing-brightgreen)](https://app.fossa.com/)
[![SemVer](https://img.shields.io/badge/versioning-SemVer_2.0-brightgreen)](https://semver.org)

> **Enterprise-grade Go SDK** providing a unified, OpenTelemetry-instrumented facade over Milvus, Qdrant, and Weaviate vector databases — with lazy-load singleton management, isomorphic YAML configuration, and a pluggable trace exporter seam.

---

## Table of Contents

- [Executive Summary](#executive-summary)
- [Why VectorX?](#why-vectorx)
- [Features](#features)
- [Compatibility & Versioning](#compatibility--versioning)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration Reference](#configuration-reference)
  - [YAML Schema](#yaml-schema)
  - [Named Instances](#named-instances)
  - [Configuration Isomorphism with mqx](#configuration-isomorphism-with-mqx)
- [Top-Level Runtime API](#top-level-runtime-api)
  - [Lifecycle](#lifecycle)
  - [Accessors & Error Contract](#accessors--error-contract)
  - [Graceful Shutdown](#graceful-shutdown)
- [Adapter API Reference](#adapter-api-reference)
  - [milvusx](#milvusx)
  - [qdrantx](#qdrantx)
  - [weaviatex](#weaviatex)
- [Observability](#observability)
  - [Trace Exporters](#trace-exporters)
  - [Metrics](#metrics)
  - [Static Decoupling Invariant](#static-decoupling-invariant)
- [Architecture](#architecture)
- [Performance & Benchmarks](#performance--benchmarks)
  - [Lazy Singleton Latency](#lazy-singleton-latency)
  - [Concurrent Access Patterns](#concurrent-access-patterns)
- [Security Posture](#security-posture)
  - [Security Vulnerability Reporting](#security-vulnerability-reporting)
- [Production Deployment](#production-deployment)
  - [Configuration Management](#configuration-management)
  - [Observability Pipeline](#observability-pipeline)
  - [Resource Management](#resource-management)
  - [Monitoring & Alerting](#monitoring--alerting)
- [Migration Guide](#migration-guide)
  - [From Vendor SDKs](#from-vendor-sdks)
  - [From Other Multi-Backend Libraries](#from-other-multi-backend-libraries)
- [Testing Strategy](#testing-strategy)
  - [Unit Tests](#unit-tests)
  - [Integration Tests](#integration-tests)
  - [Static Analysis & Linting](#static-analysis--linting)
- [Project Layout](#project-layout)
- [Makefile Reference](#makefile-reference)
- [Design Decisions & Rationale](#design-decisions--rationale)
- [Comparison with Alternatives](#comparison-with-alternatives)
- [Governance & Contributing](#governance--contributing)
  - [Release Process](#release-process)
  - [Contributing](#contributing)
  - [Code of Conduct](#code-of-conduct)
- [Roadmap](#roadmap)
- [Support & FAQ](#support--faq)
- [Examples](#examples)
- [License & Acknowledgments](#license--acknowledgments)

---

## Executive Summary

**VectorX** is a production-grade Go library that eliminates the operational overhead of managing multiple vector database SDKs. It provides:

| Concern | What VectorX handles | What you write |
|---------|----------------------|----------------|
| Client lifecycle | Lazy-builds and caches per-backend clients; idempotent shutdown | One `vectorx.Init("mq.yaml")` call |
| Distributed tracing | Auto-instruments every adapter method with OpenTelemetry spans; flush on `Close()` | Zero instrumentation code |
| Configuration | Single YAML file shared with sibling mqx SDK; field-isomorphic types prevent drift | `vectorx:` block in your existing `mq.yaml` |
| Multi-instance management | `sync.Map`-backed singleton cache per adapter supports named instances (primary, analytics, audit) | `rt.Milvus("primary")` vs `rt.Milvus("audit")` |
| Export flexibility | Pluggable `SpanPublisher` seam for OTLP (gRPC/HTTP), Redis Stream, Kafka Topic | Inject publisher implementation at startup |

**Target audience:** Platform engineering teams, infrastructure SREs, and Go backend developers operating multi-backend vector search infrastructure in production environments.

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
- **Fail-fast configuration.** `Init` returns `ErrNoAdaptersConfigured` when the YAML is missing every adapter block.
- **Built-in Prometheus metrics.** `vectorx_trace_exports_total` (counter) and `vectorx_trace_export_duration_seconds` (histogram).

---

## Compatibility & Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org). The public API consists of:

1. The top-level `vectorx` package (`Init`, `MustInit`, `Runtime`, public types)
2. The adapter packages (`milvusx`, `qdrantx`, `weaviatex`) — their exported `Get*`, `MustGet*`, and client methods
3. The `config` package — all exported `Config`, `*Config` structs, and `Load`
4. The `observability` package — `InitTracing`, `StartSpan`, and the `exporter` sub-package

| Component | Minimum Version | Recommended Version | Notes |
|---|---|---|---|
| Go | 1.26.2 | 1.26.2+ | Uses `errors.Join`, `atomic.Bool`, generic-friendly OTel APIs |
| OpenTelemetry SDK | 1.44.0 | 1.44.0 | `go.opentelemetry.io/otel`, `otel/sdk`, `otel/exporters/otlp/*` |
| Milvus server | 2.4.17 | 2.4.x | Backward-compatible within 2.x minor series |
| Qdrant server | 1.13.4 | 1.13.x+ | gRPC (default) or HTTP |
| Weaviate server | 1.30.2 | 1.30.x+ | HTTP, default port 8080 |
| mqx (sibling) | local `../mqx` replace | matching version | Field-isomorphic `TracingConfig` |

> **Backward compatibility guarantee:** Within a major version, adapter API signatures and config field names will not change. New adapter methods may be added with new minor versions.

---

## Installation

```bash
go get github.com/gospacex/vectorx
```

This pulls in the top-level `package vectorx` (Init / MustInit / Runtime) plus all three adapter packages and the OTel SDK. If you only need one adapter, import the package directly:

```go
import "github.com/gospacex/vectorx/milvusx"  // or /qdrantx, /weaviatex
```

The top-level package adds no transitive dependencies beyond the three adapter packages — no penalty for using it.

---

## Quick Start

The recommended one-line startup uses the Top-Level Runtime:

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

See [EXAMPLES.md](EXAMPLES.md) for RAG pipelines, multi-tenant setups, semantic cache, and graceful shutdown patterns.

---

## Configuration Reference

### YAML Schema

All configuration resides in a single `mq.yaml`, shared with the mqx SDK:

```yaml
vectorx:
  # --- Tracing (optional — omit or set enabled: false to disable) ---
  trace:
    enabled: true
    service_name: my-vector-service
    exporter: otlp                    # otlp | jaeger | redis | redis_stream | kafka
    endpoint: localhost:4317          # defaults: otlp-grpc=4317, otlp-http=4318, redis=6379, kafka=9092
    protocol: grpc                    # grpc | http (otlp only)
    insecure: true                    # set false (or omit) for TLS in production
    sampler_type: always_on
    sampler_ratio: 1.0

  # --- Milvus instances (optional — omit if not used) ---
  milvus:
    - name: primary
      address: localhost:19530
      username: ""
      password: ""
      db_name: default
      collection: vectorx_test

  # --- Qdrant instances (optional — omit if not used) ---
  qdrant:
    - name: primary
      host: localhost
      port: 6334
      grpc: true

  # --- Weaviate instances (optional — omit if not used) ---
  weaviate:
    - name: primary
      scheme: http
      host: localhost:8080
      class: VectorXTest
```

### Named Instances

Each adapter supports multiple named instances configured as a YAML list:

```yaml
milvus:
  - name: primary
    address: milvus-primary.internal:19530
  - name: audit
    address: milvus-audit.internal:19530
```

Each is independently lazy-loaded on first access via `Get*("name")`. This enables per-tenant, per-environment, or per-workload isolation from a single binary.

### Configuration Isomorphism with mqx

The `TracingConfig` type is a type alias for `mqx/config.TracingConfig`, enforced at compile time:

```go
type TracingConfig = mqx.TracingConfig  // isomorphic identity
```

A reflect-based test suite catches field drift between `vectorx` and `mqx` config structs. This guarantees that a YAML file authored for one SDK works without modification for the other.

---

## Top-Level Runtime API

### Lifecycle

```
Init(path)    → (*Runtime, error)   // Parse YAML, init tracing, register adapters
MustInit(path) → *Runtime           // Panics on error
Close()       → error               // Idempotent flush & shutdown
```

### Accessors & Error Contract

```go
rt := vectorx.MustInit("mq.yaml")
defer rt.Close()

m := rt.Milvus("primary")    // (*milvusx.Milvusx, error)
q := rt.Qdrant("primary")    // (*qdrantx.Qdrantx, error)
w := rt.Weaviate("primary")  // (*weaviatex.Weaviatex, error)
```

| Condition | Behavior |
|---|---|
| Adapter not in config | Returns `ErrNoSuchAdapter` |
| Client construction fails | Returns wrapped error from vendor SDK |
| `Close()` already called | Returns `ErrClosed` on any accessor |
| Empty config (no adapters) | `Init` returns `ErrNoAdaptersConfigured` |
| Concurrent access + `Close` | `sync.RWMutex` prevents TOCTOU races |

`Must*` variants panic with a non-nil `error` value — callers can `errors.As` uniformly.

### Graceful Shutdown

```go
// Signal handler example
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

done := make(chan struct{})
go func() {
    if err := rt.Close(); err != nil {
        log.Printf("vectorx shutdown: %v", err)
    }
    close(done)
}()

select {
case <-done:
case <-ctx.Done():
    log.Fatal("vectorx shutdown timed out")
}
```

---

## Adapter API Reference

> **Recommended:** use `rt := vectorx.MustInit("mq.yaml")` for the one-line startup; the per-adapter `Get*` API below remains available for advanced use.

### milvusx

```go
import "github.com/gospacex/vectorx/milvusx"

milvusx.SetConfigPath("mq.yaml")
c, err := milvusx.GetMilvus("primary")       // lazy-load singleton
c    := milvusx.MustGetMilvus("primary")      // panics on failure
defer c.Close()

// Collection operations
has, err := c.HasCollection(ctx, "name")
err     := c.CreateCollection(ctx, "name", dims)
err     := c.DropCollection(ctx, "name")
desc, err := c.DescribeCollection(ctx, "name")

// Data operations
ids, err := c.Insert(ctx, "name", vectors)
err     := c.Flush(ctx, "name")
results, err := c.Search(ctx, "name", vector, limit)
```

**Lifecycle:** `Close()` releases the gRPC connection. Safe to call multiple times.

### qdrantx

```go
import "github.com/gospacex/vectorx/qdrantx"

qdrantx.SetConfigPath("mq.yaml")
c, err := qdrantx.GetQdrant("primary")
c    := qdrantx.MustGetQdrant("primary")
defer c.Close()

err     := c.Upsert(ctx, "collection", points)
results, err := c.Search(ctx, "collection", vector, limit)
err     := c.Delete(ctx, "collection", filter)
```

**Lifecycle:** `Close()` releases the gRPC connection. Safe to call multiple times.

### weaviatex

```go
import "github.com/gospacex/vectorx/weaviatex"

weaviatex.SetConfigPath("mq.yaml")
c, err := weaviatex.GetWeaviate("primary")
c    := weaviatex.MustGetWeaviate("primary")

alive, err := c.IsLive(ctx)
data, err  := c.GraphQLRaw(ctx, query)
err        := c.CreateObject(ctx, obj)
err        := c.DeleteObject(ctx, className, id)
err        := c.CreateClass(ctx, class)
```

**Lifecycle:** Uses HTTP client; no persistent connections. `Close()` is intentionally absent — the client is garbage-collected with the singleton.

---

## Observability

### Trace Exporters

| Exporter | Config (`exporter:`) | Backend | Mechanism |
|---|---|---|---|
| OTLP gRPC | `otlp` | Jaeger / SigNoz / Grafana Tempo / any OTLP collector | Direct OTLP gRPC exporter (TLS by default; `insecure: true` to disable) |
| OTLP HTTP | `otlp` + `protocol: http` | Same | `otlptracehttp` exporter (TLS by default; `insecure: true` to disable) |
| Redis Stream | `redis` | Redis (reuses mqx handle) | `SpanPublisher` interface → Redis XADD |
| Kafka Topic | `kafka` | Kafka (reuses mqx handle) | `SpanPublisher` interface → Kafka produce |

#### Exporter Name Aliases

The `exporter:` field accepts both vectorx-native names and the canonical
mqx names. After `config.Load`, mqx's `Validate()` normalizes unknown
exporters to `jaeger`; vectorx's `Build` accepts both spellings:

| YAML value | mqx post-Validate | vectorx accepts |
|---|---|---|
| `otlp` | `jaeger` (fallback) | ✅ `otlp` (gRPC or HTTP) |
| `jaeger` | `jaeger` | ✅ `otlp` (modern Jaeger collectors speak OTLP natively) |
| `redis` | `redis_stream` (fallback) | ✅ `redis` (legacy vectorx name) |
| `redis_stream` | `redis_stream` | ✅ `redis` (mqx canonical) |
| `kafka` | `kafka` | ✅ `kafka` |

In other words, every name that is valid in **either** the mqx or the
vectorx schema round-trips through `config.Load` and into a working
`SpanExporter`. The accepted exporter values are case-insensitive.

#### TLS Posture

OTLP exporters default to **secure (TLS) connections**. To talk to a
plaintext collector (typical for local development), set
`insecure: true` in the `trace:` block:

```yaml
vectorx:
  trace:
    enabled: true
    exporter: otlp
    endpoint: localhost:4317
    protocol: grpc
    insecure: true   # plaintext; omit for TLS in production
```

For authenticated collectors, prefer a bearer token via custom headers
(works with both gRPC and HTTP):

```yaml
vectorx:
  trace:
    exporter: otlp
    endpoint: tempo.example.com:4317
    headers:
      Authorization: "Bearer ${OTLP_TOKEN}"   # expanded by envsubst at deploy time
```

`username` / `password` are also accepted; mqx's `Validate()` will
auto-fill `headers["Authorization"]` with a Basic-Auth header when both
are set. Headers you set explicitly take precedence.

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

This seam keeps the `observability` package free from direct redis/kafka SDK dependencies.

### Metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `vectorx_trace_exports_total` | Counter | `exporter`, `status` | Total trace exports by exporter and status (success/failure) |
| `vectorx_trace_export_duration_seconds` | Histogram | `exporter` | Export latency distribution by exporter |

Metrics are registered on the OpenTelemetry meter `github.com/gospacex/vectorx/observability` and available via any Prometheus-compatible scrape endpoint.

### Static Decoupling Invariant

```bash
go list -deps ./observability/...   # MUST NOT contain vectorx/{milvusx,qdrantx,weaviatex}
                                    # MUST NOT contain any /redis or /kafka module path
```

This is enforced in CI via `make validate`. The invariant is what makes VectorX portable: `observability/` can be vendored into a different binary (e.g., a sidecar) without dragging in heavyweight gRPC clients.

---

## Architecture

### Component Diagram

```
                    ┌──────────────────────────────────────────────┐
                    │              Your Application                 │
                    │  mq.yaml (single config, vectorx: + mqx: key) │
                    └──────────────────────┬───────────────────────┘
                                           │
                                  ┌────────▼────────┐
                                  │   config.Load()   │  yaml.v3 only
                                  └────────┬────────┘
                                           │
                     ┌─────────────────────┼──────────────────────┐
                     │                     │                      │
              ┌──────▼──────┐      ┌───────▼──────┐      ┌───────▼──────┐
              │   milvusx    │      │   qdrantx    │      │  weaviatex    │
              │  GetMilvus() │      │  GetQdrant() │      │ GetWeaviate() │
              │  Search()    │      │  Upsert()    │      │ GraphQLRaw()  │
              │  Insert()    │      │  Search()    │      │ IsLive()      │
              │  HasCol..()  │      │  Delete()    │      │ CreateObj()   │
              └──────┬───────┘      └───────┬───────┘      └───────┬───────┘
                     │                      │                      │
                     │           ┌──────────▼──────────┐           │
                     └──────────►│   observability     │◄──────────┘
                                │  InitTracing()       │
                                │  StartSpan()         │
                                ├──────────────────────┤
                                │  exporter.Build()    │
                                │  ├─ OTLP gRPC        │
                                │  ├─ OTLP HTTP        │
                                │  ├─ Redis Stream     │
                                │  └─ Kafka Topic      │
                                └──────────────────────┘
```

### Layer Isolation

| Layer | Package | Imports | Depends On |
|---|---|---|---|
| Config | `config/` | `yaml.v3` only | Nothing |
| Observability | `observability/` | OTel SDK, Prometheus | Nothing infra-specific |
| Adapter | `milvusx/`, `qdrantx/`, `weaviatex/` | config + observability + vendor SDK | Config, Observability |
| Facade | `vectorx` (root) | config + observability + all three adapters | Everything |

### Dependency Flow

```
config  ──►  observability  ──►  milvusx / qdrantx / weaviatex  ──►  vectorx (facade)
                                            │
                                            └──► vendor SDKs (milvus-sdk-go, go-client, weaviate-go-client)
```

The observability package never imports adapter packages or message-queue SDKs — verified by the static decouple invariant.

---

## Performance & Benchmarks

### Lazy Singleton Latency

| Scenario | First call (cold) | Subsequent calls (cached) |
|---|---|---|
| Single-threaded | gRPC handshake latency (~50–200ms) | ~50ns (pointer dereference) |
| 100 concurrent | One gRPC handshake; 99 wait | ~100ns (sync.RWMutex.RLock) |
| Multi-name, 100 concurrent | One handshake per unique name | ~150ns (sync.Map load) |

Benchmarks located in each adapter's `*_test.go`:

```bash
go test -bench=BenchmarkGet -benchmem ./milvusx/ ./qdrantx/ ./weaviatex/
```

### Concurrent Access Patterns

- **`sync.RWMutex` over `atomic.Bool`** for the closed gate: the accessor delegates to `Get*` after reading the closed flag. `atomic.Bool` is a snapshot; `sync.RWMutex` is a continuous gate that blocks `Close` until every in-flight accessor returns.
- **Per-key mutex** inside each adapter's `sync.Map`: ensures only one goroutine pays the gRPC dial cost per named instance.
- **Idempotent `Close`**: calling `rt.Close()` after the runtime is already closed is a no-op.
- **Concurrent reads**: accessor methods acquire `RLock`, so 100 concurrent `rt.Milvus("primary")` calls do not serialize.

---

## Security Posture

### Security Vulnerability Reporting

**Do not open a public GitHub issue for security vulnerabilities.** Report privately to the maintainers:

1. Email: [security@gospacex.com](mailto:security@gospacex.com)
2. Expected response time: **72 hours** for initial triage
3. We follow a **90-day disclosure deadline** from the date of fix release

### Operational Security

| Concern | Recommendation |
|---|---|
| **Plain YAML config** | Treat `mq.yaml` like any other config file. Do not commit credentials to VCS. Use a secrets manager (Vault, AWS Secrets Manager) or env-var expansion before `vectorx.Init` reads the file. |
| **Vendor SDK credentials** | `milvus.username` / `milvus.password`, Weaviate API keys, and OTLP auth headers pass straight through to the vendor SDKs — do not log them. |
| **OTLP endpoints** | Default is **TLS** `localhost:4317`. For local development, set `insecure: true` to talk to a plaintext collector. For production, use the default TLS path and supply a bearer token via `headers.Authorization` (or `username`/`password`, which mqx's `Validate()` converts to a Basic-Auth header). |
| **Span data** | OTLP exporters may emit PII (query text, payload sizes, collection names). Apply your redaction policy at the collector or via a custom `SpanProcessor`. |
| **Dependency hygiene** | Run `go mod tidy` and `govulncheck ./...` as part of your release pipeline. The `make validate` target enforces that `observability/` does not pull in redis/kafka SDKs. |
| **Race safety** | `go test -race ./...` must pass before every release. The `sync.RWMutex` gate prevents TOCTOU between accessor and `Close` paths. |
| **Supply chain** | All dependencies are pinned in `go.sum`. Use `go mod verify` to detect tampering. Run `golangci-lint run` with the `gosec` linter enabled. |

### Vulnerability Disclosure Policy

1. Reporter submits finding via `security@gospacex.com`
2. Maintainer acknowledges within 3 business days
3. Maintainer triages and validates within 10 business days
4. Fix is developed and reviewed in a private fork
5. Fix is released; CVE is published
6. Public disclosure after 90 days

---

## Production Deployment

### Configuration Management

- **Secret injection:** Use `envsubst` or a templating tool before `vectorx.Init` reads the file: `envsubst < mq.yaml.tpl > mq.yaml`
- **Config validation:** Run `vectorx.Init` with tracing disabled in your startup health check to validate YAML before exposing traffic
- **Hot-reload:** Not natively supported. Restart the process to pick up config changes. For zero-downtime, use a rolling restart pattern.

### Observability Pipeline

```
Application  ──►  OTLP gRPC/HTTP  ──►  Collector (optional)  ──►  Backend
  (StartSpan)                          (batch, filter, redact)     (Jaeger/Tempo/SigNoz)
```

- **Recommended:** Deploy an OTel Collector (e.g., `opentelemetry-collector-contrib`) between your application and the trace backend for batching, retry, and redaction
- **Sampling:** Configure `sampler_type: parentbased_traceidratio` and `sampler_ratio: 0.1` for high-throughput services to control trace volume

### Resource Management

| Resource | Recommendation |
|---|---|
| gRPC connections | One per named adapter instance; capped by vendor SDK. Set `MaxOpenConns` via vendor SDK options. |
| OpenTelemetry batch span processor | Default: 512 batch size, 5s export interval. Tune via OTel SDK options. |
| Memory | Lazy singleton pattern means zero memory for unused adapters. Each active gRPC connection uses ~1–2 MB. |

### Monitoring & Alerting

| Alert Rule | Metric | Threshold |
|---|---|---|
| Trace export failures | `rate(vectorx_trace_exports_total{status="failure"}[5m])` | > 0 over 5m |
| Slow trace export | `histogram_quantile(0.99, rate(vectorx_trace_export_duration_seconds_bucket[5m]))` | > 1s |
| Adapter unreachable | Vendor SDK errors propagated as trace export errors | > 5% error rate |
| gRPC connection drain | Application logs with `ErrClosed` | Any occurrence in steady state |

---

## Migration Guide

### From Vendor SDKs

**Step 1: Install VectorX**

```bash
go get github.com/gospacex/vectorx
```

**Step 2: Create configuration**

Create `mq.yaml` with your existing database credentials. See [Configuration Reference](#configuration-reference).

**Step 3: Replace direct SDK initialization**

Before (using Milvus SDK directly):

```go
cfg := milvusclient.NewConfig("localhost:19530", "")
c, err := milvusclient.NewClient(ctx, cfg)
if err != nil { ... }
defer c.Close()
```

After (using VectorX):

```go
rt := vectorx.MustInit("mq.yaml")
defer rt.Close()
c, err := rt.Milvus("primary")
```

**Step 4: Replace SDK method calls**

VectorX adapter methods match the vendor SDK parameter signatures closely. Most call sites require only changing the receiver type and removing the client-construction boilerplate.

### From Other Multi-Backend Libraries

VectorX differs from generic vector DB abstraction layers in that it does **not** define a unified `VectorDB` interface. Each backend's semantics (Milvus partitions vs Qdrant collections vs Weaviate classes) are surfaced natively. Migration involves:

1. Replace config loading with VectorX's YAML loader
2. Replace client construction with `Get*("<name>")` calls
3. Replace unified-interface method calls with direct adapter method calls

---

## Testing Strategy

### Unit Tests

```bash
make test          # go test ./...
make test-race     # go test -race ./...
make cover         # go test -coverprofile=cover.out ./...
```

Unit tests cover everything that can be exercised without talking to a
real database or message broker. They are intended to be fast, hermetic,
and runnable on every commit.

| Package | Coverage | What is covered |
|---|---|---|
| `vectorx` (top-level) | ~77% | Config loading, runtime lifecycle, accessor error contract, `Close` cascade to adapter caches, TOCTOU regression, idempotent close, joined-error aggregation across `CloseAll`, OTel closer-hook LIFO execution, closer-hook error propagation |
| `config` | ~89% | YAML parsing, field isomorphism, default values, mqx `Validate` normalization |
| `observability` | ~84% | Tracer no-op paths, error propagation, nil safety, sampler decision matrix |
| `observability/exporter` | ~96% | `Build` for every supported exporter alias; TLS-on / TLS-off; header propagation; recording `SpanPublisher` + sync span processor (the end-to-end span-publish path) |
| `milvusx` | ~75% | Singleton cache lifecycle (first-call / cache hit / concurrent access / close-eviction), config lookup, `Must*` panic values, error wrapping, every wrapped method (`Search`, `Insert`, `Flush`, `CreateCollection`, `DropCollection`, `HasCollection`, `DescribeCollection`) with span-name + int + string-attribute + error-recording assertions |
| `qdrantx` | ~89% | Singleton cache lifecycle, TLS dial-options (plaintext / TLS on / CA-file missing / CA-file invalid / CA-file valid), every wrapped method (24 methods: write, index, read paths) with span-name and error-recording assertions via interface-embedding fakes |
| `weaviatex` | ~77% | Singleton cache lifecycle, config lookup, every wrapped method (`GraphQLRaw`, `CreateObject`, `DeleteObject`, `CreateClass`, `IsLive`) with span-name and error-recording assertions via a `weaviateOps` interface seam |

For the `milvusx` and `qdrantx` adapters, vendor SDK interfaces
(`milvus client.Client`, `qdrant.PointsClient`) are interface-typed and
are unit-tested via thin fakes that embed the interface and override the
methods each test cares about — that gives us span-name, attribute, and
error-recording coverage without the 5–30-method mock surface. The
`weaviatex` adapter's `weaviate.Client` is a concrete type and cannot
be mocked at the interface level, so its per-method behavior is
verified in `example/weaviatex_test` (integration) only.

The "close evicts the cache" regression is unit-tested for `milvusx`
and `qdrantx` (see `TestMilvusx_Close_EvictsFromCache`,
`TestQdrantx_Close_EvictsFromCache`); it is the root cause of the
intermittent `grpc: the client connection is closing` errors seen
when multiple test cases share a named instance.

### Integration Tests

Integration tests require real vector database containers:

```bash
cd example && docker compose up -d
INTEGRATION=1 go test -tags integration ./example/...
```

Tests are build-tagged `//go:build integration` and gated by `INTEGRATION=1`:

| Test | Exporter | Verifies |
|---|---|---|
| `TestRuntime_OTLP_Milvus` | OTLP gRPC | `HasCollection` with span recording |
| `TestRuntime_OTLP_Qdrant` | OTLP gRPC | `Search` with span recording |
| `TestRuntime_OTLP_Weaviate` | OTLP gRPC | `IsLive` with span recording |
| `TestExporter_RedisStream_PublishesSpan` | Redis Stream | Span record delivered to `SpanPublisher` |
| `TestExporter_KafkaTopic_PublishesSpan` | Kafka Topic | Span record delivered to `SpanPublisher` |

### Static Analysis & Linting

```bash
make lint       # golangci-lint run (gosec, govet, staticcheck)
make validate   # build + vet + race test + static decouple check
```

---

## Project Layout

```
vectorx/                          # Module root (github.com/gospacex/vectorx)
├── vectorx.go                    # Top-level package: Init / MustInit / Runtime
├── vectorx_test.go               # Unit tests for Runtime accessors & lifecycle
├── vectorx_close_test.go         # TOCTOU race, idempotent close tests
├── vectorx_deps_test.go          # Static dependency constraint tests
├── config/                       # YAML configuration
│   ├── load.go                   # Load() — config struct definition
│   ├── tracing.go                # TracingConfig type alias (mqx isomorphic)
│   ├── milvus.go / qdrant.go / weaviate.go  # Per-adapter config structs
│   └── *_test.go                 # Config parsing, isomorphism, validation
├── milvusx/                      # Milvus adapter
│   ├── milvusx.go                # GetMilvus / MustGetMilvus + sync.Map cache
│   ├── client.go                 # Milvusx struct, Search / Insert / HasCollection...
│   ├── config.go                 # Config loading
│   └── tracing.go                # Blank import of observability
├── qdrantx/                      # Qdrant adapter
│   ├── qdrantx.go                # GetQdrant / MustGetQdrant
│   ├── client.go                 # Qdrantx struct, Upsert / Search / Delete
│   └── config.go                 # Config loading
├── weaviatex/                    # Weaviate adapter
│   ├── weaviatex.go              # GetWeaviate / MustGetWeaviate
│   ├── client.go                 # Weaviatex struct, GraphQLRaw / CreateObject...
│   └── config.go                 # Config loading
├── observability/                 # OpenTelemetry integration
│   ├── tracing.go                # InitTracing / StartSpan
│   ├── tracer.go                 # Tracer name, propagator
│   ├── metrics.go                # Prometheus counters & histograms
│   └── exporter/                 # Pluggable span exporters
│       ├── exporter.go           # Build(), SpanPublisher interface
│       ├── jaeger.go             # OTLP gRPC/HTTP exporter builder
│       ├── redis.go              # Redis Stream exporter builder
│       └── kafka.go              # Kafka Topic exporter builder
├── example/                      # Build-tagged integration tests
│   ├── docker-compose.yml        # Milvus + Qdrant + Weaviate containers
│   ├── milvusx_test/             # Milvus e2e tests
│   ├── qdrantx_test/             # Qdrant e2e tests
│   ├── weaviatex_test/           # Weaviate e2e tests
│   └── vectorx_runtime_test/     # Runtime e2e tests
├── utils/                        # Shared helpers (doc.go placeholder)
├── go.mod                        # Module definition
├── go.sum                        # Dependency checksums
├── Makefile                      # Build / test / lint / validate
├── README.md                     # This file
└── EXAMPLES.md                   # Usage patterns (RAG, multi-tenant, etc.)
```

---

## Makefile Reference

| Target | Command | Description |
|---|---|---|
| `build` | `go build ./...` | Compile all packages |
| `test` | `go test ./...` | Run unit tests |
| `test-race` | `go test -race ./...` | Run unit tests with race detector |
| `cover` | `go test -coverprofile=cover.out ./...` | Run tests with coverage output |
| `lint` | `golangci-lint run` | Run Go linters (gosec, govet, staticcheck) |
| `validate` | `build + vet + race + deps` | Pre-commit / CI gate |

---

## Design Decisions & Rationale

| Decision | Rationale |
|---|---|
| Lazy-load singleton per name | `sync.Map` + per-key mutex supports multiple named instances, unlike `sync.Once` which is global |
| Observability as standalone package | Adapters import it; it imports nothing from adapters. Enforces unidirectional dependency and enables sidecar reuse |
| Thin wrapper per adapter | No unified `VectorDB` interface — each backend's semantics differ (Milvus partitions vs Qdrant collections vs Weaviate classes). A unified interface would be lossy or lowest-common-denominator |
| Config fields isomorphic to mqx | Same `TracingConfig` struct shared between mqx and vectorx; reflect-based test catches drift at compile time |
| `SpanPublisher` seam for redis/kafka | Avoids importing redis/kafka SDKs in `observability`; application injects the publisher at startup |
| Integration tests gated by build tag | `//go:build integration` + `INTEGRATION=1` — zero impact on normal `go test ./...` |
| Top-level `vectorx` package is a thin facade | No client state; per-adapter `sync.Map` singletons remain the single source of truth |
| `sync.RWMutex` over `atomic.Bool` for closed gate | `atomic.Bool` is a snapshot; `sync.RWMutex` blocks `Close` until every in-flight accessor returns, preventing TOCTOU |
| YAML config (not Viper/koanf) | A shared struct + reflect-based test catches drift at compile time; a generic config library would let the two projects diverge silently |

---

## Comparison with Alternatives

| Project | Scope | Tracing | Multi-backend | Lazy Singleton | OTel-native |
|---|---|---|---|---|---|
| **vectorx** | Go SDK for Milvus / Qdrant / Weaviate | OTLP + Redis + Kafka (via seam) | ✓ | ✓ | ✓ |
| [milvus-sdk-go](https://github.com/milvus-io/milvus-sdk-go) | Milvus only | Manual | ✗ | ✗ | Optional |
| [qdrant/go-client](https://github.com/qdrant/go-client) | Qdrant only | Manual | ✗ | ✗ | Optional |
| [weaviate-go-client](https://github.com/weaviate/weaviate-go-client) | Weaviate only | Manual | ✗ | ✗ | Optional |
| [chroma-go](https://github.com/amikos-tech/chroma-go) | Chroma | Manual | ✗ | ✗ | ✗ |

**Choose vectorx when** you need (a) a single Go binary that talks to multiple vector backends, (b) OpenTelemetry tracing that works out of the box, and (c) lazy-loaded named instances (e.g., per-tenant or per-environment). For a single-backend, single-language project, the vendor SDKs may be sufficient.

---

## Governance & Contributing

### Release Process

| Step | Action | Responsible |
|---|---|---|
| 1 | Feature development on `main` (linear history) | Contributors |
| 2 | `make validate` passes | CI |
| 3 | `govulncheck ./...` passes | CI |
| 4 | Code review (minimum 1 maintainer approval) | Maintainers |
| 5 | Tag `vX.Y.Z` (SemVer) | Maintainers |
| 6 | Changelog updated in release notes | Maintainers |
| 7 | GitHub Release created with artifacts | Automation |

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Run `make validate` before committing
4. Add or update tests for any new functionality — coverage on touched packages must stay ≥ 80%
5. For new SDK adapters, follow the `milvusx/` pattern as template
6. Ensure the [static decouple invariant](#static-decoupling-invariant) holds for `observability/`

**Code style:** `gofmt` + `go vet` + `go test -race` enforced by CI.

**Commit convention:** [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(milvusx): add CreatePartition support
fix(qdrantx): race in concurrent upsert
docs: update migration guide
```

### Code of Conduct

This project adheres to the [Contributor Covenant](https://www.contributor-covenant.org/) code of conduct. By participating, you agree to maintain a harassment-free environment for everyone.

---

## Roadmap

| Version | Theme | Items |
|---|---|---|
| v1.1 | Connection pools | Per-adapter `MaxOpenConns`, `IdleTimeout`, health-check pinging |
| v1.2 | Resilience | Built-in retry / circuit-breaker middleware for transient gRPC errors |
| v1.3 | More backends | pgvector adapter (ParadeDB, Apache AGE-style hybrid search) |
| v1.4 | Bulk operations | `BulkInsert(ctx, collection, embeddings, metadata)` for >10k vectors |
| v2.0 | Generated stubs | Drop `replace` directives; use generated gRPC stubs from upstream protos |

Items above are plans, not commitments. See [GitHub issues](https://github.com/gospacex/vectorx/issues) for the live backlog.

---

## Support & FAQ

**Where do I ask questions?**
- Usage / design questions: [GitHub Discussions](https://github.com/gospacex/vectorx/discussions)
- Bug reports / feature requests: [GitHub Issues](https://github.com/gospacex/vectorx/issues)
- Security vulnerabilities: `security@gospacex.com` (private)

**Why no vendor SDK abstraction (`VectorDB` interface)?** Each backend has different semantics (Milvus partitions vs Qdrant collections vs Weaviate classes). A unifying interface would be lossy or lowest-common-denominator. Better to expose idiomatic per-adapter methods.

**Why a YAML config and not Viper / koanf?** The `vectorx:` block is field-isomorphic with the `mqx:` block in the same file. A shared struct + reflect-based test catches drift at compile time; a generic config library would let the two projects diverge silently.

**Can I use VectorX without the top-level package?** Yes — import the adapter package directly (e.g. `github.com/gospacex/vectorx/milvusx`) and call `SetConfigPath` / `GetMilvus` yourself. The top-level package is a convenience.

**What is the production support model?** Community support via GitHub. Enterprise support agreements are available — contact `info@gospacex.com`.

**How do I benchmark my configuration?** Run `go test -bench=BenchmarkGet -benchmem ./milvusx/ ./qdrantx/ ./weaviatex/` for singleton access latency. For end-to-end query performance, use your application's benchmark suite against the vendor SDKs.

---

## Examples

Full usage patterns are documented in [EXAMPLES.md](EXAMPLES.md), including:

- **RAG pipeline** — embed → store → search → generate
- **Multi-tenant isolation** — one Runtime, per-tenant named instances
- **Semantic cache** — Qdrant-backed query cache with TTL
- **Graceful shutdown** — signal handling + context timeout + Close
- **Custom exporter** — implementing `SpanPublisher` for a custom backend

---

## License & Acknowledgments

**License:** Released under the **MIT License**. See [`LICENSE`](LICENSE) for the full text.

**Acknowledgments:** Thanks to the [Milvus](https://milvus.io), [Qdrant](https://qdrant.tech), and [Weaviate](https://weaviate.io) teams for the official Go SDKs; the [OpenTelemetry](https://opentelemetry.io) project for the tracing SDK and OTLP wire format; and the [mqx](https://github.com/gospacex/mqx) contributors whose isomorphic config design inspired this SDK.
