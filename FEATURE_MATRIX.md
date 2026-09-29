# vectorx 驱动功能矩阵

## 支持的向量数据库

| 驱动 | 包路径 | 状态 | 说明 |
|------|--------|------|------|
| **milvus** | `github.com/gospacex/vectorx/milvus` | ✅ 完整实现 | Zilliz Cloud / 自建 Milvus |
| **qdrant** | `github.com/gospacex/vectorx/qdrant` | ✅ 完整实现 | Qdrant 向量数据库 |
| **pgvector** | `github.com/gospacex/vectorx/pgvector` | ✅ 完整实现 | PostgreSQL + pgvector 扩展 |
| **pinecone** | `github.com/gospacex/vectorx/pinecone` | ⚠️ Stub | Pinecone 云向量数据库 |

## 功能对比

| 功能 | Milvus | Qdrant | Pgvector | Pinecone |
|------|--------|--------|----------|----------|
| **CreateCollection** | ✅ FLAT + L2 索引 | ✅ Cosine 距离 | ✅ ivfflat + cosine_ops | ⚠️ 需要 Pod 端点 |
| **DropCollection** | ✅ | ✅ | ✅ | ❌ 未实现 |
| **Insert (批量)** | ✅ 动态字段 | ✅ Payload 映射 | ✅ pgx.Batch | ⚠️ 需要 Pod 端点 |
| **Search** | ✅ L2 相似度 | ✅ Cosine 相似度 | ✅ `<->` 余弦距离 | ⚠️ 返回空结果 |
| **Delete** | ✅ int64 PK | ✅ UUID | ✅ TEXT PK | ❌ 未实现 |
| **HealthCheck** | ✅ GetVersion | ✅ HealthCheck | ✅ Ping | ⚠️ 验证 API Key |

## SDK 依赖

| 驱动 | SDK | 版本 | 连接方式 |
|------|-----|------|----------|
| milvus | `github.com/milvus-io/milvus-sdk-go/v2` | v2.4.2 | gRPC (`localhost:19530`) |
| qdrant | `github.com/qdrant/go-client` | v1.18.2 | gRPC (`localhost:6334`) |
| pgvector | `github.com/jackc/pgx/v5` + `github.com/pgvector/pgvector-go` | pgx v5 | TCP (`localhost:5432`) |
| pinecone | `github.com/pinecone-io/go-pinecone` | — | gRPC (云服务) |

## 特色能力

### Milvus
- **动态字段存储**：支持 Dynamic Field，无需预定义 schema
- **int64 自增主键**：使用 Milvus 原生自增 ID
- **多索引支持**：FLAT / IVF_FLAT / HNSW 等
- **Milvus v2.4+**：支持 Sparse Vector、Grouping Search

### Qdrant
- **Payload 映射**：支持 map[string]interface{} 直接存储
- **Cosine 相似度**：默认使用余弦距离
- **多向量支持**：支持命名向量
- **gRPC 高性能**：使用 gRPC 协议通信

### Pgvector
- **SQL 原生**：直接使用 SQL 查询，易于调试
- **JSONB 字段**：fields 使用 JSONB 类型存储
- **ivfflat 索引**：支持 ivfflat + cosine_ops
- **事务支持**：支持 pgx 事务操作

### Pinecone
- **云原生**：完全托管的向量数据库服务
- **API Key 认证**：支持 cfg.Token 或 PINECONE_API_KEY 环境变量
- **Stub 状态**：当前为占位实现，需接入完整 Pinecone gRPC API

## 配置示例

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
      dim: 128

    pinecone_main:
      driver: pinecone
      address: "https://index-xxx.svc.xxx.pinecone.io"
      token: "your-api-key"
      dim: 128
```

## Docker 环境

```bash
# 启动所有向量数据库
docker compose up -d

# 仅启动 Milvus
docker compose up -d milvus etcd minio

# 仅启动 Qdrant
docker compose up -d qdrant

# 仅启动 PostgreSQL + pgvector
docker compose up -d postgres
```

## 使用方式

```go
package main

import (
    "github.com/gospacex/vectorx"
    _ "github.com/gospacex/vectorx/milvus"   // 注册 milvus 驱动
    _ "github.com/gospacex/vectorx/qdrant"   // 注册 qdrant 驱动
    _ "github.com/gospacex/vectorx/pgvector" // 注册 pgvector 驱动
)

func main() {
    client, err := vectorx.New(&vectorx.Config{
        Driver:  "milvus", // 或 "qdrant", "pgvector"
        Address: "localhost:19530",
        Dim:     128,
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 创建集合
    _ = client.CreateCollection(ctx, "my_collection", 128)

    // 插入向量
    _ = client.Insert(ctx, "my_collection", []vectorx.Embedding{
        {ID: "doc1", Vector: vectorx.Vector{0.1, 0.2, ...}},
    })

    // 搜索
    result, _ := client.Search(ctx, "my_collection", query, 10)
}
```
