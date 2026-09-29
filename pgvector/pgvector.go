// vectorx/pgvector: PostgreSQL + pgvector 向量数据库驱动
package pgvector

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
	"github.com/gospacex/vectorx"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
	"go.opentelemetry.io/otel/attribute"
)

type clientImpl struct {
	conn   *pgx.Conn
	cfg    *vectorx.Config
	tracer *traceflag.Tracer
	logger logx.Logger
}

func ctx() context.Context { return context.Background() }

// New 创建 pgvector 客户端
func New(cfg *vectorx.Config) (*clientImpl, error) {
	if cfg.Driver != "pgvector" {
		return nil, fmt.Errorf("pgvector: expected driver 'pgvector', got %q", cfg.Driver)
	}
	addr := cfg.Address
	if addr == "" {
		addr = "postgres://postgres:postgres@localhost:5432/vectorx?sslmode=disable"
	}
	conn, err := pgx.Connect(ctx(), addr)
	if err != nil {
		return nil, fmt.Errorf("vectorx/pgvector: connect: %w", err)
	}
	_, err = conn.Exec(ctx(), "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		conn.Close(ctx())
		return nil, fmt.Errorf("vectorx/pgvector: create extension: %w", err)
	}
	logger := logx.GetGlobal()
	return &clientImpl{
		conn:   conn,
		cfg:    cfg,
		tracer: traceflag.NewTracer(cfg.EnableTrace, "github.com/gospacex/vectorx/pgvector", "vectorx"),
		logger: logger,
	}, nil
}

func (c *clientImpl) CreateCollection(ctx context.Context, name string, dim int) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.create_collection")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.vector.collection", name),
		attribute.Int("db.rows_affected", dim),
	)
	schema := c.schema()
	table := c.table(name)
	_, err := c.conn.Exec(ctx, fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			embedding vector(%d),
			fields jsonb
		)`, quotedTable(schema, table), dim))
	if err != nil {
		return fmt.Errorf("vectorx/pgvector: create table: %w", err)
	}
	_, err = c.conn.Exec(ctx, fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS idx_%s_embedding ON %s USING ivfflat (embedding vector_cosine_ops)
		WITH (lists = 100)`, quoteIdentifier(table), quotedTable(schema, table)))
	return err
}

func (c *clientImpl) DropCollection(ctx context.Context, name string) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.drop_collection")
	defer span.End()
	span.SetAttributes(attribute.String("db.vector.collection", name))
	_, err := c.conn.Exec(ctx, fmt.Sprintf(
		`DROP TABLE IF EXISTS %s`, quotedTable(c.schema(), c.table(name))))
	return err
}

func (c *clientImpl) Insert(ctx context.Context, name string, embeddings []vectorx.Embedding) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.insert")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.vector.collection", name),
		attribute.Int("db.vector.insert_count", len(embeddings)),
	)
	if len(embeddings) == 0 {
		return nil
	}
	schema := c.schema()
	table := c.table(name)
	batch := &pgx.Batch{}
	for _, e := range embeddings {
		emb := pgvector.NewVector([]float32(e.Vector))
		fieldsJSON := []byte("[]")
		if len(e.Fields) > 0 {
			b, _ := json.Marshal(e.Fields)
			fieldsJSON = b
		}
		batch.Queue(fmt.Sprintf("INSERT INTO %s (id, embedding, fields) VALUES ($1, $2, $3)",
			quotedTable(schema, table)), e.ID, emb, fieldsJSON)
	}
	br := c.conn.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("vectorx/pgvector: insert row %d: %w", i, err)
		}
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "pgvector insert done",
			logx.String("collection", name), logx.Int("count", len(embeddings)))
	}
	return nil
}

func (c *clientImpl) Search(ctx context.Context, name string, query vectorx.Vector, limit int) (*vectorx.SearchResult, error) {
	ctx, span := c.tracer.Start(ctx, "db.vector.search")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.vector.collection", name),
		attribute.Int("db.vector.limit", limit),
	)
	if limit <= 0 {
		limit = 10
	}
	schema := c.schema()
	table := c.table(name)
	emb := pgvector.NewVector([]float32(query))
	rows, err := c.conn.Query(ctx, fmt.Sprintf(`
		SELECT id, embedding, fields, embedding <-> $1 AS distance FROM %s
		ORDER BY embedding <-> $1 LIMIT $2`, quotedTable(schema, table)), emb, limit)
	if err != nil {
		return nil, fmt.Errorf("vectorx/pgvector: search: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0)
	distances := make([]float32, 0)
	fields := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id string
		var embedding pgvector.Vector
		var raw []byte
		var distance float64
		if err := rows.Scan(&id, &embedding, &raw, &distance); err != nil {
			return nil, fmt.Errorf("vectorx/pgvector: scan row: %w", err)
		}
		ids = append(ids, id)
		distances = append(distances, float32(distance))
		var m map[string]interface{}
		if len(raw) > 0 {
			if uerr := json.Unmarshal(raw, &m); uerr != nil {
				m = map[string]interface{}{}
			}
		} else {
			m = map[string]interface{}{}
		}
		fields = append(fields, m)
	}
	return &vectorx.SearchResult{IDs: ids, Distances: distances, Fields: fields}, nil
}

func (c *clientImpl) Delete(ctx context.Context, name string, ids []string) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.vector.collection", name),
		attribute.Int("db.vector.delete_count", len(ids)),
	)
	if len(ids) == 0 {
		return nil
	}
	schema := c.schema()
	table := c.table(name)
	for _, id := range ids {
		if _, err := c.conn.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = $1", quotedTable(schema, table)), id); err != nil {
			return err
		}
	}
	return nil
}

func (c *clientImpl) HealthCheck() error { return c.conn.Ping(ctx()) }
func (c *clientImpl) Close() error       { return c.conn.Close(ctx()) }
func (c *clientImpl) Driver() string     { return "pgvector" }

func (c *clientImpl) schema() string {
	if c.cfg.Namespace != "" {
		return c.cfg.Namespace
	}
	return "public"
}

func (c *clientImpl) table(name string) string { return name }

// quoteIdentifier quotes a PostgreSQL identifier to prevent SQL injection.
func quoteIdentifier(s string) string {
	return `"` + s + `"`
}

// quotedTable returns "schema"."table" for use in SQL.
func quotedTable(schema, table string) string {
	return quoteIdentifier(schema) + "." + quoteIdentifier(table)
}

func init() {
	// Ensure pgvector's driver types are registered
	_ = driver.Value(nil)
	vectorx.RegisterCreator("pgvector", func(cfg *vectorx.Config) (vectorx.CollectionClient, error) {
		return New(cfg)
	})
}
