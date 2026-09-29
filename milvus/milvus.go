// vectorx/milvus: Milvus 向量数据库驱动
package milvus

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
	"github.com/gospacex/vectorx"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"go.opentelemetry.io/otel/attribute"
)

type clientImpl struct {
	mc     client.Client
	cfg    *vectorx.Config
	tracer *traceflag.Tracer
	logger logx.Logger
}

func ctx() context.Context { return context.Background() }

// New 创建 Milvus 客户端
func New(cfg *vectorx.Config) (*clientImpl, error) {
	if cfg.Driver != "milvus" {
		return nil, fmt.Errorf("milvus: expected driver 'milvus', got %q", cfg.Driver)
	}
	addr := cfg.Address
	if addr == "" {
		addr = "localhost:19530"
	}
	c, err := client.NewClient(ctx(), client.Config{Address: addr})
	if err != nil {
		return nil, fmt.Errorf("vectorx/milvus: connect: %w", err)
	}
	logger := logx.GetGlobal()
	return &clientImpl{
		mc:     c,
		cfg:    cfg,
		tracer: traceflag.NewTracer(cfg.EnableTrace, "github.com/gospacex/vectorx/milvus", "vectorx"),
		logger: logger,
	}, nil
}

func (c *clientImpl) CreateCollection(ctx context.Context, name string, dim int) error {
	c.logger.InfoContext(ctx, "vectorx/milvus create collection started", logx.String("collection", name), logx.Int("dim", dim))
	ctx, span := c.tracer.Start(ctx, "db.vector.create_collection")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "milvus"),
		attribute.String("db.statement", name),
		attribute.Int("db.rows_affected", dim),
	)
	schema := entity.NewSchema().
		WithName(name).
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("embedding").WithDataType(entity.FieldTypeFloatVector).WithTypeParams(entity.TypeParamDim, strconv.Itoa(dim))).
		WithDynamicFieldEnabled(true)
	if err := c.mc.CreateCollection(ctx, schema, 1); err != nil {
		c.logger.ErrorContext(ctx, "vectorx/milvus create collection failed", logx.Err(err), logx.String("collection", name))
		return fmt.Errorf("vectorx/milvus: create collection %q: %w", name, err)
	}
	c.logger.InfoContext(ctx, "vectorx/milvus create collection done", logx.String("collection", name))
	idx, err := entity.NewIndexFlat(entity.L2)
	if err != nil {
		return err
	}
	return c.mc.CreateIndex(ctx, name, "embedding", idx, false)
}

func (c *clientImpl) DropCollection(ctx context.Context, name string) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.drop_collection")
	defer span.End()
	span.SetAttributes(attribute.String("db.vector.collection", name))
	if err := c.mc.DropCollection(ctx, name); err != nil {
		c.logger.ErrorContext(ctx, "vectorx/milvus drop collection failed", logx.Err(err), logx.String("collection", name))
		return err
	}
	c.logger.InfoContext(ctx, "vectorx/milvus drop collection done", logx.String("collection", name))
	return nil
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
	vecs := make([][]float32, 0, len(embeddings))
	jsonBytes := make([][]byte, 0, len(embeddings))
	for _, e := range embeddings {
		vecs = append(vecs, []float32(e.Vector))
		if len(e.Fields) > 0 {
			b, err := json.Marshal(e.Fields)
			if err != nil {
				return fmt.Errorf("vectorx/milvus: marshal fields: %w", err)
			}
			jsonBytes = append(jsonBytes, b)
		} else {
			jsonBytes = append(jsonBytes, []byte("{}"))
		}
	}
	columns := []entity.Column{
		entity.NewColumnFloatVector("embedding", len(vecs[0]), vecs),
		entity.NewColumnDynamic(entity.NewColumnJSONBytes("fields", jsonBytes), ""),
	}
	if _, err := c.mc.Insert(ctx, name, "", columns...); err != nil {
		return fmt.Errorf("vectorx/milvus: insert: %w", err)
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "milvus insert done", logx.String("collection", name), logx.Int("count", len(embeddings)))
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
	searchParam, err := entity.NewIndexFlatSearchParam()
	if err != nil {
		return nil, err
	}
	resp, err := c.mc.Search(ctx, name, nil, "", []string{"*"},
		[]entity.Vector{entity.FloatVector(query)},
		"embedding", entity.L2, limit, searchParam)
	if err != nil {
		return nil, fmt.Errorf("vectorx/milvus: search: %w", err)
	}
	if len(resp) == 0 {
		return &vectorx.SearchResult{}, nil
	}
	sr := resp[0]
	ids, err := extractIDs(sr.IDs)
	if err != nil {
		return nil, fmt.Errorf("vectorx/milvus: extract ids: %w", err)
	}
	distances := sr.Scores
	fields := make([]map[string]interface{}, 0, len(ids))
	for i := range ids {
		row := map[string]interface{}{}
		if sr.Fields != nil {
			for _, col := range sr.Fields {
				data, err := col.Get(i)
				if err == nil {
					row[col.Name()] = data
				}
			}
		}
		fields = append(fields, row)
	}
	c.logger.InfoContext(ctx, "vectorx/milvus search done", logx.String("collection", name), logx.Int("topk", limit), logx.Int("hits", len(ids)))
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
	intIDs, err := parseIDs(ids)
	if err != nil {
		return err
	}
	if err := c.mc.DeleteByPks(ctx, name, "", entity.NewColumnInt64("id", intIDs)); err != nil {
		c.logger.ErrorContext(ctx, "vectorx/milvus delete failed", logx.Err(err), logx.String("collection", name))
		return err
	}
	c.logger.InfoContext(ctx, "vectorx/milvus delete done", logx.String("collection", name), logx.Int("count", len(ids)))
	return nil
}

func (c *clientImpl) HealthCheck() error {
	_, err := c.mc.GetVersion(ctx())
	return err
}

func (c *clientImpl) Close() error   { return c.mc.Close() }
func (c *clientImpl) Driver() string { return "milvus" }

func parseIDs(ids []string) ([]int64, error) {
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		v, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("vectorx/milvus: parse id %q as int64: %w", id, err)
		}
		result = append(result, v)
	}
	return result, nil
}

func extractIDs(col entity.Column) ([]string, error) {
	if col == nil || col.Len() == 0 {
		return nil, nil
	}
	strs := make([]string, col.Len())
	for i := 0; i < col.Len(); i++ {
		v, err := col.Get(i)
		if err != nil {
			return nil, fmt.Errorf("milvus: extract id at %d: %w", i, err)
		}
		strs[i] = fmt.Sprintf("%v", v)
	}
	return strs, nil
}

func init() {
	vectorx.RegisterCreator("milvus", func(cfg *vectorx.Config) (vectorx.CollectionClient, error) {
		return New(cfg)
	})
}
