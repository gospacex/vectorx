# VectorX

[![Go 版本](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go)](https://go.dev)
[![Go 参考](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/gospacex/vectorx)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-1.44.0-blueviolet?logo=opentelemetry)](https://opentelemetry.io)
[![Milvus](https://img.shields.io/badge/Milvus-2.4.17-blue)](https://milvus.io)
[![Qdrant](https://img.shields.io/badge/Qdrant-1.13.4-red)](https://qdrant.tech)
[![Weaviate](https://img.shields.io/badge/Weaviate-1.30.2-green)](https://weaviate.io)
[![竞态检测](https://img.shields.io/badge/race%20detector-clean-success)](Makefile)
[![漏洞检查](https://img.shields.io/badge/vulncheck-passing-success)](https://go.dev/security/vuln)
[![Go 报告卡](https://goreportcard.com/badge/github.com/gospacex/vectorx)](https://goreportcard.com/report/github.com/gospacex/vectorx)
[![许可证](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![FOSSA](https://img.shields.io/badge/FOSSA-passing-brightgreen)](https://app.fossa.com/)
[![SemVer](https://img.shields.io/badge/versioning-SemVer_2.0-brightgreen)](https://semver.org)

> **企业级 Go SDK**，为 Milvus、Qdrant 和 Weaviate 向量数据库提供统一的、基于 OpenTelemetry 可观测性的外观层 —— 具备懒加载单例管理、同构 YAML 配置和可插拔的追踪导出器接口。

---

## 目录

- [执行摘要](#执行摘要)
- [为什么选择 VectorX？](#为什么选择-vectorx)
- [功能特性](#功能特性)
- [兼容性与版本](#兼容性与版本)
- [安装](#安装)
- [快速开始](#快速开始)
- [配置参考](#配置参考)
  - [YAML 模式](#yaml-模式)
  - [命名实例](#命名实例)
  - [与 mqx 的配置同构性](#与-mqx-的配置同构性)
- [顶层运行时 API](#顶层运行时-api)
  - [生命周期](#生命周期)
  - [访问器与错误约定](#访问器与错误约定)
  - [优雅关闭](#优雅关闭)
- [适配器 API 参考](#适配器-api-参考)
  - [milvusx](#milvusx)
  - [qdrantx](#qdrantx)
  - [weaviatex](#weaviatex)
- [可观测性](#可观测性)
  - [追踪导出器](#追踪导出器)
  - [指标](#指标)
  - [静态解耦不变式](#静态解耦不变式)
- [架构](#架构)
- [性能与基准测试](#性能与基准测试)
  - [懒加载单例延迟](#懒加载单例延迟)
  - [并发访问模式](#并发访问模式)
- [安全态势](#安全态势)
  - [漏洞报告](#漏洞报告)
- [生产部署](#生产部署)
  - [配置管理](#配置管理)
  - [可观测性管道](#可观测性管道)
  - [资源管理](#资源管理)
  - [监控与告警](#监控与告警)
- [迁移指南](#迁移指南)
  - [从厂商 SDK 迁移](#从厂商-sdk-迁移)
  - [从其他多后端库迁移](#从其他多后端库迁移)
- [测试策略](#测试策略)
  - [单元测试](#单元测试)
  - [集成测试](#集成测试)
  - [静态分析与 Lint](#静态分析与-lint)
- [项目结构](#项目结构)
- [Makefile 参考](#makefile-参考)
- [设计决策与原理](#设计决策与原理)
- [竞品对比](#竞品对比)
- [治理与贡献](#治理与贡献)
  - [发布流程](#发布流程)
  - [贡献指南](#贡献指南)
  - [行为准则](#行为准则)
- [路线图](#路线图)
- [支持与 FAQ](#支持与-faq)
- [示例](#示例)
- [许可证与致谢](#许可证与致谢)

---

## 执行摘要

**VectorX** 是一个生产级 Go 库，消除了管理多个向量数据库 SDK 的操作开销：

| 关注点 | VectorX 的处理 | 你需要写的代码 |
|---------|---------------|---------------|
| 客户端生命周期 | 懒加载并缓存每个后端的客户端；幂等关闭 | 一次 `vectorx.Init("mq.yaml")` 调用 |
| 分布式追踪 | 自动为每个适配器方法注入 OpenTelemetry 追踪；`Close()` 时统一刷新 | 零埋点代码 |
| 配置 | 单个 YAML 文件，与兄弟 SDK mqx 共享；字段同构类型防止漂移 | 现有 `mq.yaml` 中的 `vectorx:` 块 |
| 多实例管理 | 每个适配器的 `sync.Map` 单例缓存，支持命名实例（primary、analytics、audit） | `rt.Milvus("primary")` vs `rt.Milvus("audit")` |
| 导出灵活性 | 可插拔 `SpanPublisher` 接口，支持 OTLP（gRPC/HTTP）、Redis Stream、Kafka Topic | 启动时注入发布器实现 |

**目标用户：** 平台工程团队、基础设施 SRE、在生产环境中运维多后端向量搜索基础设施的 Go 后端开发者。

---

## 为什么选择 VectorX？

生产级向量数据库工作负载会遇到三个痛点，而现成的厂商 SDK 无法干净地解决：

| 痛点 | 后果 | VectorX 的答案 |
|------|------|---------------|
| 三种不同的 SDK，三种不同的生命周期 | 你需要为每个后端编写 `NewClient` 样板代码，各有不同的连接池、重试和关闭逻辑 | 每个后端一个 `*Runtime` 访问器；客户端仅在首次使用时懒加载创建 |
| 追踪是后加的，不是内置的 | 你需要手动为每个 gRPC/HTTP 调用埋点，跨 goroutine 传播 span 上下文，并记得刷新 | 每个适配器方法都自动埋点；一次 `rt.Close()` 刷新并关闭整个 OTel 管道 |
| 配置文件漂移 | 消息队列团队的 YAML、向量团队的 YAML、认证团队的 YAML 有微妙的字段名差异 | 单个 `mq.yaml` 的 `vectorx:` 块与 `mqx:` 块字段完全同构 —— 相同的 `TracingConfig`，相同的 `Trace` 结构体 |

VectorX **不是**重新实现厂商 SDK 的包装器。它是在官方 `milvus-sdk-go`、`qdrant/go-client` 和 `weaviate-go-client` 包之上的薄外观层，增加了生产代码需要而厂商 SDK 未提供的横切关注点（配置、追踪、生命周期）。

---

## 功能特性

- **一行启动。** `rt := vectorx.MustInit("mq.yaml")` 加载配置、初始化追踪并注册适配器路径。
- **按名称懒加载单例。** 多个命名实例（`primary`、`analytics`、`audit`）共享每个适配器的 `sync.Map` —— 第二次 `GetMilvus("primary")` 返回与第一次相同的指针。
- **竞态安全设计。** 所有访问器/Close 路径通过 `sync.RWMutex`；底层的 `Get*` 构造函数也使用 `sync.Map` + 每个键的互斥锁。`go test -race` 在 100+ 并发访问器下保持干净。
- **可插拔追踪导出器。** OTLP gRPC、OTLP HTTP、Redis Stream（通过 `SpanPublisher` 接口）、Kafka Topic（通过 `SpanPublisher` 接口）。添加新导出器无需修改适配器包。
- **静态解耦不变式。** `observability/` 从不导入适配器或 redis/kafka SDK。通过 CI 中的 `go list -deps` 验证。
- **幂等关闭。** `rt.Close()` 可被调用任意次数；后续调用返回 `nil` 而不重新调用关闭器。
- **快速失败配置。** 当 YAML 缺少所有适配器块时，`Init` 返回 `ErrNoAdaptersConfigured`。
- **内置 Prometheus 指标。** `vectorx_trace_exports_total`（计数器）和 `vectorx_trace_export_duration_seconds`（直方图）。

---

## 兼容性与版本

本项目遵循[语义化版本 2.0.0](https://semver.org)。公开 API 包括：

1. 顶层 `vectorx` 包（`Init`、`MustInit`、`Runtime`、公开类型）
2. 适配器包（`milvusx`、`qdrantx`、`weaviatex`）—— 其导出的 `Get*`、`MustGet*` 和客户端方法
3. `config` 包 —— 所有导出的 `Config`、`*Config` 结构体和 `Load`
4. `observability` 包 —— `InitTracing`、`StartSpan` 和 `exporter` 子包

| 组件 | 最低版本 | 推荐版本 | 说明 |
|--------|-------------|-----------------|-------|
| Go | 1.26.2 | 1.26.2+ | 使用 `errors.Join`、`atomic.Bool`、通用友好的 OTel API |
| OpenTelemetry SDK | 1.44.0 | 1.44.0 | `go.opentelemetry.io/otel`、`otel/sdk`、`otel/exporters/otlp/*` |
| Milvus 服务器 | 2.4.17 | 2.4.x | 在 2.x 次要版本系列内向后兼容 |
| Qdrant 服务器 | 1.13.4 | 1.13.x+ | gRPC（默认）或 HTTP |
| Weaviate 服务器 | 1.30.2 | 1.30.x+ | HTTP，默认端口 8080 |
| mqx（兄弟项目） | local `../mqx` replace | 匹配版本 | 字段同构的 `TracingConfig` |

> **向后兼容性保证：** 在一个主版本内，适配器 API 签名和配置字段名称不会更改。新的适配器方法可能通过新的次版本添加。

---

## 安装

```bash
go get github.com/gospacex/vectorx
```

这会拉取顶层 `package vectorx`（Init / MustInit / Runtime）以及所有三个适配器包和 OTel SDK。如果你只需要一个适配器，直接导入该包：

```go
import "github.com/gospacex/vectorx/milvusx"  // 或 /qdrantx, /weaviatex
```

顶层包除了三个适配器包外不增加任何传递依赖 —— 使用它没有额外开销。

---

## 快速开始

推荐的一行启动方式使用顶层运行时：

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

适用于库/长时间运行服务的返回错误版本：

```go
rt, err := vectorx.Init("mq.yaml")
if err != nil {
    log.Fatalf("vectorx init: %v", err)
}
defer rt.Close()
```

参见 [EXAMPLES.md](EXAMPLES.md) 了解 RAG 管道、多租户设置、语义缓存和优雅关闭模式。

---

## 配置参考

### YAML 模式

所有配置都位于单个 `mq.yaml` 中，与 mqx SDK 共享：

```yaml
vectorx:
  # --- 追踪（可选 —— 省略或设置 enabled: false 以禁用它） ---
  trace:
    enabled: true
    service_name: my-vector-service
    exporter: otlp                    # "otlp" | "redis" | "kafka"
    endpoint: localhost:4317          # 默认值：otlp-grpc=4317, otlp-http=4318, redis=6379, kafka=9092
    protocol: grpc                    # "grpc" | "http"（仅 otlp）
    sampler_type: always_on
    sampler_ratio: 1.0

  # --- Milvus 实例（可选 —— 不使用则省略） ---
  milvus:
    - name: primary
      address: localhost:19530
      username: ""
      password: ""
      db_name: default
      collection: vectorx_test

  # --- Qdrant 实例（可选 —— 不使用则省略） ---
  qdrant:
    - name: primary
      host: localhost
      port: 6334
      grpc: true

  # --- Weaviate 实例（可选 —— 不使用则省略） ---
  weaviate:
    - name: primary
      scheme: http
      host: localhost:8080
      class: VectorXTest
```

### 命名实例

每个适配器支持多个命名实例，配置为 YAML 列表：

```yaml
milvus:
  - name: primary
    address: milvus-primary.internal:19530
  - name: audit
    address: milvus-audit.internal:19530
```

每个实例在首次通过 `Get*("name")` 访问时独立懒加载。这实现了从单个二进制文件的每个租户、每个环境或每个工作负载的隔离。

### 与 mqx 的配置同构性

`TracingConfig` 类型是 `mqx/config.TracingConfig` 的类型别名，在编译时强制：

```go
type TracingConfig = mqx.TracingConfig  // 同构标识
```

基于反射的测试套件捕获 `vectorx` 和 `mqx` 配置结构体之间的字段漂移。这保证了一个 SDK 编写的 YAML 文件无需修改即可用于另一个 SDK。

---

## 顶层运行时 API

### 生命周期

```
Init(path)    → (*Runtime, error)   // 解析 YAML，初始化追踪，注册适配器
MustInit(path) → *Runtime           // 出错时 panic
Close()       → error               // 幂等刷新和关闭
```

### 访问器与错误约定

```go
rt := vectorx.MustInit("mq.yaml")
defer rt.Close()

m := rt.Milvus("primary")    // (*milvusx.Milvusx, error)
q := rt.Qdrant("primary")    // (*qdrantx.Qdrantx, error)
w := rt.Weaviate("primary")  // (*weaviatex.Weaviatex, error)
```

| 条件 | 行为 |
|---|---|
| 适配器未在配置中 | 返回 `ErrNoSuchAdapter` |
| 客户端构造失败 | 返回来自厂商 SDK 的包装错误 |
| 已调用 `Close()` | 任何访问器返回 `ErrClosed` |
| 空配置（无适配器） | `Init` 返回 `ErrNoAdaptersConfigured` |
| 并发访问 + `Close` | `sync.RWMutex` 防止 TOCTOU 竞态 |

`Must*` 变体使用非 nil `error` 值 panic —— 调用者可以统一使用 `errors.As`。

### 优雅关闭

```go
// 信号处理示例
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

## 适配器 API 参考

> **推荐：** 使用 `rt := vectorx.MustInit("mq.yaml")` 进行一行启动；下面的每个适配器 `Get*` API 仍可用于高级用途。

### milvusx

```go
import "github.com/gospacex/vectorx/milvusx"

milvusx.SetConfigPath("mq.yaml")
c, err := milvusx.GetMilvus("primary")       // 懒加载单例
c    := milvusx.MustGetMilvus("primary")      // 失败时 panic
defer c.Close()

// 集合操作
has, err := c.HasCollection(ctx, "name")
err     := c.CreateCollection(ctx, "name", dims)
err     := c.DropCollection(ctx, "name")
desc, err := c.DescribeCollection(ctx, "name")

// 数据操作
ids, err := c.Insert(ctx, "name", vectors)
err     := c.Flush(ctx, "name")
results, err := c.Search(ctx, "name", vector, limit)
```

**生命周期：** `Close()` 释放 gRPC 连接。可安全多次调用。

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

**生命周期：** `Close()` 释放 gRPC 连接。可安全多次调用。

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

**生命周期：** 使用 HTTP 客户端；无持久连接。`Close()` 有意缺席 —— 客户端随单例一起被 GC。

---

## 可观测性

### 追踪导出器

| 导出器 | 配置（`exporter:`） | 后端 | 机制 |
|---|---|---|---|
| OTLP gRPC | `otlp` | Jaeger / SigNoz / Grafana Tempo / 任意 OTLP 收集器 | 直接 OTLP gRPC 导出器 |
| OTLP HTTP | `otlp` + `protocol: http` | 同上 | `otlptracehttp` 导出器 |
| Redis Stream | `redis` | Redis（复用 mqx 句柄） | `SpanPublisher` 接口 → Redis XADD |
| Kafka Topic | `kafka` | Kafka（复用 mqx 句柄） | `SpanPublisher` 接口 → Kafka 生产 |

**Redis 和 Kafka** 导出器需要应用程序注入发布器实现：

```go
import "github.com/gospacex/vectorx/observability/exporter"

exporter.SetRedisPublisher(myRedisPublisher)   // 在 InitTracing 之前注入
exporter.SetKafkaPublisher(myKafkaPublisher)
```

发布器必须满足 `exporter.SpanPublisher`：

```go
type SpanPublisher interface {
    PublishSpan(ctx context.Context, destination string, payload []byte) error
}
```

这个接口层使得 `observability` 包免于直接依赖 redis/kafka SDK。

### 指标

| 指标 | 类型 | 标签 | 说明 |
|---|---|---|---|
| `vectorx_trace_exports_total` | 计数器 | `exporter`、`status` | 按导出器和状态（成功/失败）统计的追踪导出总数 |
| `vectorx_trace_export_duration_seconds` | 直方图 | `exporter` | 按导出器统计的导出延迟分布 |

指标注册在 OpenTelemetry meter `github.com/gospacex/vectorx/observability` 上，可通过任意 Prometheus 兼容的抓取端点获取。

### 静态解耦不变式

```bash
go list -deps ./observability/...   # 不得包含 vectorx/{milvusx,qdrantx,weaviatex}
                                    # 不得包含任何 /redis 或 /kafka 模块路径
```

这在 CI 中通过 `make validate` 强制执行。该不变式是 VectorX 可移植性的关键：`observability/` 可以 vendored 到另一个二进制文件中（例如，sidecar），而不会拖入重量级的 gRPC 客户端。

---

## 架构

### 组件图

```
                    ┌──────────────────────────────────────────────┐
                    │              你的应用程序                       │
                    │  mq.yaml（单个配置，vectorx: + mqx: 键）         │
                    └──────────────────────┬───────────────────────┘
                                           │
                                  ┌────────▼────────┐
                                  │   config.Load()   │  yaml.v3 仅此而已
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

### 层隔离

| 层 | 包 | 导入 | 依赖 |
|---|---|---|
| 配置 | `config/` | 仅 `yaml.v3` | 无 |
| 可观测性 | `observability/` | OTel SDK、Prometheus | 无基础设施特定 |
| 适配器 | `milvusx/`、`qdrantx/`、`weaviatex/` | config + observability + 厂商 SDK | Config、Observability |
| 外观层 | `vectorx`（根） | config + observability + 所有三个适配器 | 全部 |

### 依赖流向

```
config  ──►  observability  ──►  milvusx / qdrantx / weaviatex  ──►  vectorx（外观层）
                                            │
                                            └──► 厂商 SDK（milvus-sdk-go、go-client、weaviate-go-client）
```

可观测性包从不导入适配器包或消息队列 SDK —— 由静态解耦不变式验证。

---

## 性能与基准测试

### 懒加载单例延迟

| 场景 | 首次调用（冷启动） | 后续调用（已缓存） |
|---|---|---|
| 单线程 | gRPC 握手延迟（约 50–200ms） | 约 50ns（指针解引用） |
| 100 并发 | 一次 gRPC 握手；99 个等待 | 约 100ns（sync.RWMutex.RLock） |
| 多名称，100 并发 | 每个唯一名称一次握手 | 约 150ns（sync.Map 加载） |

基准测试位于每个适配器的 `*_test.go` 中：

```bash
go test -bench=BenchmarkGet -benchmem ./milvusx/ ./qdrantx/ ./weaviatex/
```

### 并发访问模式

- **关闭闸门使用 `sync.RWMutex` 而非 `atomic.Bool`：** 访问器在读取关闭标志后委托给 `Get*`。`atomic.Bool` 是一个快照；`sync.RWMutex` 是一个连续闸门，会阻塞 `Close` 直到每个正在进行的访问器返回。
- **每个适配器的 `sync.Map` 内每个键的互斥锁：** 确保每个命名实例只有一个 goroutine 支付 gRPC 拨号成本。
- **幂等 `Close`：** 运行时已关闭后调用 `rt.Close()` 是无操作的。
- **并发读取：** 访问器方法获取 `RLock`，因此 100 个并发 `rt.Milvus("primary")` 调用不会串行化。

---

## 安全态势

### 漏洞报告

**请勿为安全漏洞公开 GitHub issue。** 私下向维护者报告：

1. 邮箱：[security@gospacex.com](mailto:security@gospacex.com)
2. 预期响应时间：**72 小时**内进行初步分类
3. 我们遵循从修复版本发布之日起的 **90 天披露截止日期**

### 操作安全

| 关注点 | 建议 |
|---|---|
| **明文 YAML 配置** | 将 `mq.yaml` 视为任何其他配置文件。不要将凭据提交到 VCS。使用密钥管理器（Vault、AWS Secrets Manager）或在 `vectorx.Init` 读取文件之前进行环境变量替换。 |
| **厂商 SDK 凭据** | `milvus.username` / `milvus.password`、Weaviate API 密钥和 OTLP 认证头部直接传递给厂商 SDK —— 不要记录它们。 |
| **OTLP 端点** | 默认是明文 `localhost:4317`。对于生产环境，使用 TLS 和 bearer token 认证；OTel SDK 通过标准 OTLP 导出器选项原生支持两者。 |
| **Span 数据** | OTLP 导出器可能泄露 PII（查询文本、负载大小、集合名称）。在收集器或通过自定义 `SpanProcessor` 应用你的脱敏策略。 |
| **依赖项卫生** | 在发布管道中运行 `go mod tidy` 和 `govulncheck ./...`。`make validate` 目标强制执行 `observability/` 不会引入 redis/kafka SDK。 |
| **竞态安全** | 每次发布前必须通过 `go test -race ./...`。`sync.RWMutex` 闸门防止访问器和 `Close` 路径之间的 TOCTOU。 |
| **供应链** | 所有依赖项都固定在 `go.sum` 中。使用 `go mod verify` 检测篡改。运行启用了 `gosec` linter 的 `golangci-lint run`。 |

### 漏洞披露政策

1. 报告者通过 `security@gospacex.com` 提交发现
2. 维护者在 3 个工作日内确认
3. 维护者在 10 个工作日内分类和验证
4. 在私有分支中开发和审查修复
5. 发布修复；发布 CVE
6. 90 天后公开披露

---

## 生产部署

### 配置管理

- **密钥注入：** 在 `vectorx.Init` 读取文件之前使用 `envsubst` 或模板工具：`envsubst < mq.yaml.tpl > mq.yaml`
- **配置验证：** 在暴露流量之前，在启动健康检查中使用禁用追踪的 `vectorx.Init` 验证 YAML
- **热重载：** 不原生支持。重启进程以应用配置更改。如需零停机，使用滚动重启模式。

### 可观测性管道

```
应用程序  ──►  OTLP gRPC/HTTP  ──►  收集器（可选）  ──►  后端
  (StartSpan)                          (批处理、过滤、脱敏)     (Jaeger/Tempo/SigNoz)
```

- **推荐：** 在应用程序和追踪后端之间部署 OTel 收集器（例如 `opentelemetry-collector-contrib`），用于批处理、重试和脱敏
- **采样：** 对于高吞吐量服务，配置 `sampler_type: parentbased_traceidratio` 和 `sampler_ratio: 0.1` 以控制追踪量

### 资源管理

| 资源 | 建议 |
|---|---|
| gRPC 连接 | 每个命名适配器实例一个；由厂商 SDK 限制。通过厂商 SDK 选项设置 `MaxOpenConns`。 |
| OpenTelemetry 批处理 span 处理器 | 默认：512 批量大小，5 秒导出间隔。通过 OTel SDK 选项调整。 |
| 内存 | 懒加载单例模式意味着未使用的适配器零内存。每个活动 gRPC 连接使用约 1–2 MB。 |

### 监控与告警

| 告警规则 | 指标 | 阈值 |
|---|---|---|
| 追踪导出失败 | `rate(vectorx_trace_exports_total{status="failure"}[5m])` | 5 分钟内 > 0 |
| 慢追踪导出 | `histogram_quantile(0.99, rate(vectorx_trace_export_duration_seconds_bucket[5m]))` | > 1 秒 |
| 适配器不可达 | 厂商 SDK 错误作为追踪导出错误传播 | 错误率 > 5% |
| gRPC 连接断开 | 应用程序日志中的 `ErrClosed` | 稳态下的任何出现 |

---

## 迁移指南

### 从厂商 SDK 迁移

**步骤 1：安装 VectorX**

```bash
go get github.com/gospacex/vectorx
```

**步骤 2：创建配置**

使用你现有的数据库凭据创建 `mq.yaml`。参见[配置参考](#配置参考)。

**步骤 3：替换直接的 SDK 初始化**

之前（直接使用 Milvus SDK）：

```go
cfg := milvusclient.NewConfig("localhost:19530", "")
c, err := milvusclient.NewClient(ctx, cfg)
if err != nil { ... }
defer c.Close()
```

之后（使用 VectorX）：

```go
rt := vectorx.MustInit("mq.yaml")
defer rt.Close()
c, err := rt.Milvus("primary")
```

**步骤 4：替换 SDK 方法调用**

VectorX 适配器方法的参数签名与厂商 SDK 非常接近。大多数调用点只需更改接收器类型并移除客户端构造样板代码。

### 从其他多后端库迁移

VectorX 与其他通用向量 DB 抽象层的不同之处在于它**不**定义统一的 `VectorDB` 接口。每个后端的语义（Milvus 分区 vs Qdrant 集合 vs Weaviate 类）都原生暴露。迁移包括：

1. 用 VectorX 的 YAML 加载器替换配置加载
2. 用 `Get*("<name>")` 调用替换客户端构造
3. 用直接的适配器方法调用替换统一接口的方法调用

---

## 测试策略

### 单元测试

```bash
make test          # go test ./...
make test-race     # go test -race ./...
make cover         # go test -coverprofile=cover.out ./...
```

| 测试领域 | 覆盖范围 | 覆盖率 |
|---|---|---|
| 配置加载和验证 | YAML 解析、字段同构、默认值、边界情况 | ~89% |
| 可观测性初始化 | 无操作路径、错误传播、nil 安全、采样器决策矩阵 | ~84% |
| 懒加载单例 | 首次调用构造、缓存命中、`Must*` panic 值、并发访问、`Close` 缓存驱逐 | — |
| 适配器方法包装（milvusx / qdrantx） | 通过接口嵌入假对象（fake）单元测试每个包装方法（`Search` / `Insert` / `Upsert` / 24 种 qdrantx 方法 / 等），含 span 名、int + string 属性、错误记录断言 | milvusx ~75% / qdrantx ~89% |
| 适配器方法包装（weaviatex） | 通过 `weaviateOps` 接口层注入假对象，单元测试 5 种包装方法（`GraphQLRaw` / `CreateObject` / `DeleteObject` / `CreateClass` / `IsLive`） | ~77% |
| 顶层运行时生命周期 | Init 成功/失败、访问器错误传播、`Close` 级联到适配器缓存、`Close` 幂等性、跨 `CloseAll` 错误聚合、OTel closer 钩子 LIFO 执行、closer 错误传播 | ~77% |
| Close TOCTOU 回归 | 100 个并发访问器 + `Close` | — |
| 静态解耦不变式 | 适配器包不得导入顶层 `vectorx`；`observability/` 不得导入适配器或 redis/kafka SDK | — |
| 导出器单元测试 | 每个支持的 exporter 别名、TLS 开/关、header 传递、录制型 `SpanPublisher` + 同步 span 处理器 | ~96% |

### 集成测试

集成测试需要真实的向量数据库容器：

```bash
cd example && docker compose up -d
INTEGRATION=1 go test -tags integration ./example/...
```

测试使用构建标签 `//go:build integration` 并通过 `INTEGRATION=1` 环境变量控制：

| 测试 | 导出器 | 验证内容 |
|---|---|---|
| `TestRuntime_OTLP_Milvus` | OTLP gRPC | 带 span 记录的 `HasCollection` |
| `TestRuntime_OTLP_Qdrant` | OTLP gRPC | 带 span 记录的 `Search` |
| `TestRuntime_OTLP_Weaviate` | OTLP gRPC | 带 span 记录的 `IsLive` |
| `TestExporter_RedisStream_PublishesSpan` | Redis Stream | 传递给 `SpanPublisher` 的 span 记录 |
| `TestExporter_KafkaTopic_PublishesSpan` | Kafka Topic | 传递给 `SpanPublisher` 的 span 记录 |

### 静态分析与 Lint

```bash
make lint       # golangci-lint run（gosec、govet、staticcheck）
make validate   # build + vet + race 测试 + 静态解耦检查
```

---

## 项目结构

```
vectorx/                          # 模块根（github.com/gospacex/vectorx）
├── vectorx.go                    # 顶层包：Init / MustInit / Runtime
├── vectorx_test.go               # Runtime 访问器和生命周期单元测试
├── vectorx_close_test.go         # TOCTOU 竞态、幂等关闭测试
├── vectorx_deps_test.go          # 静态依赖约束测试
├── config/                       # YAML 配置
│   ├── load.go                   # Load() — 配置结构体定义
│   ├── tracing.go                # TracingConfig 类型别名（mqx 同构）
│   ├── milvus.go / qdrant.go / weaviate.go  # 每个适配器的配置结构体
│   └── *_test.go                 # 配置解析、同构性、验证
├── milvusx/                      # Milvus 适配器
│   ├── milvusx.go                # GetMilvus / MustGetMilvus + sync.Map 缓存
│   ├── client.go                 # Milvusx 结构体，Search / Insert / HasCollection...
│   ├── config.go                 # 配置加载
│   └── tracing.go                # 可观测性的空导入
├── qdrantx/                      # Qdrant 适配器
│   ├── qdrantx.go                # GetQdrant / MustGetQdrant
│   ├── client.go                 # Qdrantx 结构体，Upsert / Search / Delete
│   └── config.go                 # 配置加载
├── weaviatex/                    # Weaviate 适配器
│   ├── weaviatex.go              # GetWeaviate / MustGetWeaviate
│   ├── client.go                 # Weaviatex 结构体，GraphQLRaw / CreateObject...
│   └── config.go                 # 配置加载
├── observability/                 # OpenTelemetry 集成
│   ├── tracing.go                # InitTracing / StartSpan
│   ├── tracer.go                 # Tracer 名称、传播器
│   ├── metrics.go                # Prometheus 计数器和直方图
│   └── exporter/                 # 可插拔 span 导出器
│       ├── exporter.go           # Build()、SpanPublisher 接口
│       ├── jaeger.go             # OTLP gRPC/HTTP 导出器构建器
│       ├── redis.go              # Redis Stream 导出器构建器
│       └── kafka.go              # Kafka Topic 导出器构建器
├── example/                      # 构建标签集成测试
│   ├── docker-compose.yml        # Milvus + Qdrant + Weaviate 容器
│   ├── milvusx_test/             # Milvus e2e 测试
│   ├── qdrantx_test/             # Qdrant e2e 测试
│   ├── weaviatex_test/           # Weaviate e2e 测试
│   └── vectorx_runtime_test/     # Runtime e2e 测试
├── utils/                        # 共享工具（doc.go 占位符）
├── go.mod                        # 模块定义
├── go.sum                        # 依赖校验和
├── Makefile                      # 构建 / 测试 / lint / 验证
├── README.md                     # 英文文档
└── EXAMPLES.md                   # 使用模式（RAG、多租户等）
```

---

## Makefile 参考

| 目标 | 命令 | 说明 |
|---|---|---|
| `build` | `go build ./...` | 编译所有包 |
| `test` | `go test ./...` | 运行单元测试 |
| `test-race` | `go test -race ./...` | 使用竞态检测器运行单元测试 |
| `cover` | `go test -coverprofile=cover.out ./...` | 运行测试并输出覆盖率 |
| `lint` | `golangci-lint run` | 运行 Go linter（gosec、govet、staticcheck） |
| `validate` | `build + vet + race + deps` | 提交前 / CI 关卡 |

---

## 设计决策与原理

| 决策 | 原理 |
|---|---|
| 按名称懒加载单例 | `sync.Map` + 每个键的互斥锁支持多个命名实例，不像 `sync.Once` 是全局的 |
| 可观测性作为独立包 | 适配器导入它；它不导入适配器的任何内容。强制单向依赖，实现 sidecar 复用 |
| 每个适配器薄封装 | 无统一 `VectorDB` 接口 —— 每个后端的语义不同（Milvus 分区 vs Qdrant 集合 vs Weaviate 类）。统一接口会有损或变成最低公共分母 |
| 配置字段与 mqx 同构 | mqx 和 vectorx 共享相同的 `TracingConfig` 结构体；基于反射的测试在编译时捕获漂移 |
| 用于 redis/kafka 的 `SpanPublisher` 接口 | 避免在 `observability` 中导入 redis/kafka SDK；应用程序在启动时注入发布器 |
| 集成测试由构建标签控制 | `//go:build integration` + `INTEGRATION=1` —— 对正常 `go test ./...` 零影响 |
| 顶层 `vectorx` 包是薄外观层 | 无客户端状态；每个适配器的 `sync.Map` 单例仍然是唯一事实来源 |
| 关闭闸门使用 `sync.RWMutex` 而非 `atomic.Bool` | `atomic.Bool` 是快照；`sync.RWMutex` 阻塞 `Close` 直到每个正在进行的访问器返回，防止 TOCTOU |
| YAML 配置（非 Viper/koanf） | 共享结构体 + 基于反射的测试在编译时捕获漂移；通用配置库会让两个项目无声地分叉 |

---

## 竞品对比

| 项目 | 范围 | 追踪 | 多后端 | 懒加载单例 | OTel 原生 |
|---|---|---|---|---|---|
| **vectorx** | Milvus / Qdrant / Weaviate 的 Go SDK | OTLP + Redis + Kafka（通过接口） | ✓ | ✓ | ✓ |
| [milvus-sdk-go](https://github.com/milvus-io/milvus-sdk-go) | 仅 Milvus | 手动 | ✗ | ✗ | 可选 |
| [qdrant/go-client](https://github.com/qdrant/go-client) | 仅 Qdrant | 手动 | ✗ | ✗ | 可选 |
| [weaviate-go-client](https://github.com/weaviate/weaviate-go-client) | 仅 Weaviate | 手动 | ✗ | ✗ | 可选 |
| [chroma-go](https://github.com/amikos-tech/chroma-go) | Chroma | 手动 | ✗ | ✗ | ✗ |

**选择 vectorx 当** 你需要 (a) 一个与多个向量后端通信的单一 Go 二进制文件，(b) 开箱即用的 OpenTelemetry 追踪，以及 (c) 懒加载的命名实例（例如，每个租户或每个环境）。对于单个后端、单个语言的项目，厂商 SDK 可能就足够了。

---

## 治理与贡献

### 发布流程

| 步骤 | 操作 | 责任人 |
|---|---|---|
| 1 | 在 `main` 上进行功能开发（线性历史） | 贡献者 |
| 2 | `make validate` 通过 | CI |
| 3 | `govulncheck ./...` 通过 | CI |
| 4 | 代码审查（至少 1 名维护者批准） | 维护者 |
| 5 | 标记 `vX.Y.Z`（SemVer） | 维护者 |
| 6 | 在发布说明中更新变更日志 | 维护者 |
| 7 | 创建 GitHub Release 并附带制品 | 自动化 |

### 贡献指南

1. Fork 仓库
2. 创建功能分支（`git checkout -b feat/my-feature`）
3. 提交前运行 `make validate`
4. 为任何新功能添加或更新测试 —— 受影响的包覆盖率必须保持 ≥ 80%
5. 对于新的 SDK 适配器，遵循 `milvusx/` 模式作为模板
6. 确保 `observability/` 的[静态解耦不变式](#静态解耦不变式)成立

**代码风格：** CI 强制执行 `gofmt` + `go vet` + `go test -race`。

**提交约定：** [Conventional Commits](https://www.conventionalcommits.org/)：

```
feat(milvusx): add CreatePartition support
fix(qdrantx): race in concurrent upsert
docs: update migration guide
```

### 行为准则

本项目遵守[贡献者公约](https://www.contributor-covenant.org/)行为准则。参与即表示您同意为所有人维护一个无骚扰的环境。

---

## 路线图

| 版本 | 主题 | 内容 |
|---|---|---|
| v1.1 | 连接池 | 每个适配器的 `MaxOpenConns`、`IdleTimeout`、健康检查 Ping |
| v1.2 | 弹性 | 针对瞬时 gRPC 错误的内置重试/熔断器中间件 |
| v1.3 | 更多后端 | pgvector 适配器（ParadeDB、Apache AGE 风格混合搜索） |
| v1.4 | 批量操作 | `BulkInsert(ctx, collection, embeddings, metadata)` 用于 >10k 向量 |
| v2.0 | 生成的存根 | 移除 `replace` 指令；使用上游 protos 生成的 gRPC 存根 |

以上项目是计划，而非承诺。参见 [GitHub issues](https://github.com/gospacex/vectorx/issues) 了解实时积压。

---

## 支持与 FAQ

**我在哪里提问？**
- 使用/设计问题：[GitHub Discussions](https://github.com/gospacex/vectorx/discussions)
- 错误报告/功能请求：[GitHub Issues](https://github.com/gospacex/vectorx/issues)
- 安全漏洞：`security@gospacex.com`（私密）

**为什么没有厂商 SDK 抽象层（`VectorDB` 接口）？** 每个后端有不同的语义（Milvus 分区 vs Qdrant 集合 vs Weaviate 类）。统一接口会是有损的或最低公共分母。更好的做法是暴露惯用的每个适配器方法。

**为什么是 YAML 配置而不是 Viper / koanf？** `vectorx:` 块与同一文件中 `mqx:` 块字段同构。共享结构体 + 基于反射的测试在编译时捕获漂移；通用配置库会让两个项目无声地分叉。

**我可以在没有顶层包的情况下使用 VectorX 吗？** 可以 —— 直接导入适配器包（例如 `github.com/gospacex/vectorx/milvusx`）并自己调用 `SetConfigPath` / `GetMilvus`。顶层包是一个便利设施。

**生产支持模式是什么？** 通过 GitHub 的社区支持。企业支持协议可用 —— 联系 `info@gospacex.com`。

**我如何对我的配置进行基准测试？** 运行 `go test -bench=BenchmarkGet -benchmem ./milvusx/ ./qdrantx/ ./weaviatex/` 测试单例访问延迟。对于端到端查询性能，使用你的应用程序针对厂商 SDK 的基准测试套件。

---

## 示例

完整的使用模式在 [EXAMPLES.md](EXAMPLES.md) 中有记录，包括：

- **RAG 管道** —— 嵌入 → 存储 → 搜索 → 生成
- **多租户隔离** —— 一个 Runtime，每个租户的命名实例
- **语义缓存** —— 基于 Qdrant 的查询缓存，带 TTL
- **优雅关闭** —— 信号处理 + context 超时 + Close
- **自定义导出器** —— 为自定义后端实现 `SpanPublisher`

---

## 许可证与致谢

**许可证：** 根据 **MIT 许可证**发布。完整文本请参见 [`LICENSE`](LICENSE)。

**致谢：** 感谢 [Milvus](https://milvus.io)、[Qdrant](https://qdrant.tech) 和 [Weaviate](https://weaviate.io) 团队提供的官方 Go SDK；感谢 [OpenTelemetry](https://opentelemetry.io) 项目提供的追踪 SDK 和 OTLP 线格式；以及 [mqx](https://github.com/gospacex/mqx) 的贡献者，其同构配置设计启发了这个 SDK。
