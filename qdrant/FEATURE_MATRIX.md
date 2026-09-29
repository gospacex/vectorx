# Qdrant 驱动功能矩阵

> 代码依据：`qdrant.go`、`register.go`（vectorx/qdrant）
> 更新日期：2026-09-09 | 驱动注册名：`qdrant` | 客户端库：`github.com/qdrant/go-client/qdrant`

## 1. 基础命令矩阵（CollectionClient 接口）

| 接口方法 | 底层 API | 状态 | 行为说明 |
|---|---|---|---|
| `CreateCollection(ctx, name, dim)` | `client.CreateCollection()` | ✅ 支持 | 创建 Collection + VectorParams(Size=dim, Distance=Cosine) |
| `DropCollection(ctx, name)` | `client.DeleteCollection()` | ✅ 支持 | — |
| `Insert(ctx, name, embeddings)` | `client.Upsert()` | ✅ 支持 | Upsert 语义；ID 以 UUID 字符串形式写入 |
| `Search(ctx, name, query, limit)` | `client.GetPointsClient().Search()` | ✅ 支持 | 返回 SearchPoints；WithPayload=true 带 payload |
| `Delete(ctx, name, ids)` | `client.Delete()` PointsSelector_Points | ✅ 支持 | UUID PointId 删除 |
| `HealthCheck()` | `client.HealthCheck()` | ✅ 支持 | — |
| `Close()` | `client.Close()` | ✅ 支持 | — |
| `Driver()` | 返回 `"qdrant"` | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `db.vector.{operation}` |
| `db.system` | `qdrant` |
| `db.vector.collection` | collection 名称 |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| Qdrant 单机 | ✅ 支持 | 默认 `localhost:6334` |
| Qdrant Cluster | ⚠️ 部分 | SDK 支持多节点，但驱动仅接受单地址 `cfg.Address` |
| Qdrant Cloud | ✅ 支持 | 配置 Cloud URL + API Key（通过 Token 字段） |
| TLS | ⚠️ 部分 | go-client 支持 TLS，但驱动未暴露 TLSConfig 配置 |
| API Key 认证 | ⚠️ 部分 | 仅支持 Cloud 模式，集群内部认证未封装 |

## 3. 配置字段生效状态（vectorx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Address` | ✅ 生效 | host:port，默认 `localhost:6334` |
| `Token` | ❌ 未使用 | 未传递给 go-client，API Key 需通过环境变量或 header 方式传入 |
| `Namespace` | ⚠️ 部分 | 定义但未使用于 collection name 前缀 |
| `Dim` | ✅ 生效 | CreateCollection 向量维度 |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| 自定义距离度量（L2/IP/Cosine） | ❌ 不支持 | 固定 Cosine |
| 自定义 Index 类型（Product Quantization 等） | ❌ 不支持 | 使用默认 HNSW 图索引 |
| 过滤查询（Filter/WHERE） | ❌ 不支持 | Search 接口未传 Filter 参数 |
| Batch Upsert 分片 | ⚠️ 部分 | 直接传入全部 embeddings，无内置分片 |
| Scroll（全量遍历） | ❌ 不支持 | 未封装 Scroll API |
| 快照/备份 | ❌ 不支持 | — |
| 命名空间（Qdrant Namespace） | ❌ 不支持 | 未使用 Namespace 配置 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 图像/视频特征检索 | ✅ 完整支持 | Cosine 距离 + HNSW |
| 文本语义检索（Embedding） | ✅ 完整支持 | — |
| RAG 知识库向量存储 | ✅ 完整支持 | Payload 存 metadata |
| 大规模向量（千万级） | ✅ 完整支持 | HNSW 索引 + 集群部署 |
| 实时增量更新 | ✅ 完整支持 | Upsert 语义 |
| 过滤向量搜索（带条件） | ⚠️ 部分 | 需业务侧扩展 Search 接口传 Filter |
| 多租户隔离 | ⚠️ 部分 | 通过不同 Collection 名隔离，Namespace 未使用 |
| 健康探针 | ✅ 支持 | HealthCheck RPC |
| 链路追踪 | ✅ 支持 | `db.vector.*` span |

## 6. 已知限制与风险

1. **距离度量固定 Cosine**：无法选择 L2 或 IP，需在业务侧确认 Cosine 符合需求。
2. **Token 未传递**：`cfg.Token` 字段被忽略，Qdrant Cloud API Key 无法通过 Config 传入，需走环境变量或手动设置 Header。
3. **无过滤查询**：Search 接口未传入 Filter，无法做"标签+向量"混合检索，如需过滤需扩展驱动。
4. **单节点地址**：`cfg.Address` 仅支持单个地址，集群模式需业务侧自行选主节点连接。
5. **Name→Collection 名称映射**：`cfg.Name` 不直接用于 collection 名，collection 名由 `CreateCollection` 调用时传入。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/vectorx"
    _ "github.com/gospacex/vectorx/qdrant" // 注册驱动
)

client, _ := vectorx.New(&vectorx.Config{
    Name: "default", Driver: "qdrant",
    Address: "localhost:6334",
    Dim: 768,
})
client.CreateCollection(ctx, "docs", 768)
client.Insert(ctx, "docs", []vectorx.Embedding{
    {ID: "doc-001", Vector: vec, Fields: map[string]interface{}{"source": "web"}},
})
result, _ := client.Search(ctx, "docs", query, 10)
```
