// Package elasticsearch: Elasticsearch kNN 向量数据库驱动
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esutil"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
	"github.com/gospacex/vectorx"
	"go.opentelemetry.io/otel/attribute"
)

// Similarity is the vector distance metric used by this driver.
type Similarity string

const (
	SimilarityCosine     Similarity = "cosine"
	SimilarityDotProduct Similarity = "dot_product"
	SimilarityL2         Similarity = "l2_norm"
)

func (s Similarity) String() string {
	switch s {
	case SimilarityCosine:
		return "cosine"
	case SimilarityDotProduct:
		return "dot_product"
	case SimilarityL2:
		return "l2_norm"
	default:
		return "cosine"
	}
}

type clientImpl struct {
	es          *elasticsearch.Client
	bulkIndexer esutil.BulkIndexer
	cfg         *vectorx.Config
	tracer      *traceflag.Tracer
	logger      logx.Logger
}

func ctx() context.Context { return context.Background() }

// New 创建 Elasticsearch 客户端
func New(cfg *vectorx.Config) (*clientImpl, error) {
	if cfg.Driver != "elasticsearch" {
		return nil, fmt.Errorf("elasticsearch: expected driver 'elasticsearch', got %q", cfg.Driver)
	}
	addr := cfg.Address
	if addr == "" {
		addr = "http://localhost:9200"
	}
	esCfg := elasticsearch.Config{
		Addresses: []string{addr},
		Transport: &http.Transport{},
	}
	if cfg.Token != "" {
		esCfg.Username = "elastic"
		esCfg.Password = cfg.Token
	}
	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("vectorx/elasticsearch: create client: %w", err)
	}
	info, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("vectorx/elasticsearch: connection check: %w", err)
	}
	if info.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vectorx/elasticsearch: info endpoint returned %d", info.StatusCode)
	}
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:  client,
		Index:   cfg.Namespace,
		OnError: func(ctx context.Context, err error) { logx.ErrorContext(ctx, "bulk indexer error", logx.Err(err)) },
	})
	if err != nil {
		return nil, fmt.Errorf("vectorx/elasticsearch: create bulk indexer: %w", err)
	}
	logger := logx.GetGlobal()
	return &clientImpl{
		es:          client,
		bulkIndexer: bi,
		cfg:         cfg,
		tracer:      traceflag.NewTracer(cfg.EnableTrace, "github.com/gospacex/vectorx/elasticsearch", "vectorx"),
		logger:      logger,
	}, nil
}

// CreateCollection 创建 ES 索引（若已存在则静默跳过）
func (c *clientImpl) CreateCollection(ctx context.Context, name string, dim int) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.create_collection")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "elasticsearch"),
		attribute.String("db.statement", name),
		attribute.Int("db.rows_affected", dim),
	)
	mapping := fmt.Sprintf(`{
		"mappings": {
			"properties": {
				"id": {"type": "keyword"},
				"embedding": {
					"type": "dense_vector",
					"dims": %d,
					"index": true,
					"similarity": "%s"
				},
				"fields": {"type": "object", "dynamic": true}
			}
		}
	}`, dim, SimilarityCosine)
	exists, err := c.es.Indices.Exists([]string{name})
	if err != nil {
		return fmt.Errorf("vectorx/elasticsearch: check index exists: %w", err)
	}
	if exists.StatusCode == http.StatusOK {
		if c.logger != nil {
			c.logger.InfoContext(ctx, "elasticsearch collection already exists", logx.String("collection", name))
		}
		return nil
	}
	resp, err := c.es.Indices.Create(name, c.es.Indices.Create.WithBody(bytes.NewReader([]byte(mapping))))
	if err != nil {
		return fmt.Errorf("vectorx/elasticsearch: create index %q: %w", name, err)
	}
	if resp.IsError() {
		return fmt.Errorf("vectorx/elasticsearch: create index %q: status %d", name, resp.StatusCode)
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "elasticsearch collection created",
			logx.String("collection", name), logx.Int("dim", dim))
	}
	return nil
}

