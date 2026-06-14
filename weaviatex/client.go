package weaviatex

import (
	"context"
	"fmt"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/observability"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/auth"
	"github.com/weaviate/weaviate/entities/models"
	"go.opentelemetry.io/otel/attribute"
)

type Weaviatex struct {
	client *weaviate.Client
	cfg    *config.WeaviateConfig
}

func newClient(cfg *config.WeaviateConfig) (*Weaviatex, error) {
	wcfg := weaviate.Config{
		Host:   cfg.Host,
		Scheme: cfg.Scheme,
	}
	if cfg.APIKey != "" {
		wcfg.AuthConfig = auth.ApiKey{Value: cfg.APIKey}
	}
	c, err := weaviate.NewClient(wcfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return &Weaviatex{client: c, cfg: cfg}, nil
}

func (w *Weaviatex) GraphQLRaw(ctx context.Context, query string) (any, error) {
	ctx, span := observability.StartSpan(ctx, "weaviatex.GraphQLRaw")
	defer span.End()

	resp, err := w.client.GraphQL().Raw().WithQuery(query).Do(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if resp.Errors != nil {
		err = fmt.Errorf("graphql errors: %v", resp.Errors)
		span.RecordError(err)
		return nil, err
	}
	return resp.Data, nil
}

func (w *Weaviatex) CreateObject(ctx context.Context, className string, properties map[string]any, vector []float32) (any, error) {
	ctx, span := observability.StartSpan(ctx, "weaviatex.CreateObject",
		attribute.String("class", className),
	)
	defer span.End()

	obj, err := w.client.Data().Creator().
		WithClassName(className).
		WithProperties(properties).
		WithVector(vector).
		Do(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return obj, err
}

func (w *Weaviatex) DeleteObject(ctx context.Context, className string, id string) error {
	ctx, span := observability.StartSpan(ctx, "weaviatex.DeleteObject",
		attribute.String("class", className),
	)
	defer span.End()

	err := w.client.Data().Deleter().
		WithClassName(className).
		WithID(id).
		Do(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (w *Weaviatex) CreateClass(ctx context.Context, class *models.Class) error {
	ctx, span := observability.StartSpan(ctx, "weaviatex.CreateClass",
		attribute.String("class", class.Class),
	)
	defer span.End()

	err := w.client.Schema().ClassCreator().
		WithClass(class).
		Do(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (w *Weaviatex) IsLive(ctx context.Context) (bool, error) {
	ctx, span := observability.StartSpan(ctx, "weaviatex.IsLive")
	defer span.End()

	live, err := w.client.Misc().LiveChecker().Do(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return live, err
}
