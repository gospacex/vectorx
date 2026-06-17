package qdrantx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/observability"
	qdrant "github.com/qdrant/go-client/qdrant"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type Qdrantx struct {
	client qdrant.PointsClient
	cfg    *config.QdrantConfig
	conn   *grpc.ClientConn
}

// newClient builds a gRPC connection to Qdrant. TLS posture is driven by
// cfg.TLS — when true, the gRPC channel uses server-authenticated TLS with
// the trust store augmented by cfg.CAFile (private CAs) and cfg.ServerName
// overriding hostname verification. cfg.InsecureSkipVerify short-circuits
// the chain check (self-signed clusters; NOT recommended for production).
//
// cfg.TLS == false preserves the legacy plaintext behaviour. This is the
// only branch that uses insecure.NewCredentials; every secure path uses
// credentials.NewTLS so a misconfigured YAML never silently downgrades to
// plaintext — if cfg.TLS is true and the trust store cannot be built, the
// error is returned at startup, not at first RPC.
func newClient(cfg *config.QdrantConfig) (*Qdrantx, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	opts, err := dialOptions(cfg)
	if err != nil {
		return nil, fmt.Errorf("build dial options: %w", err)
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return &Qdrantx{
		client: qdrant.NewPointsClient(conn),
		cfg:    cfg,
		conn:   conn,
	}, nil
}

// dialOptions is split out so the TLS construction can be unit-tested
// independently from the gRPC dial itself. Returns insecure credentials
// when cfg.TLS is false (legacy default); tls credentials otherwise.
func dialOptions(cfg *config.QdrantConfig) ([]grpc.DialOption, error) {
	if !cfg.TLS {
		return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, nil
	}
	tlsCfg := &tls.Config{
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // gated on explicit YAML opt-in
		MinVersion:         tls.VersionTLS12,
	}
	if cfg.CAFile != "" {
		pool, err := loadCAPool(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		tlsCfg.RootCAs = pool
	}
	return []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))}, nil
}

// loadCAPool reads a PEM bundle from path and returns a CertPool ready
// to plug into a tls.Config. Returns an error when the file cannot be
// read or no certificates were parsed — the latter catches "I dropped
// the wrong file at /etc/qdrant/ca.pem" instead of silently trusting
// the system pool only.
func loadCAPool(path string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CA file %q: %w", path, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("CA file %q: no certificates parsed (not a valid PEM bundle?)", path)
	}
	return pool, nil
}

// All public methods wrap the corresponding qdrant.PointsClient call with
// observability.StartSpan so the OTel pipeline captures every gRPC
// round-trip with the right span name + collection attribute. Errors
// are recorded on the span but not wrapped — caller's errors.Is
// checks against the raw grpc status must still work.

// Write paths