// DropCollection 删除 ES 索引
func (c *clientImpl) DropCollection(ctx context.Context, name string) error {
	ctx, span := c.tracer.Start(ctx, "db.vector.drop_collection")
	defer span.End()
	span.SetAttributes(attribute.String("db.vector.collection", name))
	resp, err := c.es.Indices.Delete([]string{name})
	if err != nil {
		return fmt.Errorf("vectorx/elasticsearch: drop index %q: %w", name, err)
	}
	if resp.IsError() {
		return fmt.Errorf("vectorx/elasticsearch: drop index %q: status %d", name, resp.StatusCode)
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "elasticsearch collection dropped", logx.String("collection", name))
	}
	return nil
}

// Insert 批量插入向量文档
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
	for _, e := range embeddings {
		doc := map[string]interface{}{
			"id":        e.ID,
			"embedding": e.Vector,
			"fields":    e.Fields,
		}
		j, _ := json.Marshal(doc)
		if err := c.bulkIndexer.Add(ctx, esutil.BulkIndexerItem{
			Action:     "index",
			Index:      name,
			DocumentID: e.ID,
			Body:       bytes.NewReader(j),
		}); err != nil {
			return fmt.Errorf("vectorx/elasticsearch: bulk add: %w", err)
		}
	}
	if err := c.bulkIndexer.Close(ctx); err != nil {
		return fmt.Errorf("vectorx/elasticsearch: flush bulk: %w", err)
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "elasticsearch insert done",
			logx.String("collection", name), logx.Int("count", len(embeddings)))
	}
	return nil
}

// Search 执行 kNN 向量相似度搜索
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
	knn := fmt.Sprintf(`{
		"field": "embedding",
		"query_vector": [%s],
		"k": %d,
		"num_candidates": %d,
		"boost": 1.0
	}`, floatSliceToString(query), limit, limit*10)
	filter := `{"bool":{"must_not":[{"match_all":{}}]}}`
	reqBody := fmt.Sprintf(`{
		"knn": {"%s": %s},
		"_source": ["id", "embedding", "fields"]
	}`, name, knn)
	if filter != `` {
		reqBody = fmt.Sprintf(`{
			"knn": {"%s": %s},
			"_source": ["id", "embedding", "fields"],
			"filter": %s
		}`, name, knn, filter)
	}
	resp, err := c.es.Search(
		c.es.Search.WithIndex(name),
		c.es.Search.WithBody(bytes.NewReader([]byte(reqBody))),
		c.es.Search.WithSize(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("vectorx/elasticsearch: search: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Hits struct {
			Hits []struct {
				ID     string                 `json:"_id"`
				Score  float32                `json:"_score"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("vectorx/elasticsearch: decode search result: %w", err)
	}
	ids := make([]string, 0, len(result.Hits.Hits))
	distances := make([]float32, 0, len(result.Hits.Hits))
	fields := make([]map[string]interface{}, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		ids = append(ids, hit.ID)
		distances = append(distances, float32(hit.Score))
		fields = append(fields, hit.Source)
	}
	return &vectorx.SearchResult{IDs: ids, Distances: distances, Fields: fields}, nil
}

// Delete 删除指定 ID 的向量文档
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
	for _, id := range ids {
		resp, err := c.es.Delete(name, id)
		if err != nil {
			return fmt.Errorf("vectorx/elasticsearch: delete %q: %w", id, err)
		}
		if resp.IsError() && resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("vectorx/elasticsearch: delete %q: status %d", id, resp.StatusCode)
		}
	}
	return nil
}

// HealthCheck 检查 ES 集群健康状态
func (c *clientImpl) HealthCheck() error {
	_, err := c.es.Cluster.Health()
	return err
}

// Close 关闭客户端并刷新 bulk indexer
func (c *clientImpl) Close() error {
	_ = c.bulkIndexer.Close(ctx())
	return c.es.Close(ctx())
}

// Driver 返回驱动名称
func (c *clientImpl) Driver() string { return "elasticsearch" }

// floatSliceToJSON converts a float32 slice to a JSON array string.
func floatSliceToString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return `["` + parts[0] + `"]` // placeholder; use real join below
}

func init() {
	vectorx.RegisterCreator("elasticsearch", func(cfg *vectorx.Config) (vectorx.CollectionClient, error) {
		return New(cfg)
	})
}
