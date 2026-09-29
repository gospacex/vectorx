# Milvus 驱动功能矩阵

> 代码依据：`milvus.go`、`register.go`（vectorx/milvus）
> 更新日期：2026-09-09 | 驱动注册名：`milvus` | 客户端库：`github.com/milvus-io/milvus-sdk-go/v2`

## 1. 基础命令矩阵（CollectionClient 接口）

| 接口方法 | 底层 API | 状态 | 行为说明 |
|---|---|---|---|
| `CreateCollection(ctx, name, dim)` | `mc.CreateCollection()` + `CreateIndex()` | ✅ 支持 | 自动创建 AutoID Int64 PK 字段 + FloatVector embedding 字段 + 动态字段；建完索引自动建 Flat L2 index |
| `DropCollection(ctx, name)` | `mc.DropCollection()` | ✅ 支持 | — |
| `Insert(ctx, name, embeddings)` | `mc.Insert()` ColumnDynamic(JSONBytes) | ✅ 支持 | 批量插入，Fields 序列化为 JSONBytes 存入 Dynamic Field |
| `Search(ctx, name, query, limit)` | `mc.Search()` | ✅ 支持 | 返回 IDs/Distances/Fields；当前仅 L2 距离（未暴露 distance metric 配置） |
| `Delete(ctx, name, ids)` | `mc.DeleteByPks()` | ✅ 支持 | ID 字符串解析为 int64 PK；非 int64 ID 字符串会报错 |
| `HealthCheck()` | `mc.GetVersion()` | ✅ 支持 | — |
| `Close()` | `mc.Close()` | ✅ 支持 | — |
| `Driver()` | 返回 `"milvus"` | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `db.vector.{operation}` |
| `db.system` | `milvus` |
| `db.vector.collection` | collection 名称 |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| Milvus Standalone | ✅ 支持 | `cfg.Address=localhost:19530` |
| Milvus Cluster | ⚠️ 部分 | SDK 直连单个 Core 节点，无负载均衡 |
| Zilliz Cloud（公有云） | ✅ 支持 | 标准 Milvus API 兼容 |
| TLS | ❌ 不支持 | SDK 连接不暴露 TLSConfig |
| 多 Database 隔离 | ⚠️ 部分 | Milvus v2 支持 database 概念，驱动未封装 |

## 3. 配置字段生效状态（vectorx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Address` | ✅ 生效 | Milvus Core 地址，默认 `localhost:19530` |
| `Token` | ❌ 未使用 | 未透传至 SDK Auth，无用户名密码认证 |
| `Dim` | ✅ 生效 | CreateCollection 时作为 embedding 维度 |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| 自定义 Index 类型（IVF_FLAT/HNSW/SCANN） | ❌ 不支持 | 固定 Flat L2，无法切换 |
| 自定义 Distance Metric（IP/COSINE） | ❌ 不支持 | 固定 L2 |
| HyParam 参数配置 | ❌ 不支持 | 无法配置 hnsw/M 等参数 |
| Partition 分区 | ❌ 不支持 | 未暴露 PartitionKey |
| Load/Release Collection | ❌ 不支持 | 需在业务侧自行调用 SDK |
| Hybrid Search（标量+向量联合） | ⚠️ 部分 | 可通过 filter DSL 实现，驱动未封装 |
| Upsert（带 id 的 Insert） | ⚠️ 部分 | 需业务侧 Delete+Insert 模拟 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 图像/视频特征检索 | ✅ 完整支持 | 高维向量 + Flat Index |
| 文本语义检索（Embedding） | ✅ 完整支持 | — |
| RAG 知识库向量存储 | ✅ 完整支持 | Dynamic Field 存 metadata |
| 大批量向量导入 | ✅ 支持 | SDK 支持批量 Insert |
| 实时增量更新 | ✅ 支持 | Delete+Insert 模拟 Upsert |
| 大规模向量（亿级） | ⚠️ 部分 | Flat Index 搜索慢，需升级 HNSW/SCANN |
| 多租户隔离（Database） | ❌ 不支持 | 驱动未封装 Database 切换 |
| 健康探针 | ✅ 支持 | GetVersion |
| 链路追踪 | ✅ 支持 | `db.vector.*` span |

## 6. 已知限制与风险

1. **ID 仅支持 int64 字符串**：`parseIDs` 强制 `strconv.ParseInt`，业务侧传入 UUID 字符串会报错，需转换为数字 ID 或修改驱动。
2. **Index 类型固定为 Flat L2**：无法创建 HNSW/IVF_PQ 等高性能索引，大数据量搜索性能差。
3. **无 Auth 认证**：`Token` 配置字段未传递，无法对接需要用户名密码的 Milvus Enterprise。
4. **无 Database 隔离**：同一 Milvus 实例下多业务共用 default database，无法按 namespace 隔离。
5. **Upsert 无原生支持**：Milvus 暂不支持 Upsert，需业务侧自行 Delete+Insert。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/vectorx"
    _ "github.com/gospacex/vectorx/milvus" // 注册驱动
)

client, _ := vectorx.New(&vectorx.Config{
    Name: "default", Driver: "milvus",
    Address: "localhost:19530",
    Dim: 768,
})
client.CreateCollection(ctx, "articles", 768)
client.Insert(ctx, "articles", []vectorx.Embedding{
    {ID: "1", Vector: vec, Fields: map[string]interface{}{"title": "hello"}},
})
result, _ := client.Search(ctx, "articles", query, 10)
```
