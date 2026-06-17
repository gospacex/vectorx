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

// weaviateOps is the seam the adapter uses to talk to the underlying
// Weaviate SDK. Weaviate's *weaviate.Client is concrete, not
// interface-typed, so without this seam every public method would
// need an integration test to exercise — and unit tests would not
// be able to assert span-name / error-recording contracts at all.
//
// Production wires *liveWeaviateOps; tests can inject their own
// implementation by assigning the field after construction. The
// field is unexported so the seam is internal — the public surface
// of *Weaviatex stays the same.
//
// CRITICAL: this is the *only* path the public methods use. The
// previous revision kept a separate `client *weaviate.Client` field
// on *Weaviatex alongside `ops`, which let future maintainers
// bypass the seam by calling w.client.X directly. That dual-field
// shape is what the code review flagged as "half-applied". The
// Weaviatex struct now holds only the seam, not the underlying
// SDK client — the SDK handle lives inside *liveWeaviateOps, where
// tests cannot reach it.
//
// The method names mirror *Weaviatex's public methods one-to-one
// (lowercase would collide; we use the unexported "ops" prefix on
// each method to keep the intent clear at the call site).
type weaviateOps interface {
	opsGraphQLRaw(ctx context.Context, query string) (any, error)
	opsCreateObject(ctx context.Context, className string, properties map[string]any, vector []float32) (any, error)
	opsDeleteObject(ctx context.Context, className string, id string) error
	opsCreateClass(ctx context.Context, class *models.Class) error
	opsIsLive(ctx context.Context) (bool, error)
}

// liveWeaviateOps is the production implementation of weaviateOps.
// It is a thin shim over *weaviate.Client: each method maps the
// adapter's plain-args signature onto the SDK's fluent builder API.
type liveWeaviateOps struct {
	client *weaviate.Client
}

func (l *liveWeaviateOps) opsGraphQLRaw(ctx context.Context, query string) (any, error) {
	resp, err := l.client.GraphQL().Raw().WithQuery(query).Do(ctx)
	if err != nil {
		return nil, err
	}
	if resp.Errors != nil {
		return nil, fmt.Errorf("graphql errors: %v", resp.Errors)
	}
	return resp.Data, nil
}

func (l *liveWeaviateOps) opsCreateObject(ctx context.Context, className string, properties map[string]any, vector []float32) (any, error) {
	return l.client.Data().Creator().
		WithClassName(className).
		WithProperties(properties).
		WithVector(vector).
		Do(ctx)
}

func (l *liveWeaviateOps) opsDeleteObject(ctx context.Context, className string, id string) error {
	return l.client.Data().Deleter().
		WithClassName(className).
		WithID(id).
		Do(ctx)
}

func (l *liveWeaviateOps) opsCreateClass(ctx context.Context, class *models.Class) error {
	return l.client.Schema().ClassCreator().
		WithClass(class).
		Do(ctx)
}

func (l *liveWeaviateOps) opsIsLive(ctx context.Context) (bool, error) {
	return l.client.Misc().LiveChecker().Do(ctx)
}

type Weaviatex struct {
	ops weaviateOps
	cfg *config.WeaviateConfig
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
	// The SDK handle lives inside *liveWeaviateOps — there is no
	// `client` field on *Weaviatex. Tests inject their own weaviateOps
	// implementation; production keeps the weaviate.Client handle here
	// where tests cannot reach it.
	return &Weaviatex{ops: &liveWeaviateOps{client: c}, cfg: cfg}, nil
}

func (w *Weaviatex) GraphQLRaw(ctx context.Context, query string) (any, error) {
	ctx, span := observability.StartSpan(ctx, "weaviatex.GraphQLRaw")
	defer span.End()

	resp, err := w.ops.opsGraphQLRaw(ctx, query)
	if err != nil {
		span.RecordError(err)
	}
	return resp, err
}

func (w *Weaviatex) CreateObject(ctx context.Context, className string, properties map[string]any, vector []float32) (any, error) {
	ctx, span := observability.StartSpan(ctx, "weaviatex.CreateObject",
		attribute.String("class", className),
	)
	defer span.End()

	obj, err := w.ops.opsCreateObject(ctx, className, properties, vector)
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

	err := w.ops.opsDeleteObject(ctx, className, id)
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

	err := w.ops.opsCreateClass(ctx, class)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (w *Weaviatex) IsLive(ctx context.Context) (bool, error) {
	ctx, span := observability.StartSpan(ctx, "weaviatex.IsLive")
	defer span.End()

	live, err := w.ops.opsIsLive(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return live, err
}

// Close evicts this instance from the package-level cache. The Weaviate
// client does not expose a network-level Close (HTTP/2 long-lived
// connections are managed by the underlying http.Client), so this is a
// no-op on the network side and a one-line cache eviction. Safe to call
// multiple times: subsequent calls return nil.
//
// Without the eviction, the next GetWeaviate(name) would hand back the
// same *Weaviatex with a stale APIKey / Host / Scheme — evicting forces
// a fresh client (e.g. after a config reload), matching the
// lazy-build-on-first-use contract of the package.
func (w *Weaviatex) Close() error {
	if w.cfg == nil {
		return nil
	}
	clientCache.Delete(w.cfg.Name)
	if v, ok := weaviateForcedCloseErr.LoadAndDelete(w.cfg.Name); ok {
		// Comma-ok on purpose: a non-error value would panic the
		// process. The sync.Map is test-only (production never
		// stores anything), so this should always be an error,
		// but defending against bad test fixtures is cheap.
		if e, ok := v.(error); ok {
			return e
		}
	}
	return nil
}
