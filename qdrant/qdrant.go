// vectorx/qdrant: Qdrant 向量数据库驱动
package qdrant

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
	"github.com/gospacex/vectorx"
	"github.com/qdrant/go-client/qdrant"
	"go.opentelemetry.io/otel/attribute"
)

type clientImpl struct {
	client *qdrant.Client
	cfg    *vectorx.Config
	tracer *traceflag.Tracer
	logger logx.Logger
}

func ctx() context.Context { return context.Background() }

// New 创建 Qdrant 客户端
func New(cfg *vectorx.Config) (*clientImpl, error) {
	if cfg.Driver != "qdrant" {
		return nil, fmt.Errorf("qdrant: expected driver 'qdrant', got %q", cfg.Driver)
	}
	addr := cfg.Address
	if addr == "" {
		addr = "localhost:6334"
	}
	host, port := parseAddress(addr)
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("vectorx/qdrant: connect: %w", err)
	}
	logger := logx.GetGlobal()
	return &clientImpl{
		client: client,
		cfg:    cfg,
		tracer: traceflag.NewTracer(cfg.EnableTrace, "github.com/gospacex/vectorx/qdrant", "vectorx"),
		logger: logger,
	}, nil
}

func (c *clientImpl) CreateCollection(ctx context.Context, name string, dim int) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.create_collection")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "qdrant"),
		attribute.String("db.vector.collection", name),
		attribute.Int("db.vector.dim", dim),
	)
	err := c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(dim),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("vectorx/qdrant: create collection %q: %w", name, err)
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "qdrant collection created",
			logx.String("collection", name), logx.Int("dim", dim))
	}
	return nil
}

func (c *clientImpl) DropCollection(ctx context.Context, name string) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.drop_collection")
	defer span.End()
	span.SetAttributes(attribute.String("db.vector.collection", name))
	err := c.client.DeleteCollection(ctx, name)
	if err != nil {
		return fmt.Errorf("vectorx/qdrant: drop collection %q: %w", name, err)
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "qdrant collection dropped",
			logx.String("collection", name))
	}
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
	points := make([]*qdrant.PointStruct, 0, len(embeddings))
	for _, e := range embeddings {
		vectors := qdrant.NewVectorsDense(e.Vector)
		pointID := qdrant.NewIDUUID(e.ID)
		pt := &qdrant.PointStruct{
			Id:      pointID,
			Vectors: vectors,
			Payload: toPayload(e.Fields),
		}
		points = append(points, pt)
	}
	_, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: name,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("vectorx/qdrant: upsert: %w", err)
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "qdrant insert done",
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
	resp, err := c.client.GetPointsClient().Search(ctx, &qdrant.SearchPoints{
		CollectionName: name,
		Vector:         query,
		Limit:          uint64(limit),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("vectorx/qdrant: search: %w", err)
	}
	result := resp.GetResult()
	ids := make([]string, 0, len(result))
	scores := make([]float32, 0, len(result))
	fieldsList := make([]map[string]interface{}, 0, len(result))
	for _, p := range result {
		id := p.GetId()
		if id != nil {
			ids = append(ids, id.GetUuid())
		} else {
			ids = append(ids, "")
		}
		scores = append(scores, p.GetScore())
		fieldsList = append(fieldsList, fromPayload(p.GetPayload()))
	}
	return &vectorx.SearchResult{IDs: ids, Distances: scores, Fields: fieldsList}, nil
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
	pointIDs := make([]*qdrant.PointId, 0, len(ids))
	for _, id := range ids {
		pointIDs = append(pointIDs, qdrant.NewIDUUID(id))
	}
	_, err := c.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: name,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{Ids: pointIDs},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("vectorx/qdrant: delete: %w", err)
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "qdrant delete done",
			logx.String("collection", name), logx.Int("count", len(ids)))
	}
	return nil
}

func (c *clientImpl) HealthCheck() error {
	_, err := c.client.HealthCheck(ctx())
	return err
}

func (c *clientImpl) Close() error   { return c.client.Close() }
func (c *clientImpl) Driver() string { return "qdrant" }

// parseAddress splits "host:port" into (host, port), defaulting to localhost:6334
func parseAddress(addr string) (string, int) {
	host := "localhost"
	port := 6334
	parts := strings.SplitN(addr, ":", 2)
	if len(parts) >= 1 && parts[0] != "" {
		host = parts[0]
	}
	if len(parts) >= 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil && p > 0 {
			port = p
		}
	}
	return host, port
}

// toPayload converts map[string]interface{} to Qdrant payload format
func toPayload(m map[string]interface{}) map[string]*qdrant.Value {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]*qdrant.Value, len(m))
	for k, v := range m {
		val, err := qdrant.NewValue(v)
		if err != nil {
			// fallback to string representation
			val = qdrant.NewValueString(fmt.Sprintf("%v", v))
		}
		out[k] = val
	}
	return out
}

// fromPayload converts Qdrant payload to map[string]interface{}
func fromPayload(p map[string]*qdrant.Value) map[string]interface{} {
	if len(p) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(p))
	for k, v := range p {
		out[k] = valueToInterface(v)
	}
	return out
}

// valueToInterface converts a Qdrant Value to a Go interface{}
func valueToInterface(v *qdrant.Value) interface{} {
	if v == nil {
		return nil
	}
	kind := v.GetKind()
	if kind == nil {
		return nil
	}
	switch val := kind.(type) {
	case *qdrant.Value_NullValue:
		return nil
	case *qdrant.Value_DoubleValue:
		return val.DoubleValue
	case *qdrant.Value_IntegerValue:
		return val.IntegerValue
	case *qdrant.Value_StringValue:
		return val.StringValue
	case *qdrant.Value_BoolValue:
		return val.BoolValue
	case *qdrant.Value_StructValue:
		structVal := val.StructValue
		if structVal == nil || structVal.Fields == nil {
			return nil
		}
		result := make(map[string]interface{}, len(structVal.Fields))
		for k, elem := range structVal.Fields {
			result[k] = valueToInterface(elem)
		}
		return result
	case *qdrant.Value_ListValue:
		listVal := val.ListValue
		if listVal == nil || listVal.Values == nil {
			return nil
		}
		result := make([]interface{}, 0, len(listVal.Values))
		for _, elem := range listVal.Values {
			result = append(result, valueToInterface(elem))
		}
		return result
	default:
		return nil
	}
}

func init() {
	vectorx.RegisterCreator("qdrant", func(cfg *vectorx.Config) (vectorx.CollectionClient, error) {
		return New(cfg)
	})
}
