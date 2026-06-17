package milvusx

import (
	"context"
	"fmt"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/observability"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"go.opentelemetry.io/otel/attribute"
)

type Milvusx struct {
	client.Client
	cfg *config.MilvusConfig
}

func newClient(cfg *config.MilvusConfig) (*Milvusx, error) {
	c, err := client.NewClient(context.Background(), client.Config{
		Address:  cfg.Address,
		Username: cfg.Username,
		Password: cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return &Milvusx{Client: c, cfg: cfg}, nil
}

func (m *Milvusx) Search(ctx context.Context, collName string, partitions []string, expr string, outputFields []string, vectors []entity.Vector, vectorField string, metricType entity.MetricType, topK int, sp entity.SearchParam, opts ...client.SearchQueryOptionFunc) ([]client.SearchResult, error) {
	ctx, span := observability.StartSpan(ctx, "milvusx.Search",
		attribute.String("collection", collName),
		attribute.Int("topK", topK),
	)
	defer span.End()

	res, err := m.Client.Search(ctx, collName, partitions, expr, outputFields, vectors, vectorField, metricType, topK, sp, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (m *Milvusx) Insert(ctx context.Context, collName string, partitionName string, columns ...entity.Column) (entity.Column, error) {
	ctx, span := observability.StartSpan(ctx, "milvusx.Insert",
		attribute.String("collection", collName),
	)
	defer span.End()

	res, err := m.Client.Insert(ctx, collName, partitionName, columns...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (m *Milvusx) Flush(ctx context.Context, collName string, async bool, opts ...client.FlushOption) error {
	ctx, span := observability.StartSpan(ctx, "milvusx.Flush",
		attribute.String("collection", collName),
	)
	defer span.End()

	err := m.Client.Flush(ctx, collName, async, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (m *Milvusx) CreateCollection(ctx context.Context, schema *entity.Schema, shardsNum int32, opts ...client.CreateCollectionOption) error {
	ctx, span := observability.StartSpan(ctx, "milvusx.CreateCollection",
		attribute.String("collection", schema.CollectionName),
	)
	defer span.End()

	err := m.Client.CreateCollection(ctx, schema, shardsNum, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (m *Milvusx) DropCollection(ctx context.Context, collName string, opts ...client.DropCollectionOption) error {
	ctx, span := observability.StartSpan(ctx, "milvusx.DropCollection",
		attribute.String("collection", collName),
	)
	defer span.End()

	err := m.Client.DropCollection(ctx, collName, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (m *Milvusx) HasCollection(ctx context.Context, collName string) (bool, error) {
	ctx, span := observability.StartSpan(ctx, "milvusx.HasCollection",
		attribute.String("collection", collName),
	)
	defer span.End()

	ok, err := m.Client.HasCollection(ctx, collName)
	if err != nil {
		span.RecordError(err)
	}
	return ok, err
}

func (m *Milvusx) DescribeCollection(ctx context.Context, collName string) (*entity.Collection, error) {
	ctx, span := observability.StartSpan(ctx, "milvusx.DescribeCollection",
		attribute.String("collection", collName),
	)
	defer span.End()

	desc, err := m.Client.DescribeCollection(ctx, collName)
	if err != nil {
		span.RecordError(err)
	}
	return desc, err
}

// Close releases the underlying gRPC connection and evicts this instance
// from the package-level cache. The eviction is what makes Close safe
// across multiple test cases in the same binary (and across reloads in
// long-running services that switch named instances): the next
// GetMilvus(name) call will construct a fresh client with a live
// connection, instead of returning the cached closed one — which would
// surface as a transient `grpc: the client connection is closing`
// error on the next RPC.
//
// The cache eviction runs before the conn-close attempt so that
// callers who use a zero-value *Milvusx (e.g. unit tests that drive
// the cache directly) still get the eviction. The conn-close
// short-circuits when the embedded client is nil.
//
// Safe to call multiple times; the second call sees m.Client == nil
// and a cache miss, both of which are no-ops.
func (m *Milvusx) Close() error {
	if m.cfg != nil {
		clientCache.Delete(m.cfg.Name)
		if v, ok := forcedCloseErr.LoadAndDelete(m.cfg.Name); ok {
			// Comma-ok on purpose: a non-error value would panic the
			// process. The sync.Map is test-only (production never
			// stores anything), so this should always be an error,
			// but defending against bad test fixtures is cheap.
			if e, ok := v.(error); ok {
				return e
			}
		}
	}
	if m.Client == nil {
		return nil
	}
	err := m.Client.Close()
	m.Client = nil
	return err
}
