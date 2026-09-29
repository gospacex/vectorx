# Pinecone 驱动功能矩阵

> 代码依据：`pinecone.go`、`register.go`（vectorx/pinecone）
> 更新日期：2026-09-09 | 驱动注册名：`pinecone` | 客户端库：`github.com/pinecone-io/go-pinecone/pinecone`（注意：当前绑定的是 stub 版本）

## 1. 基础命令矩阵（CollectionClient 接口）

| 接口方法 | 底层 API | 状态 | 行为说明 |
|---|---|---|---|
| `CreateCollection(ctx, name, dim)` | `pc.Index(...).CreateIndex()` | ❌ 未实现 | 返回 `requires a configured Pinecone pod endpoint` 错误 |
| `DropCollection(ctx, name)` | `pc.Index(...).DeleteIndex()` | ❌ 未实现 | 返回 `not yet implemented` 错误 |
| `Insert(ctx, name, embeddings)` | `pc.Index(...).Upsert()` | ❌ 未实现 | 返回 `requires a configured Pinecone pod endpoint` 错误 |
| `Search(ctx, name, query, limit)` | `pc.Index(...).Query()` | ⚠️ Stub | 返回空 `SearchResult{}`，未调用真实 API |
| `Delete(ctx, name, ids)` | `pc.Index(...).Delete()` | ❌ 未实现 | 返回 `not yet implemented` 错误 |
| `HealthCheck()` | 无实际调用 | ✅ 支持 | 空实现，始终返回 `nil` |
| `Close()` | 无操作 | ✅ 支持 | — |
| `Driver()` | 返回 `"pinecone"` | ✅ 支持 | — |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| Pinecone Serverless | ❌ 未实现 | 驱动使用 stub，未接入 serverless index API |
| Pinecone Podded（专用集群） | ❌ 未实现 | 需要 gRPC PodClient，当前仅 stub |
| TLS | ⚠️ 部分 | SDK 内置 TLS，驱动未暴露配置项 |
| API Key 认证 | ✅ 支持 | `cfg.Token` 或 `PINECONE_API_KEY` 环境变量 |

## 3. 配置字段生效状态（vectorx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Token` | ✅ 生效 | 作为 API Key，也支持 `PINECONE_API_KEY` 环境变量回退 |
| `Address` | ⚠️ 部分 | 提示用于 pod gRPC 地址，但当前 stub 未真正使用 |
| `Dim` | ❌ 未使用 | CreateCollection 未实现，维度不参与实际创建 |
| `Namespace` | ❌ 未使用 | 未传递到 SDK Namespace 参数 |
| `EnableTrace` | ⚠️ 部分 | span 始终上报（stub 方法内） |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| 自动索引创建（Serverless） | ❌ 未实现 | Pinecone Serverless 需先创建 Index，驱动未封装 |
| 元数据过滤查询 | ❌ 未实现 | Search 未实现，Filter 参数未传入 |
| 稀疏向量（Sparse Values） | ❌ 未实现 | 当前仅 Dense Vector |
| 多 Index 路由 | ❌ 未实现 | 仅支持单个 Index |
| 增量同步（HTTP API） | ❌ 未实现 | — |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 说明 |
|---|---|---|
| Pinecone Serverless 向量检索 | ❌ 不支持 | 驱动为 stub，所有写操作返回错误 |
| Pinecone Podded 专用集群 | ❌ 不支持 | 同上 |
| 快速原型验证 | ✅ 支持 | HealthCheck 通过，可验证 API Key 有效 |

## 6. 已知限制与风险

1. **核心操作未实现**：`CreateCollection`/`Insert`/`Search`/`Delete` 均返回 stub 错误，当前版本**不可用于生产**。
2. **仅 SDK 创建成功**：`New()` 能成功创建 `pinecone.Client`，但后续 Index 操作全部失败。
3. **Search 返回空结果**：`Search` 返回空 `SearchResult{}` 而非错误，调用方可能误判为"无结果"。
4. **依赖 API Key 而非凭证**：仅支持 API Key 认证，不支持 gRPC Token 或 OAuth。
5. **无 Namespace 传递**：`cfg.Namespace` 未传入 SDK，无法在多 Namespace 场景下工作。

## 7. 快速接入示例

> ⚠️ **当前驱动为 Stub，以下为预期接口示例，核心操作尚未实现。**

```go
import (
    "github.com/gospacex/vectorx"
    _ "github.com/gospacex/vectorx/pinecone" // 注册驱动（Stub，核心操作未实现）
)

client, _ := vectorx.New(&vectorx.Config{
    Name: "default", Driver: "pinecone",
    Token: "YOUR_PINECONE_API_KEY",
    Dim: 768,
    // Address: "pod-endpoint.grpc.us-east-1.aws.pinecone.io:443", // Podded 模式需配置
})
// ⚠️ CreateCollection / Insert / Search / Delete 当前均返回错误
// 仅 HealthCheck 可用
```