func (q *Qdrantx) Upsert(ctx context.Context, in *qdrant.UpsertPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Upsert", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.Upsert(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) UpdateVectors(ctx context.Context, in *qdrant.UpdatePointVectors, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.UpdateVectors", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.UpdateVectors(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) DeleteVectors(ctx context.Context, in *qdrant.DeletePointVectors, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.DeleteVectors", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.DeleteVectors(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) SetPayload(ctx context.Context, in *qdrant.SetPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.SetPayload", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.SetPayload(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) OverwritePayload(ctx context.Context, in *qdrant.SetPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.OverwritePayload", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.OverwritePayload(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) DeletePayload(ctx context.Context, in *qdrant.DeletePayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.DeletePayload", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.DeletePayload(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) ClearPayload(ctx context.Context, in *qdrant.ClearPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.ClearPayload", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.ClearPayload(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) Delete(ctx context.Context, in *qdrant.DeletePoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Delete", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.Delete(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) UpdateBatch(ctx context.Context, in *qdrant.UpdateBatchPoints, opts ...grpc.CallOption) (*qdrant.UpdateBatchResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.UpdateBatch", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.UpdateBatch(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

// Index paths

func (q *Qdrantx) CreateFieldIndex(ctx context.Context, in *qdrant.CreateFieldIndexCollection, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.CreateFieldIndex", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.CreateFieldIndex(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) DeleteFieldIndex(ctx context.Context, in *qdrant.DeleteFieldIndexCollection, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.DeleteFieldIndex", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.DeleteFieldIndex(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) CreateVectorName(ctx context.Context, in *qdrant.CreateVectorNameRequest, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.CreateVectorName", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.CreateVectorName(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) DeleteVectorName(ctx context.Context, in *qdrant.DeleteVectorNameRequest, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.DeleteVectorName", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.DeleteVectorName(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

// Read paths

func (q *Qdrantx) Get(ctx context.Context, in *qdrant.GetPoints, opts ...grpc.CallOption) (*qdrant.GetResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Get", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.Get(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) Scroll(ctx context.Context, in *qdrant.ScrollPoints, opts ...grpc.CallOption) (*qdrant.ScrollResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Scroll", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.Scroll(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) Count(ctx context.Context, in *qdrant.CountPoints, opts ...grpc.CallOption) (*qdrant.CountResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Count", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.Count(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) Search(ctx context.Context, in *qdrant.SearchPoints, opts ...grpc.CallOption) (*qdrant.SearchResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Search",
		attribute.String("collection", in.GetCollectionName()),
		attribute.Int("limit", int(in.GetLimit())),
	)
	defer span.End()
	res, err := q.client.Search(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) SearchBatch(ctx context.Context, in *qdrant.SearchBatchPoints, opts ...grpc.CallOption) (*qdrant.SearchBatchResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.SearchBatch", attribute.Int("searches", len(in.GetSearchPoints())))
	defer span.End()
	res, err := q.client.SearchBatch(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) SearchGroups(ctx context.Context, in *qdrant.SearchPointGroups, opts ...grpc.CallOption) (*qdrant.SearchGroupsResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.SearchGroups", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.SearchGroups(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) Recommend(ctx context.Context, in *qdrant.RecommendPoints, opts ...grpc.CallOption) (*qdrant.RecommendResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Recommend", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.Recommend(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) RecommendBatch(ctx context.Context, in *qdrant.RecommendBatchPoints, opts ...grpc.CallOption) (*qdrant.RecommendBatchResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.RecommendBatch", attribute.Int("recommends", len(in.GetRecommendPoints())))
	defer span.End()
	res, err := q.client.RecommendBatch(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) RecommendGroups(ctx context.Context, in *qdrant.RecommendPointGroups, opts ...grpc.CallOption) (*qdrant.RecommendGroupsResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.RecommendGroups", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.RecommendGroups(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) Discover(ctx context.Context, in *qdrant.DiscoverPoints, opts ...grpc.CallOption) (*qdrant.DiscoverResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Discover", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.Discover(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) DiscoverBatch(ctx context.Context, in *qdrant.DiscoverBatchPoints, opts ...grpc.CallOption) (*qdrant.DiscoverBatchResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.DiscoverBatch", attribute.Int("discovers", len(in.GetDiscoverPoints())))
	defer span.End()
	res, err := q.client.DiscoverBatch(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (q *Qdrantx) Query(ctx context.Context, in *qdrant.QueryPoints, opts ...grpc.CallOption) (*qdrant.QueryResponse, error) {
	ctx, span := observability.StartSpan(ctx, "qdrantx.Query", attribute.String("collection", in.GetCollectionName()))
	defer span.End()
	res, err := q.client.Query(ctx, in, opts...)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

// Close releases the underlying gRPC connection and evicts this instance
// from the package-level cache. Without the eviction, the next
// GetQdrant(name) call would hand back the same *Qdrantx with a
// dead *grpc.ClientConn — every subsequent RPC would fail with
// `grpc: the client connection is closing`. Evicting from the cache
// forces a fresh client (and fresh conn) on the next Get, matching
// the lazy-build-on-first-use contract of the package.
//
// The cache eviction runs before the conn-close attempt so that
// callers who use a zero-value *Qdrantx (e.g. unit tests that drive
// the cache directly) still get the eviction. The conn-close
// short-circuits when the embedded conn is nil.
//
// Safe to call multiple times; the second call sees q.conn == nil
// and a cache miss, both of which are no-ops.
func (q *Qdrantx) Close() error {
	if q.cfg != nil {
		clientCache.Delete(q.cfg.Name)
		if v, ok := qdrantForcedCloseErr.LoadAndDelete(q.cfg.Name); ok {
			// Comma-ok on purpose: a non-error value would panic the
			// process. The sync.Map is test-only (production never
			// stores anything), so this should always be an error,
			// but defending against bad test fixtures is cheap.
			if e, ok := v.(error); ok {
				return e
			}
		}
	}
	if q.conn == nil {
		return nil
	}
	err := q.conn.Close()
	q.conn = nil
	return err
}
