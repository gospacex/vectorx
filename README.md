<div align="center">

# vectorx

**统一向量数据库 SDK：一套 `CollectionClient` 接口，多个可插拔后端驱动**

A unified Go SDK for vector databases — one `CollectionClient` interface with pluggable backends for Milvus, Qdrant, pgvector, Pinecone and Elasticsearch.

[![Go](https://img.shields.io/badge/Go-1.25.0-00ADD8?logo=go&logoColor=white)](https://go.dev/)
![License](https://img.shields.io/badge/License-MIT-green.svg)

</div>

---

## 目录

- [1. 定位与设计边界](#1-定位与设计边界)
- [2. 核心特性](#2-核心特性)
- [3. 架构](#3-架构)
- [4. 快速开始](#4-快速开始)
- [5. 配置参考](#5-配置参考)
- [6. API 参考](#6-api-参考)
- [7. 后端矩阵](#7-后端矩阵)
- [8. 可观测性](#8-可观测性)
- [9. 示例与文档](#9-示例与文档)
- [10. 测试与质量](#10-测试与质量)
- [11. 版本与兼容性](#11-版本与兼容性)
- [12. 许可证](#12-许可证)

---

## 1. 定位与设计边界

### 解决什么问题

- **统一抽象**：把不同向量数据库的建集合、写入、检索、删除收敛到一个 `CollectionClient` 接口（`interface.go`），业务代码面向接口编程，切换后端只改配置与一行 blank import。
- **驱动即插即用**：每个驱动包通过 `init()` 调用 `vectorx.RegisterCreator` 自注册，根模块不感知任何具体后端依赖，按需引入。
- **统一配置**：与 `configx` 共享同一配置源，`vectorx.stores` 下可声明多个命名实例，`ReadAndCreate(name)` 按名取用。
- **统一日志与链路**：驱动内部使用 `logx` 输出结构化日志，五个驱动的 CRUD 操作都会上报 `db.vector.*` OpenTelemetry span。

### 不解决什么问题

- **不是向量数据库本身**：不实现索引、副本、调度，仅封装各后端官方客户端。
- **不是 ORM / 查询构建器**：`Search` 只有 `name + query + limit` 三个参数，不支持元数据过滤、混合检索、分页、聚合；`SearchResult` 只返回 `IDs / Distances / Fields`。
- **不做连接池与重试策略**：`pgvector` 使用单条 `pgx.Conn`，`milvus` / `qdrant` / `elasticsearch` 均为单客户端实例；配置结构体里的连接池字段当前不参与建连。
- **不保证驱动能力对等**：各驱动实现状态不同（完整 / 部分 / Stub），见 [§7 后端矩阵](#7-后端矩阵)。

### 与同系列 SDK 的关系

| 依赖 | 用途 | vectorx 中的落点 |
|---|---|---|
| `github.com/gospacex/configx` | 统一配置读取 | `ReadConfig` / `ReadAndCreate` / `ReadStoreConfig` 等 |
| `github.com/gospacex/logx` | 统一结构化日志 | 各驱动 `logx.GetGlobal()` + `*Context` 方法 |
| `github.com/gospacex/traceflag` | span 开关（`EnableTrace`） | milvus / qdrant / pgvector / pinecone 的 `traceflag.NewTracer` |
| `go.opentelemetry.io/otel` | Span 上报 | 五个驱动统一的 `db.vector.*` span |

---

## 2. 核心特性

1. **一套接口覆盖五个后端**：`CollectionClient` 定义 8 个方法，Milvus / Qdrant / pgvector / Pinecone / Elasticsearch 各自实现。
2. **blank import 决定可用驱动**：`_ "github.com/gospacex/vectorx/qdrant"` 即完成注册；未导入时 `New` 返回 `driver "xxx" not registered`。
3. **驱动为独立 Go 模块**：`milvus` / `qdrant` / `pgvector` / `pinecone` 各有 `go.mod`，只拉用到的后端 SDK；`elasticsearch` 归属根模块。
4. **configx 多实例配置**：`vectorx.stores.<name>` 声明多个存储，按名字创建客户端。
5. **结构化日志 + OTel 链路**：`logx` 日志与 `db.vector.create_collection` / `insert` / `search` / `delete` / `drop_collection` span 打通。
6. **驱动级配置结构体**：每个驱动导出 `XxxStandaloneConfig` / `XxxClusterConfig`，覆盖地址、连接池、超时、TLS/SSL 等键。
7. **本地开发环境一键起**：根目录与各驱动的 `docker-compose.yml` + `scripts/{start,stop,verify}.sh`。
8. **实现状态如实标注**：完整 / 部分 / Stub 以源码为准，见 §7。

---

## 3. 架构

```
业务代码
   │  vectorx.New / ReadAndCreate / MustReadAndCreate / ReadConfigAndCreate*
   ▼
┌──────────────────────────────────────────────────────────────────────┐
│ vectorx 根模块  github.com/gospacex/vectorx                          │
│  interface.go ─ CollectionClient 接口 + Vector/Embedding/SearchResult│
│               ─ Config / VectorDBSection + RegisterCreator 工厂表    │
│  config.go    ─ configx 读取辅助（ReadStoreConfig 等）                │
└──────────────────────────────────────────────────────────────────────┘
   │  cfg.Driver → creators 注册表查找
   ▼
┌──────────────────────────────────────────────────────────────────────┐
│ 驱动层（init() 自注册；四个子模块各自 go.mod）                        │
│  vectorx/milvus         → github.com/milvus-io/milvus-sdk-go/v2      │
│  vectorx/qdrant         → github.com/qdrant/go-client                │
│  vectorx/pgvector       → jackc/pgx/v5 + pgvector/pgvector-go        │
│  vectorx/pinecone       → github.com/pinecone-io/go-pinecone         │
│  vectorx/elasticsearch  → elastic/go-elasticsearch/v9（根模块内）    │
└──────────────────────────────────────────────────────────────────────┘
   │  gRPC / TCP / HTTP
   ▼
向量数据库实例（自建）或云服务（Pinecone / Zilliz Cloud）
```

调用链约定：

- `New(cfg)` 先执行 `cfg.FillDefaults()`，再按 `cfg.Driver` 查注册表。
- 每个驱动的 `New` 校验 `cfg.Driver` 必须等于自身注册名（如 `milvus.New` 要求 `driver == "milvus"`）。
- 所有驱动方法接收 `context.Context`，span 在方法内创建并 `defer span.End()`。

---

## 4. 快速开始

### 4.1 安装

```bash
go get github.com/gospacex/vectorx
go get github.com/gospacex/vectorx/qdrant   # 按需选择驱动子模块
```

> `milvus` / `qdrant` / `pgvector` / `pinecone` 是独立模块，需要单独 `go get`；`elasticsearch` 随根模块一起获取。

### 4.2 最小可用示例

先起一个后端（以 Qdrant 为例，也可以用 `milvus` / `postgres` 等服务）：

```bash
cd qdrant && bash scripts/start.sh && bash scripts/verify.sh
```

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gospacex/vectorx"
	_ "github.com/gospacex/vectorx/qdrant" // 引入即注册驱动
)

func main() {
	ctx := context.Background()

	client, err := vectorx.New(&vectorx.Config{
		Driver:  "qdrant",
		Address: "localhost:6334",
		Dim:     4,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	if err := client.HealthCheck(); err != nil {
		log.Fatal(err)
	}

	if err := client.CreateCollection(ctx, "demo", 4); err != nil {
		log.Fatal(err)
	}
	defer client.DropCollection(ctx, "demo")

	// Qdrant 以 UUID 作为点 ID
	err = client.Insert(ctx, "demo", []vectorx.Embedding{
		{ID: "550e8400-e29b-41d4-a716-446655440000", Vector: vectorx.Vector{1, 0, 0, 0},
			Fields: map[string]interface{}{"title": "hello"}},
		{ID: "550e8400-e29b-41d4-a716-446655440001", Vector: vectorx.Vector{0, 1, 0, 0},
			Fields: map[string]interface{}{"title": "world"}},
	})
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.Search(ctx, "demo", vectorx.Vector{1, 0, 0, 0}, 3)
	if err != nil {
		log.Fatal(err)
	}
	for i, id := range res.IDs {
		fmt.Printf("[%d] id=%s score=%.4f fields=%v\n", i, id, res.Distances[i], res.Fields[i])
	}

	if err := client.Delete(ctx, "demo", []string{"550e8400-e29b-41d4-a716-446655440000"}); err != nil {
		log.Fatal(err)
	}
}
```

### 4.3 进阶用法（配置文件方式）

`config.yaml`：

```yaml
vectorx:
  stores:
    default:
      driver: qdrant
      address: "localhost:6334"
      dim: 4
      enable_trace: true
```

```go
client, err := vectorx.ReadAndCreate("default", configx.ConfigOptions{
	ConfigFile: "config.yaml",
	ConfigType: "yaml",
})
if err != nil {
	log.Fatal(err)
}
defer client.Close()
```

同一份配置的其他取用方式：

```go
// 失败时向 stderr 打印错误并以退出码 1 结束进程
mustClient := vectorx.MustReadAndCreate("default", configx.ConfigOptions{
	ConfigFile: "config.yaml",
	ConfigType: "yaml",
})

// 已有 configx.Config 句柄时
cfg, _ := configx.ReadConfig(configx.ConfigOptions{ConfigFile: "config.yaml", ConfigType: "yaml"})
storeCfg, err := vectorx.ReadStoreConfig(cfg, "default") // 取单个 store 的 *Config
client2, err := vectorx.ReadConfigAndCreate(cfg, "default")
client3, err := vectorx.ReadConfigAndCreateWithOpts("default", configx.ConfigOptions{
	ConfigFile: "config.yaml",
	ConfigType: "yaml",
})
```

---

## 5. 配置参考

### 5.1 配置结构

配置根键为 `vectorx`，其下 `stores` 是以实例名为 key 的映射（`VectorDBSection.Stores map[string]*Config`）。

```yaml
vectorx:
  stores:
    milvus_main:
      driver: milvus
      address: "localhost:19530"
      dim: 128
      enable_trace: true

    qdrant_main:
      driver: qdrant
      address: "localhost:6334"
      dim: 128

    pgvector_main:
      driver: pgvector
      address: "postgres://postgres:postgres@localhost:5432/vectorx?sslmode=disable"
      namespace: "public"
      dim: 128

    pinecone_main:
      driver: pinecone
      token: "${PINECONE_API_KEY}"
      namespace: "example-namespace"
      dim: 128

    es_main:
      driver: elasticsearch
      address: "http://localhost:9200"
      dim: 128
```

### 5.2 `vectorx.Config` 字段

结构体定义见 `interface.go`。

| 字段 | YAML 键 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `Driver` | `driver` | string | 是 | — | 驱动注册名：`milvus` / `qdrant` / `pgvector` / `pinecone` / `elasticsearch` |
| `Name` | `name` | string | 否 | `default` | 实例名；`ReadAndCreate` / `ConfigFromYAML` 会用配置键名覆盖该字段 |
| `Mode` | `mode` | string | 否 | — | 结构体注释为 `standalone \| cluster`；当前驱动均未读取 |
| `Address` | `address` | string | 否 | `localhost:19530` | 连接地址。**默认值由 `FillDefaults` 统一填充**，对 pgvector（应为 DSN）与 elasticsearch（应为 `http://host:9200`）并不适用，必须显式配置 |
| `Token` | `token` | string | 否 | — | pinecone：API Key（可回退环境变量 `PINECONE_API_KEY`）；elasticsearch：密码（用户名固定 `elastic`）；milvus / qdrant / pgvector 未读取 |
| `Namespace` | `namespace` | string | 否 | — | pgvector：schema（空则 `public`）；elasticsearch：bulk indexer 默认 index；qdrant / pinecone 未读取 |
| `Collection` | `collection` | string | 否 | — | 结构体注释标注「兼容 milvus」；当前驱动未读取，集合名以 `CreateCollection` 参数为准 |
| `Dim` | `dim` | int | 否 | `128` | 仅在 `FillDefaults` 中补默认值，**驱动不读取**；维度以 `CreateCollection(ctx, name, dim)` 的参数为准 |
| `EnableTrace` | `enable_trace` | bool | 否 | `false` | milvus / qdrant / pgvector / pinecone 传给 `traceflag.NewTracer`；elasticsearch 未读取该字段 |

`FillDefaults` 仅补齐三项：`Address` → `localhost:19530`、`Name` → `default`、`Dim` → `128`。

### 5.3 驱动专属配置结构体

各驱动包导出 `XxxStandaloneConfig` / `XxxClusterConfig`，Cluster 结构体内嵌对应 Standalone（`yaml:",inline"`），因此 Cluster 的 YAML 同时包含 Standalone 的键。

**注意**：这些结构体当前**不参与** `New` 的建连流程（驱动只读 `vectorx.Config`），是给需要按驱动维度独立解析配置的业务方使用的类型。

| 结构体 | 包路径 | yaml 键（逐一对照结构体 tag） |
|---|---|---|
| `MilvusStandaloneConfig` | `vectorx/milvus` | `address`、`token`、`dim`、`timeout` |
| `MilvusClusterConfig` | `vectorx/milvus` | 继承上行 + `max_open_conns`、`max_idle_conns`、`conn_max_lifetime`、`read_timeout`、`write_timeout`、`max_retries`、`retry_interval`、`tls_enabled`、`tls_cert_file`、`tls_key_file`、`tls_ca_file` |
| `QdrantStandaloneConfig` | `vectorx/qdrant` | `address`、`token`、`dim`、`timeout`、`tls_enabled` |
| `QdrantClusterConfig` | `vectorx/qdrant` | 继承上行 + `max_open_conns`、`max_idle_conns`、`conn_max_lifetime`、`read_timeout`、`write_timeout`、`max_retries`、`retry_interval`、`tls_cert_file`、`tls_key_file`、`tls_ca_file`、`preferred_node` |
| `PgvectorStandaloneConfig` | `vectorx/pgvector` | `dsn`、`host`、`port`、`user`、`password`、`dbname`、`dim`、`timeout` |
| `PgvectorClusterConfig` | `vectorx/pgvector` | 继承上行 + `max_open_conns`、`max_idle_conns`、`conn_max_lifetime`、`read_timeout`、`write_timeout`、`max_retries`、`retry_interval`、`sslmode`、`ssl_cert_file`、`ssl_key_file`、`ssl_root_cert` |
| `PineconeStandaloneConfig` | `vectorx/pinecone` | `api_key`、`index_name`、`namespace`、`host`、`dim`、`timeout` |
| `PineconeClusterConfig` | `vectorx/pinecone` | 继承上行 + `max_retries`、`retry_interval`、`read_timeout`、`write_timeout` |
| `ElasticsearchStandaloneConfig` | `vectorx/elasticsearch` | `address`、`token`、`dim`、`timeout` |
| `ElasticsearchClusterConfig` | `vectorx/elasticsearch` | 继承上行 + `max_open_conns`、`max_idle_conns`、`conn_max_lifetime`、`read_timeout`、`write_timeout`、`max_retries`、`retry_interval`、`tls_enabled`、`tls_cert_file`、`tls_key_file`、`tls_ca_file` |

---

## 6. API 参考

### 6.1 核心接口

```go
type CollectionClient interface {
	CreateCollection(ctx context.Context, name string, dim int) error
	DropCollection(ctx context.Context, name string) error
	Insert(ctx context.Context, name string, embeddings []Embedding) error
	Search(ctx context.Context, name string, query Vector, limit int) (*SearchResult, error)
	Delete(ctx context.Context, name string, ids []string) error
	HealthCheck() error
	Close() error
	Driver() string
}
```

| 方法 | 语义 | 说明 |
|---|---|---|
| `CreateCollection` | 建集合（建表 / 建索引） | `dim` 为向量维度；各驱动的索引与距离度量见 §7 |
| `DropCollection` | 删除集合 | 对应删表 / 删索引 |
| `Insert` | 批量写入 | `Embedding.ID` 的类型约束因驱动而异（见 §7.3）；`Fields` 作为附加字段存储 |
| `Search` | Top-K 相似度检索 | `limit <= 0` 时各驱动内部回退为 `10`；不支持过滤条件 |
| `Delete` | 按 ID 删除 | 语义为删除，不存在的 ID 由后端决定是否报错 |
| `HealthCheck` | 连接探活 | 实现方式见 §7 |
| `Close` | 关闭连接 | 建议配合 `defer` |
| `Driver` | 返回驱动名 | 与注册名一致，如 `"qdrant"` |

### 6.2 数据类型

| 类型 | 定义 | 说明 |
|---|---|---|
| `Vector` | `[]float32` | 向量本体 |
| `Embedding` | `struct { ID string; Vector Vector; Fields map[string]interface{} }` | 待写入的文档，`Fields` 为附加元数据 |
| `SearchResult` | `struct { IDs []string; Distances []float32; Fields []map[string]interface{} }` | 三个切片按下标一一对应；`Distances` 为后端返回的分数 / 距离，语义随驱动不同 |
| `Config` | 见 §5.2 | 单个 store 的统一配置 |
| `VectorDBSection` | `struct { Stores map[string]*Config }` | 对应 `vectorx` 配置节 |

### 6.3 工厂与配置函数

| 函数 | 签名（节选） | 说明 |
|---|---|---|
| `RegisterCreator` | `func RegisterCreator(name string, creator func(*Config) (CollectionClient, error))` | 注册驱动创建函数，由各驱动 `init()` 调用 |
| `New` | `func New(cfg *Config) (CollectionClient, error)` | `FillDefaults` 后按 `cfg.Driver` 查表创建 |
| `ReadConfig` | `func ReadConfig(opts ...configx.ConfigOptions) error` | 触发 configx 配置加载，只返回错误（不返回配置句柄） |
| `ReadAndCreate` | `func ReadAndCreate(name string, opts ...configx.ConfigOptions) (CollectionClient, error)` | 从 `vectorx.stores` 取实例并创建 |
| `MustReadAndCreate` | `func MustReadAndCreate(name string, opts ...configx.ConfigOptions) CollectionClient` | 失败写 stderr 并 `os.Exit(1)` |
| `ReadStoreConfig` | `func ReadStoreConfig(cfg configx.Config, storeName string) (*Config, error)` | 从已有配置取单个 `*Config` |
| `ReadConfigAndCreate` | `func ReadConfigAndCreate(cfg configx.Config, storeName string) (CollectionClient, error)` | 用已有配置创建客户端 |
| `ReadConfigAndCreateWithOpts` | `func ReadConfigAndCreateWithOpts(storeName string, opts ...configx.ConfigOptions) (CollectionClient, error)` | 读配置 + 创建的组合封装 |
| `ConfigFromYAML` | `func ConfigFromYAML(data []byte) ([]*Config, error)` | 直接解析 YAML 字节为 `*Config` 列表 |
| `Duration` | `func Duration(start time.Time) time.Duration` | `time.Since` 的便捷封装 |
| `(*Config).FillDefaults` | `func (c *Config) FillDefaults()` | 补齐 `Address` / `Name` / `Dim` 默认值 |

### 6.4 驱动构造函数

| 函数 | 返回 | 前置条件 |
|---|---|---|
| `milvus.New(cfg *vectorx.Config)` | `*milvus.clientImpl`（未导出类型） | `cfg.Driver == "milvus"` |
| `qdrant.New(cfg *vectorx.Config)` | `*qdrant.clientImpl`（未导出类型） | `cfg.Driver == "qdrant"` |
| `pgvector.New(cfg *vectorx.Config)` | `*pgvector.clientImpl`（未导出类型） | `cfg.Driver == "pgvector"` |
| `pinecone.New(cfg *vectorx.Config)` | `*pinecone.clientImpl`（未导出类型） | `cfg.Driver == "pinecone"`，且 API Key 可用 |
| `elasticsearch.New(cfg *vectorx.Config)` | `*elasticsearch.clientImpl`（未导出类型） | `cfg.Driver == "elasticsearch"` |

> 返回值是未导出的具体类型，包外无法书写其类型名；推荐统一通过 `vectorx.New` / `ReadAndCreate` 拿到 `CollectionClient` 接口值。

### 6.5 elasticsearch 包的额外导出符号

| 符号 | 定义 | 说明 |
|---|---|---|
| `Similarity` | `type Similarity string` | 向量距离度量类型，`String()` 未知值回退 `cosine` |
| `SimilarityCosine` | `Similarity = "cosine"` | 建索引时实际使用（`CreateCollection` 的 mapping 写死 `SimilarityCosine`） |
| `SimilarityDotProduct` | `Similarity = "dot_product"` | 已定义，当前建索引未使用 |
| `SimilarityL2` | `Similarity = "l2_norm"` | 已定义，当前建索引未使用 |

---

## 7. 后端矩阵

> 本节结论全部以源码为准（`milvus/milvus.go`、`qdrant/qdrant.go`、`pgvector/pgvector.go`、`pinecone/pinecone.go`、`elasticsearch/elasticsearch.go`），`FEATURE_MATRIX.md` 仅作交叉验证，差异见 §7.4。

### 7.1 实现状态总览

| 驱动 | 包路径（import） | 模块归属 | 客户端 SDK | 状态 |
|---|---|---|---|---|
| **milvus** | `github.com/gospacex/vectorx/milvus` | 独立子模块 | `milvus-sdk-go/v2` v2.4.2 | **完整** — 8 个接口方法均真实调用 SDK |
| **qdrant** | `github.com/gospacex/vectorx/qdrant` | 独立子模块 | `qdrant/go-client` v1.18.2 | **完整** — 8 个接口方法均真实调用 SDK |
| **pgvector** | `github.com/gospacex/vectorx/pgvector` | 独立子模块 | `jackc/pgx/v5` v5.7.1 + `pgvector/pgvector-go` v0.2.0 | **完整** — 8 个接口方法均真实调用 SQL |
| **pinecone** | `github.com/gospacex/vectorx/pinecone` | 独立子模块 | `pinecone-io/go-pinecone` v1.1.1 | **Stub** — 仅 `New` 可用，写入 / 删除直接返回错误，检索返回空结果 |
| **elasticsearch** | `github.com/gospacex/vectorx/elasticsearch` | 根模块 | `elastic/go-elasticsearch/v9` v9.5.2 | **部分** — 8 个方法均有真实实现，但 `Search` 的请求体构造存在实现缺陷（见 §7.2） |

### 7.2 能力对照表

只列源码能直接证明的能力（✅ 已实现 / ⚠️ 部分或有缺陷 / ❌ 未实现）。

| 能力 | milvus | qdrant | pgvector | pinecone | elasticsearch |
|---|---|---|---|---|---|
| 驱动注册（`init` → `RegisterCreator`） | ✅ | ✅ | ✅ | ✅ | ✅ |
| 连接方式 / 默认地址 | ✅ gRPC `localhost:19530` | ✅ gRPC `localhost:6334` | ✅ pgx（DSN 须显式配置） | ✅ API Key（`Token` 或 `PINECONE_API_KEY`） | ✅ HTTP `http://localhost:9200` |
| 连接探活 | ✅ `GetVersion` | ✅ `HealthCheck()` | ✅ `Ping` | ⚠️ 空实现，恒返回 `nil` | ✅ `Cluster.Health` |
| 集合创建 | ✅ schema + `CreateIndex` | ✅ `CreateCollection` | ✅ `CREATE TABLE` + `CREATE INDEX` | ❌ 返回错误 | ✅ `Indices.Create`（已存在则跳过） |
| 集合删除 | ✅ `DropCollection` | ✅ `DeleteCollection` | ✅ `DROP TABLE` | ❌ 返回错误 | ✅ `Indices.Delete` |
| 批量写入 | ✅ `Insert`（动态字段 JSON） | ✅ `Upsert`（payload 映射） | ✅ `pgx.Batch` 批量 INSERT | ❌ 返回错误 | ✅ `esutil.BulkIndexer` |
| Top-K 检索 | ✅ `Search`（L2） | ✅ `Search`（Cosine 得分） | ✅ `ORDER BY ... <->` | ⚠️ 返回空 `SearchResult{}`，**不报错** | ⚠️ 请求已发出，但请求体构造有缺陷（见下） |
| 按 ID 删除 | ✅ `DeleteByPks`（ID 须为 int64） | ✅ 按 UUID 删点 | ✅ 逐条 `DELETE ... WHERE id=$1` | ❌ 返回错误 | ✅ `es.Delete`（容忍 404） |
| 过滤检索（条件 + 向量） | ❌ | ❌ | ❌ | ❌ | ⚠️ 仅硬编码固定 `filter`，不可配置 |
| 余弦相似度 | ❌ 固定 L2 索引 / L2 检索 | ✅ `Distance_Cosine` | ⚠️ 索引 `vector_cosine_ops`，检索算子为 `<->`（L2），二者不一致 | ❌ 未实现 | ✅ mapping `similarity: cosine` |
| 自定义距离度量 | ❌ 写死 L2 | ❌ 写死 Cosine | ❌ 写死 `cosine_ops` | ❌ 未实现 | ⚠️ 常量已导出，建索引写死 `SimilarityCosine` |
| OTel span（`db.vector.*`） | ✅ | ✅ | ✅ | ✅（stub 方法内） | ✅ |
| `logx` 结构化日志 | ✅ | ✅ | ✅ | ✅（仅 `New`） | ✅ |

**elasticsearch 检索缺陷（源码可证）**：`floatSliceToString` 只取查询向量的**第一个分量**并返回 `["<首元素>"]`，随后被 `fmt.Sprintf` 再包进一层方括号，最终 `query_vector` 形如 `[["1"]]`；同时拼入的 `filter` 为固定的 `{"bool":{"must_not":[{"match_all":{}}]}}`。按源码推断，多维向量检索无法按预期返回结果，需修复该函数后方可用于生产检索。

### 7.3 距离度量与 ID 约束

| 驱动 | 索引 / 检索度量（源码位置） | `Embedding.ID` 约束 | `Fields` 存储方式 |
|---|---|---|---|
| milvus | `entity.NewIndexFlat(entity.L2)` + 检索 `entity.L2` | 忽略（主键 AutoID Int64）；`Delete` 时必须可解析为 int64 | 动态字段（`ColumnDynamic` / JSONBytes） |
| qdrant | `qdrant.Distance_Cosine` | 必须是合法 UUID（`qdrant.NewIDUUID`） | payload（`map[string]*qdrant.Value`） |
| pgvector | 索引 `ivfflat (embedding vector_cosine_ops)`；检索 `embedding <-> $1` | 任意非空字符串（`TEXT PRIMARY KEY`） | `jsonb` 列 |
| pinecone | — | — | — |
| elasticsearch | mapping `"similarity": "cosine"`（`SimilarityCosine`） | 文档 `_id` | `fields` object 字段 |

### 7.4 与 `FEATURE_MATRIX.md` 的交叉核对

| 驱动 | 根目录 `FEATURE_MATRIX.md` 的结论 | 源码复核结论 | 是否一致 |
|---|---|---|---|
| milvus | ✅ 完整实现 | 完整实现；索引与检索均为 L2，动态字段 + AutoID 属实 | ✅ 一致 |
| qdrant | ✅ 完整实现 | 完整实现；Cosine、payload、UUID 删除属实 | ✅ 一致 |
| pgvector | ✅ 完整实现，`Search` 标注为「`<->` 余弦距离」 | 实现完整；但 `<->` 在 pgvector 中是 **L2 距离算子**，余弦算子为 `<=>`，而索引按 `vector_cosine_ops` 建立 —— 文案与算子语义不符 | ⚠️ 状态一致，度量描述不一致 |
| pinecone | ⚠️ Stub；`HealthCheck` 标注为「验证 API Key」 | Stub 属实；`HealthCheck` 源码为直接 `return nil`，**并未校验 API Key**（子目录 `pinecone/FEATURE_MATRIX.md` 的「空实现、恒返回 nil」描述与源码一致） | ⚠️ 状态一致，HealthCheck 描述不一致 |
| elasticsearch | 未收录 | 存在 `elasticsearch/` 包且已实现接口，属根模块 | ❌ 根矩阵缺失该驱动 |

补充核对（根矩阵「功能对比」表未覆盖的点）：

- 根矩阵称 pgvector `Search` 为「余弦距离」，与 §7.2 的算子结论相同地不成立。
- `cfg.Dim` / `cfg.Mode` / `cfg.Collection` 在根矩阵与子矩阵中部分被标为「生效」，但全仓源码中驱动只在 `FillDefaults` 里读写 `Dim`，`Mode` / `Collection` 无任何读取点。

---

## 8. 可观测性

### 日志

- 各驱动通过 `logx.GetGlobal()` 获取全局 `logx.Logger`，使用 `InfoContext` / `ErrorContext` 变体输出，字段用 `logx.String` / `logx.Int` / `logx.Err` 构造。
- 包内 blank import `_ "github.com/gospacex/logx/zap"` 注册 zap provider。
- 典型日志：`vectorx/milvus create collection started`、`qdrant collection created`、`pgvector insert done`、`vectorx/milvus search done` 等。

### Trace（OpenTelemetry）

- milvus / qdrant / pgvector / pinecone 使用 `traceflag.NewTracer(cfg.EnableTrace, "<驱动 module 路径>", "vectorx")`，由 `Config.EnableTrace` 控制；elasticsearch 直接 `otel.Tracer("github.com/gospacex/vectorx/elasticsearch")`，不读取 `EnableTrace`。
- 统一 span 名：

| span 名 | 附加属性（源码出现的） |
|---|---|
| `db.vector.create_collection` | `db.system`、`db.vector.collection` / `db.statement`、`db.vector.dim` / `db.rows_affected` |
| `db.vector.drop_collection` | `db.vector.collection` |
| `db.vector.insert` | `db.vector.collection`、`db.vector.insert_count` |
| `db.vector.search` | `db.vector.collection`、`db.vector.limit` |
| `db.vector.delete` | `db.vector.collection`、`db.vector.delete_count` |

- `db.system` 取值：`milvus` / `qdrant` / `postgresql` / `pinecone` / `elasticsearch`。
- 导出端点由调用方自行注入（各驱动的 example 使用 Jaeger collector `http://localhost:14268/api/traces`）。

### Metrics

当前源码未定义任何 metrics / Prometheus 埋点，不提供指标接口。

---

## 9. 示例与文档

### 可运行示例

每个驱动目录下都有 `examples/main.go` + `examples/config.yaml`，覆盖健康检查 → 建集合 → 写入 → 检索 → 删除 → 删集合的完整链路，并演示 Jaeger TracerProvider 初始化：

```bash
cd qdrant && docker compose up -d && go run ./examples/
```

| 驱动 | 示例 | 示例配置 |
|---|---|---|
| milvus | `milvus/examples/main.go` | `milvus/examples/config.yaml` |
| qdrant | `qdrant/examples/main.go` | `qdrant/examples/config.yaml` |
| pgvector | `pgvector/examples/main.go` | `pgvector/examples/config.yaml` |
| pinecone | `pinecone/examples/main.go`（无 `PINECONE_API_KEY` 时直接跳过） | `pinecone/examples/config.yaml` |
| elasticsearch | `elasticsearch/examples/main.go` | `elasticsearch/examples/config.yaml` |

### 本地环境脚本

| 目录 | 文件 | 作用 |
|---|---|---|
| `milvus/` `qdrant/` `pgvector/` `elasticsearch/` | `scripts/start.sh` | `docker compose up -d` 并轮询健康状态 |
| 同上 | `scripts/verify.sh` | 探活检查（milvus `:9091/healthz`、qdrant `:6333/healthz`、pgvector `pg_isready`、elasticsearch `:_cluster/health`） |
| 同上 | `scripts/stop.sh` | 停止容器 |
| `pinecone/` | 无 compose / scripts | 云服务，配置 API Key 即可 |

### docker-compose

| 文件 | 覆盖服务 |
|---|---|
| `docker-compose.yml`（根） | `milvus`、`qdrant`、`weaviate`、`etcd`、`minio`、`postgres`（网络 `hubx-vectorx-net`） |
| `milvus/docker-compose.yml` | `milvus` + `etcd` + `minio` |
| `qdrant/docker-compose.yml` | `qdrant`（6333/6334） |
| `pgvector/docker-compose.yml` | `pgvector/pgvector:pg16`（5432） |
| `elasticsearch/docker-compose.yml` | `elasticsearch` 8.12.1 + `kibana`（9200/5601） |

- 根 compose 中的 `weaviate` 仅为环境容器，**本 SDK 没有对应驱动包**。
- pgvector 扩展初始化：`pgvector-init.sh`（根，挂载为 `docker-entrypoint-initdb.d/init-vector.sh`）与 `pgvector/init.sql`。
- 根目录 `verify.sh` 期望 `examples/<driver>` 路径，与当前目录布局不符，请使用各驱动内的 `scripts/verify.sh`。

### 文档

| 文件 | 内容 |
|---|---|
| `FEATURE_MATRIX.md`（根） | 四驱动功能对比与配置示例 |
| `milvus/FEATURE_MATRIX.md`、`qdrant/FEATURE_MATRIX.md`、`pgvector/FEATURE_MATRIX.md`、`pinecone/FEATURE_MATRIX.md` | 各驱动逐方法实现矩阵、配置字段生效状态、已知限制 |
| `SDK_AUDIT.md` | 2026-09-07 的完成度审计记录（部分结论早于驱动拆分子模块，以当前源码为准） |

---

## 10. 测试与质量

仓库当前**没有 `_test.go` 单元测试**，质量门禁以编译、静态检查与可运行示例为准：

```bash
# 根模块（含 elasticsearch 驱动）
go build ./...
go vet ./...

# 各驱动子模块
cd milvus    && go build ./... && go vet ./...
cd qdrant    && go build ./... && go vet ./...
cd pgvector  && go build ./... && go vet ./...
cd pinecone  && go build ./... && go vet ./...

# 后端就绪后跑示例（每个驱动目录内）
docker compose up -d
bash scripts/verify.sh
go run ./examples/
```

以上命令在当前源码上均通过。

---

## 11. 版本与兼容性

| 项 | 值 |
|---|---|
| 根模块 | `github.com/gospacex/vectorx` |
| Go 版本 | `go 1.25.0`（根与 `milvus` / `qdrant` / `pgvector` / `pinecone` 四个子模块一致） |
| 驱动子模块 | `github.com/gospacex/vectorx/{milvus,qdrant,pgvector,pinecone}` |
| 当前 tag | 仓库尚未打 tag（`git tag` 为空） |
| 主要依赖 | `elastic/go-elasticsearch/v9`、`configx`、`logx`、`otel v1.43.0`、`yaml.v3` |

使用注意：

- 根模块 `go.mod` 的 `replace` 把 `configx` / `logx` / `traceflag` 指向同级本地目录，把四个驱动指向 `./<driver>`；在本地 monorepo 内可直接构建，对外发布前需替换为可解析的正式版本。
- 驱动子模块同样以 `replace github.com/gospacex/vectorx => ../` 依赖根模块。
- 后端版本要求（来自各 `docker-compose.yml`）：Milvus v2.4.0、Qdrant v1.12.1、PostgreSQL + pgvector（`pgvector/pgvector:pg16`）、Elasticsearch 8.12.1。

---

## 12. 许可证

MIT License
