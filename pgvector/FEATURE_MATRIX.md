# pgvector 驱动功能矩阵

> 代码依据：`pgvector.go`、`register.go`（vectorx/pgvector）
> 更新日期：2026-09-09 | 驱动注册名：`pgvector` | 客户端库：`github.com/jackc/pgx/v5` + `github.com/pgvector/pgvector-go`

## 1. 基础命令矩阵（CollectionClient 接口）

| 接口方法 | 底层 API | 状态 | 行为说明 |
|---|---|---|---|
| `CreateCollection(ctx, name, dim)` | `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX USING ivfflat` | ✅ 支持 | 建表（id TEXT PK / embedding vector(dim) / fields jsonb）+ ivfflat 索引（lists=100，cosine ops） |
| `DropCollection(ctx, name)` | `DROP TABLE IF EXISTS` | ✅ 支持 | — |
| `Insert(ctx, name, embeddings)` | `pgx.Batch` 批量 INSERT | ✅ 支持 | 批量发送，逐行 Exec；fields 序列化为 JSONB |
| `Search(ctx, name, query, limit)` | `<->` 操作符 + ORDER BY + LIMIT | ✅ 支持 | Cosine 距离排序，返回 distance（float64→float32） |
| `Delete(ctx, name, ids)` | `DELETE FROM ... WHERE id = $1` 循环 | ⚠️ 部分 | 逐条删除，无批量 DELETE |
| `HealthCheck()` | `conn.Ping()` | ✅ 支持 | — |
| `Close()` | `conn.Close()` | ✅ 支持 | — |
| `Driver()` | 返回 `"pgvector"` | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `db.vector.{operation}` |
| `db.system` | `postgresql` |
| `db.vector.collection` | collection 名称 |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| PostgreSQL + pgvector 插件 | ✅ 支持 | 标准单机部署 |
| 连接池 | ⚠️ 部分 | 单 `pgx.Conn`，无池化（pgxpool 未使用） |
| 主从复制读分离 | ❌ 不支持 | 单连接，无法读写分离 |
| TLS | ✅ 支持 | 通过 `cfg.Address` 连接字符串 `sslmode=require` 传递 |
| 多 Schema 隔离 | ✅ 支持 | `cfg.Namespace` 映射为 schema |

## 3. 配置字段生效状态（vectorx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Address` | ✅ 生效 | PostgreSQL 连接字符串，含 schema 信息 |
| `Namespace` | ✅ 生效 | 映射为 SQL schema，空值默认 `public` |
| `Dim` | ✅ 生效 | CREATE TABLE 时 embedding 维度 |
| `Token` | ❌ 未使用 | 未传递（pgx 凭据在 Address URI 中） |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| HNSW 索引 | ⚠️ 部分 | 默认 ivfflat，HNSW 需业务侧手动 `CREATE INDEX ... USING hnsw` |
| 距离度量切换（L2/IP） | ⚠️ 部分 | 当前固定 cosine_ops，切换需修改 SQL |
| 批量 DELETE | ❌ 不支持 | 当前逐条 DELETE，大批量删除慢 |
| Upsert（ON CONFLICT） | ❌ 不支持 | Insert 不处理重复键冲突 |
| 全文检索联合查询 | ✅ 支持 | pgvector 兼容 pg_trgm/tsvector，业务侧可扩展 |
| GIN/GiST 索引优化 | ⚠️ 部分 | ivfflat 固定 lists=100，无法调参 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 轻量级向量检索（中小规模） | ✅ 完整支持 | ivfflat 适合 <千万级向量 |
| 已有 PostgreSQL 基础设施复用 | ✅ 优势场景 | 无需额外服务部署 |
| 结构化+向量混合查询 | ✅ 完整支持 | SQL 能力强，可自由 JOIN |
| RAG 知识库向量存储 | ✅ 完整支持 | jsonb 存 metadata |
| 高并发写入 | ⚠️ 部分 | 无连接池，高并发需业务侧扩展 |
| 多租户 Schema 隔离 | ✅ 完整支持 | Namespace → schema |
| 亿级向量规模 | ❌ 不支持 | ivfflat 精度在大规模下降，需 HNSW 或专用向量库 |
| 健康探针 | ✅ 支持 | Ping 检查 |
| 链路追踪 | ✅ 支持 | `db.vector.*` span |

## 6. 已知限制与风险

1. **无连接池**：使用单 `pgx.Conn`，高并发场景下连接会成为瓶颈，建议业务侧改用 `pgxpool` 封装。
2. **DELETE 逐条执行**：批量删除效率低，百万级删除耗时显著，建议改为子查询批量删除。
3. **ivfflat lists 固定 100**：未暴露 IVFFlat 的 lists 参数配置，大规模数据下召回率和查询速度受限于 lists 设置。
4. **距离度量固定 Cosine**：`vector_cosine_ops` 硬编码，切换 L2/IP 需修改驱动源码。
5. **Upsert 缺失**：重复 ID 插入会报 `duplicate key` 错误，业务侧需先检查再插入或捕获错误重试。
6. **扩展安装依赖**：首次使用前需业务侧在数据库执行 `CREATE EXTENSION IF NOT EXISTS vector`（驱动已自动执行）。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/vectorx"
    _ "github.com/gospacex/vectorx/pgvector" // 注册驱动
)

client, _ := vectorx.New(&vectorx.Config{
    Name: "default", Driver: "pgvector",
    Address: "postgres://postgres:postgres@localhost:5432/vectorx?sslmode=disable",
    Namespace: "vector_db",  // 对应 schema
    Dim: 768,
})
client.CreateCollection(ctx, "articles", 768)
client.Insert(ctx, "articles", []vectorx.Embedding{
    {ID: "article-001", Vector: vec, Fields: map[string]interface{}{"author": "张三"}},
})
result, _ := client.Search(ctx, "articles", query, 10)
```
