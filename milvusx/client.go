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

func (m *Milvusx) Close() error {
	return m.Client.Close()
}
