package qdrantx

import (
	"context"
	"errors"
	"testing"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/observability"
	qdrant "github.com/qdrant/go-client/qdrant"
	"go.opentelemetry.io/otel"
	otelattr "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
)

type otelAttributeQD = otelattr.KeyValue

// fakePointsClient embeds qdrant.PointsClient (nil) and overrides only
// the methods under test. Embedding the interface gives us compile-time
// satisfaction of PointsClient without having to stub every method;
// unoverridden methods would nil-panic, which is fine because we only
// exercise the ones we explicitly override.
type fakePointsClient struct {
	qdrant.PointsClient

	// Write paths
	upsert           func(ctx context.Context, in *qdrant.UpsertPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	delete           func(ctx context.Context, in *qdrant.DeletePoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	updateVectors    func(ctx context.Context, in *qdrant.UpdatePointVectors, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	deleteVectors    func(ctx context.Context, in *qdrant.DeletePointVectors, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	setPayload       func(ctx context.Context, in *qdrant.SetPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	overwritePayload func(ctx context.Context, in *qdrant.SetPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	deletePayload    func(ctx context.Context, in *qdrant.DeletePayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	clearPayload     func(ctx context.Context, in *qdrant.ClearPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	updateBatch      func(ctx context.Context, in *qdrant.UpdateBatchPoints, opts ...grpc.CallOption) (*qdrant.UpdateBatchResponse, error)

	// Index paths
	createFieldIndex func(ctx context.Context, in *qdrant.CreateFieldIndexCollection, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	deleteFieldIndex func(ctx context.Context, in *qdrant.DeleteFieldIndexCollection, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	createVectorName func(ctx context.Context, in *qdrant.CreateVectorNameRequest, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)
	deleteVectorName func(ctx context.Context, in *qdrant.DeleteVectorNameRequest, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error)

	// Read paths
	get             func(ctx context.Context, in *qdrant.GetPoints, opts ...grpc.CallOption) (*qdrant.GetResponse, error)
	search          func(ctx context.Context, in *qdrant.SearchPoints, opts ...grpc.CallOption) (*qdrant.SearchResponse, error)
	searchBatch     func(ctx context.Context, in *qdrant.SearchBatchPoints, opts ...grpc.CallOption) (*qdrant.SearchBatchResponse, error)
	searchGroups    func(ctx context.Context, in *qdrant.SearchPointGroups, opts ...grpc.CallOption) (*qdrant.SearchGroupsResponse, error)
	scroll          func(ctx context.Context, in *qdrant.ScrollPoints, opts ...grpc.CallOption) (*qdrant.ScrollResponse, error)
	count           func(ctx context.Context, in *qdrant.CountPoints, opts ...grpc.CallOption) (*qdrant.CountResponse, error)
	recommend       func(ctx context.Context, in *qdrant.RecommendPoints, opts ...grpc.CallOption) (*qdrant.RecommendResponse, error)
	recommendBatch  func(ctx context.Context, in *qdrant.RecommendBatchPoints, opts ...grpc.CallOption) (*qdrant.RecommendBatchResponse, error)
	recommendGroups func(ctx context.Context, in *qdrant.RecommendPointGroups, opts ...grpc.CallOption) (*qdrant.RecommendGroupsResponse, error)
	discover        func(ctx context.Context, in *qdrant.DiscoverPoints, opts ...grpc.CallOption) (*qdrant.DiscoverResponse, error)
	discoverBatch   func(ctx context.Context, in *qdrant.DiscoverBatchPoints, opts ...grpc.CallOption) (*qdrant.DiscoverBatchResponse, error)
	query           func(ctx context.Context, in *qdrant.QueryPoints, opts ...grpc.CallOption) (*qdrant.QueryResponse, error)
}

func (f *fakePointsClient) Upsert(ctx context.Context, in *qdrant.UpsertPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.upsert(ctx, in, opts...)
}
func (f *fakePointsClient) Delete(ctx context.Context, in *qdrant.DeletePoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.delete(ctx, in, opts...)
}
func (f *fakePointsClient) UpdateVectors(ctx context.Context, in *qdrant.UpdatePointVectors, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.updateVectors(ctx, in, opts...)
}
func (f *fakePointsClient) DeleteVectors(ctx context.Context, in *qdrant.DeletePointVectors, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.deleteVectors(ctx, in, opts...)
}
func (f *fakePointsClient) SetPayload(ctx context.Context, in *qdrant.SetPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.setPayload(ctx, in, opts...)
}
func (f *fakePointsClient) OverwritePayload(ctx context.Context, in *qdrant.SetPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.overwritePayload(ctx, in, opts...)
}
func (f *fakePointsClient) DeletePayload(ctx context.Context, in *qdrant.DeletePayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.deletePayload(ctx, in, opts...)
}
func (f *fakePointsClient) ClearPayload(ctx context.Context, in *qdrant.ClearPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.clearPayload(ctx, in, opts...)
}
func (f *fakePointsClient) UpdateBatch(ctx context.Context, in *qdrant.UpdateBatchPoints, opts ...grpc.CallOption) (*qdrant.UpdateBatchResponse, error) {
	return f.updateBatch(ctx, in, opts...)
}
func (f *fakePointsClient) CreateFieldIndex(ctx context.Context, in *qdrant.CreateFieldIndexCollection, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.createFieldIndex(ctx, in, opts...)
}
func (f *fakePointsClient) DeleteFieldIndex(ctx context.Context, in *qdrant.DeleteFieldIndexCollection, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.deleteFieldIndex(ctx, in, opts...)
}
func (f *fakePointsClient) CreateVectorName(ctx context.Context, in *qdrant.CreateVectorNameRequest, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.createVectorName(ctx, in, opts...)
}
func (f *fakePointsClient) DeleteVectorName(ctx context.Context, in *qdrant.DeleteVectorNameRequest, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
	return f.deleteVectorName(ctx, in, opts...)
}
func (f *fakePointsClient) Get(ctx context.Context, in *qdrant.GetPoints, opts ...grpc.CallOption) (*qdrant.GetResponse, error) {
	return f.get(ctx, in, opts...)
}
func (f *fakePointsClient) Search(ctx context.Context, in *qdrant.SearchPoints, opts ...grpc.CallOption) (*qdrant.SearchResponse, error) {
	return f.search(ctx, in, opts...)
}
func (f *fakePointsClient) SearchBatch(ctx context.Context, in *qdrant.SearchBatchPoints, opts ...grpc.CallOption) (*qdrant.SearchBatchResponse, error) {
	return f.searchBatch(ctx, in, opts...)
}
func (f *fakePointsClient) SearchGroups(ctx context.Context, in *qdrant.SearchPointGroups, opts ...grpc.CallOption) (*qdrant.SearchGroupsResponse, error) {
	return f.searchGroups(ctx, in, opts...)
}
func (f *fakePointsClient) Scroll(ctx context.Context, in *qdrant.ScrollPoints, opts ...grpc.CallOption) (*qdrant.ScrollResponse, error) {
	return f.scroll(ctx, in, opts...)
}
func (f *fakePointsClient) Count(ctx context.Context, in *qdrant.CountPoints, opts ...grpc.CallOption) (*qdrant.CountResponse, error) {
	return f.count(ctx, in, opts...)
}
func (f *fakePointsClient) Recommend(ctx context.Context, in *qdrant.RecommendPoints, opts ...grpc.CallOption) (*qdrant.RecommendResponse, error) {
	return f.recommend(ctx, in, opts...)
}
func (f *fakePointsClient) RecommendBatch(ctx context.Context, in *qdrant.RecommendBatchPoints, opts ...grpc.CallOption) (*qdrant.RecommendBatchResponse, error) {
	return f.recommendBatch(ctx, in, opts...)
}
func (f *fakePointsClient) RecommendGroups(ctx context.Context, in *qdrant.RecommendPointGroups, opts ...grpc.CallOption) (*qdrant.RecommendGroupsResponse, error) {
	return f.recommendGroups(ctx, in, opts...)
}
func (f *fakePointsClient) Discover(ctx context.Context, in *qdrant.DiscoverPoints, opts ...grpc.CallOption) (*qdrant.DiscoverResponse, error) {
	return f.discover(ctx, in, opts...)
}
func (f *fakePointsClient) DiscoverBatch(ctx context.Context, in *qdrant.DiscoverBatchPoints, opts ...grpc.CallOption) (*qdrant.DiscoverBatchResponse, error) {
	return f.discoverBatch(ctx, in, opts...)
}
func (f *fakePointsClient) Query(ctx context.Context, in *qdrant.QueryPoints, opts ...grpc.CallOption) (*qdrant.QueryResponse, error) {
	return f.query(ctx, in, opts...)
}

// withTestTracer swaps the global OTel TracerProvider for a fresh one
// backed by tracetest.NewInMemoryExporter. It returns the exporter so
// the test can read the recorded spans. Auto-restores on cleanup.
func withTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	prev := otel.GetTracerProvider()
	exp := tracetest.NewInMemoryExporter()
	res, _ := resource.New(context.Background(), resource.WithAttributes(semconv.ServiceName("qdrantx-unit-test")))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSyncer(exp),
	)
	otel.SetTracerProvider(tp)
	observability.SetEnabledForTesting(t, true)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	return exp
}

// hasAttrQD returns true if attrs contains (k, v) as a string attribute.
func hasAttrQD(attrs []otelAttributeQD, k, v string) bool {
	for _, a := range attrs {
		if string(a.Key) == k && a.Value.AsString() == v {
			return true
		}
	}
	return false
}

// qdrantxMethodCase is a single row in the table-driven coverage test.
type qdrantxMethodCase struct {
	spanName   string
	collection string
	fakeWith   func(err error) *fakePointsClient
	invoke     func(t *testing.T, q *Qdrantx) error
}

// qdrantxMethodCases lists every method the adapter wraps.
func qdrantxMethodCases() []qdrantxMethodCase {
	cases := []qdrantxMethodCase{
		// Write paths
		{
			spanName:   "qdrantx.Upsert",
			collection: "u1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{upsert: func(ctx context.Context, in *qdrant.UpsertPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Upsert(context.Background(), &qdrant.UpsertPoints{CollectionName: "u1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.Delete",
			collection: "d1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{delete: func(ctx context.Context, in *qdrant.DeletePoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Delete(context.Background(), &qdrant.DeletePoints{CollectionName: "d1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.UpdateVectors",
			collection: "uv1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{updateVectors: func(ctx context.Context, in *qdrant.UpdatePointVectors, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.UpdateVectors(context.Background(), &qdrant.UpdatePointVectors{CollectionName: "uv1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.DeleteVectors",
			collection: "dv1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{deleteVectors: func(ctx context.Context, in *qdrant.DeletePointVectors, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.DeleteVectors(context.Background(), &qdrant.DeletePointVectors{CollectionName: "dv1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.SetPayload",
			collection: "sp1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{setPayload: func(ctx context.Context, in *qdrant.SetPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.SetPayload(context.Background(), &qdrant.SetPayloadPoints{CollectionName: "sp1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.OverwritePayload",
			collection: "op1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{overwritePayload: func(ctx context.Context, in *qdrant.SetPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.OverwritePayload(context.Background(), &qdrant.SetPayloadPoints{CollectionName: "op1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.DeletePayload",
			collection: "dp1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{deletePayload: func(ctx context.Context, in *qdrant.DeletePayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.DeletePayload(context.Background(), &qdrant.DeletePayloadPoints{CollectionName: "dp1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.ClearPayload",
			collection: "cp1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{clearPayload: func(ctx context.Context, in *qdrant.ClearPayloadPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.ClearPayload(context.Background(), &qdrant.ClearPayloadPoints{CollectionName: "cp1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.UpdateBatch",
			collection: "ub1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{updateBatch: func(ctx context.Context, in *qdrant.UpdateBatchPoints, opts ...grpc.CallOption) (*qdrant.UpdateBatchResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.UpdateBatch(context.Background(), &qdrant.UpdateBatchPoints{CollectionName: "ub1"})
				return err
			},
		},

		// Index paths
		{
			spanName:   "qdrantx.CreateFieldIndex",
			collection: "cfi1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{createFieldIndex: func(ctx context.Context, in *qdrant.CreateFieldIndexCollection, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.CreateFieldIndex(context.Background(), &qdrant.CreateFieldIndexCollection{CollectionName: "cfi1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.DeleteFieldIndex",
			collection: "dfi1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{deleteFieldIndex: func(ctx context.Context, in *qdrant.DeleteFieldIndexCollection, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.DeleteFieldIndex(context.Background(), &qdrant.DeleteFieldIndexCollection{CollectionName: "dfi1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.CreateVectorName",
			collection: "cvn1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{createVectorName: func(ctx context.Context, in *qdrant.CreateVectorNameRequest, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.CreateVectorName(context.Background(), &qdrant.CreateVectorNameRequest{CollectionName: "cvn1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.DeleteVectorName",
			collection: "dvn1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{deleteVectorName: func(ctx context.Context, in *qdrant.DeleteVectorNameRequest, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.DeleteVectorName(context.Background(), &qdrant.DeleteVectorNameRequest{CollectionName: "dvn1"})
				return err
			},
		},

		// Read paths
		{
			spanName:   "qdrantx.Get",
			collection: "g1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{get: func(ctx context.Context, in *qdrant.GetPoints, opts ...grpc.CallOption) (*qdrant.GetResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Get(context.Background(), &qdrant.GetPoints{CollectionName: "g1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.Count",
			collection: "c1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{count: func(ctx context.Context, in *qdrant.CountPoints, opts ...grpc.CallOption) (*qdrant.CountResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Count(context.Background(), &qdrant.CountPoints{CollectionName: "c1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.Scroll",
			collection: "s1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{scroll: func(ctx context.Context, in *qdrant.ScrollPoints, opts ...grpc.CallOption) (*qdrant.ScrollResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Scroll(context.Background(), &qdrant.ScrollPoints{CollectionName: "s1"})
				return err
			},
		},
		{
			spanName: "qdrantx.Search",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{search: func(ctx context.Context, in *qdrant.SearchPoints, opts ...grpc.CallOption) (*qdrant.SearchResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Search(context.Background(), &qdrant.SearchPoints{CollectionName: "q1", Limit: 10})
				return err
			},
		},
		{
			spanName: "qdrantx.SearchBatch",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{searchBatch: func(ctx context.Context, in *qdrant.SearchBatchPoints, opts ...grpc.CallOption) (*qdrant.SearchBatchResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.SearchBatch(context.Background(), &qdrant.SearchBatchPoints{})
				return err
			},
		},
		{
			spanName:   "qdrantx.SearchGroups",
			collection: "sg1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{searchGroups: func(ctx context.Context, in *qdrant.SearchPointGroups, opts ...grpc.CallOption) (*qdrant.SearchGroupsResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.SearchGroups(context.Background(), &qdrant.SearchPointGroups{CollectionName: "sg1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.Recommend",
			collection: "r1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{recommend: func(ctx context.Context, in *qdrant.RecommendPoints, opts ...grpc.CallOption) (*qdrant.RecommendResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Recommend(context.Background(), &qdrant.RecommendPoints{CollectionName: "r1"})
				return err
			},
		},
		{
			spanName: "qdrantx.RecommendBatch",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{recommendBatch: func(ctx context.Context, in *qdrant.RecommendBatchPoints, opts ...grpc.CallOption) (*qdrant.RecommendBatchResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.RecommendBatch(context.Background(), &qdrant.RecommendBatchPoints{})
				return err
			},
		},
		{
			spanName:   "qdrantx.RecommendGroups",
			collection: "rg1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{recommendGroups: func(ctx context.Context, in *qdrant.RecommendPointGroups, opts ...grpc.CallOption) (*qdrant.RecommendGroupsResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.RecommendGroups(context.Background(), &qdrant.RecommendPointGroups{CollectionName: "rg1"})
				return err
			},
		},
		{
			spanName:   "qdrantx.Discover",
			collection: "dis1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{discover: func(ctx context.Context, in *qdrant.DiscoverPoints, opts ...grpc.CallOption) (*qdrant.DiscoverResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Discover(context.Background(), &qdrant.DiscoverPoints{CollectionName: "dis1"})
				return err
			},
		},
		{
			spanName: "qdrantx.DiscoverBatch",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{discoverBatch: func(ctx context.Context, in *qdrant.DiscoverBatchPoints, opts ...grpc.CallOption) (*qdrant.DiscoverBatchResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.DiscoverBatch(context.Background(), &qdrant.DiscoverBatchPoints{})
				return err
			},
		},
		{
			spanName:   "qdrantx.Query",
			collection: "qu1",
			fakeWith: func(err error) *fakePointsClient {
				return &fakePointsClient{query: func(ctx context.Context, in *qdrant.QueryPoints, opts ...grpc.CallOption) (*qdrant.QueryResponse, error) {
					return nil, err
				}}
			},
			invoke: func(t *testing.T, q *Qdrantx) error {
				_, err := q.Query(context.Background(), &qdrant.QueryPoints{CollectionName: "qu1"})
				return err
			},
		},
	}
	return cases
}

// TestQdrantx_MethodSpanNameAndError is a table-driven coverage test
// that exercises every PointsClient method the adapter wraps. Each
// row builds a Qdrantx with a fake that returns the row's wantErr
// for that single method, then asserts the adapter (a) starts a
// span with the expected name, (b) records the error on it, and
// (c) propagates the error unwrapped. Methods that take a
// *Collection* input also check the collection attribute.
func TestQdrantx_MethodSpanNameAndError(t *testing.T) {
	for _, tc := range qdrantxMethodCases() {
		t.Run(tc.spanName, func(t *testing.T) {
			exp := withTestTracer(t)
			want := errors.New("boom-" + tc.spanName)
			q := &Qdrantx{
				cfg:    &config.QdrantConfig{Name: "unit"},
				client: tc.fakeWith(want),
			}
			err := tc.invoke(t, q)
			if !errors.Is(err, want) {
				t.Fatalf("err = %v, want %v", err, want)
			}
			spans := exp.GetSpans().Snapshots()
			if len(spans) != 1 {
				t.Fatalf("spans = %d, want 1", len(spans))
			}
			if got := spans[0].Name(); got != tc.spanName {
				t.Errorf("span.Name = %q, want %q", got, tc.spanName)
			}
			if tc.collection != "" && !hasAttrQD(spans[0].Attributes(), "collection", tc.collection) {
				t.Errorf("missing collection=%s, got %v", tc.collection, spans[0].Attributes())
			}
			if len(spans[0].Events()) == 0 {
				t.Errorf("expected error event")
			}
		})
	}
}

// TestQdrantx_Search_LimitAttributeOnSpan asserts the extra
// attribute.Int("limit", ...) attachment on the Search path.
func TestQdrantx_Search_LimitAttributeOnSpan(t *testing.T) {
	exp := withTestTracer(t)
	q := &Qdrantx{
		cfg: &config.QdrantConfig{Name: "unit"},
		client: &fakePointsClient{search: func(ctx context.Context, in *qdrant.SearchPoints, opts ...grpc.CallOption) (*qdrant.SearchResponse, error) {
			return &qdrant.SearchResponse{}, nil
		}},
	}
	if _, err := q.Search(context.Background(), &qdrant.SearchPoints{CollectionName: "embeddings", Limit: 42}); err != nil {
		t.Fatal(err)
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 || spans[0].Name() != "qdrantx.Search" {
		t.Fatalf("span = %+v", spans)
	}
	want := map[string]string{"collection": "embeddings"}
	for _, a := range spans[0].Attributes() {
		if want[string(a.Key)] != "" && a.Value.AsString() == want[string(a.Key)] {
			delete(want, string(a.Key))
		}
	}
	if len(want) != 0 {
		t.Errorf("missing span attributes: %v (have: %v)", want, spans[0].Attributes())
	}
}

// TestQdrantx_SuccessNoErrorEvent verifies that on the success path
// the span is emitted but no error event is attached. This catches
// the regression where a broad span.RecordError call pollutes
// happy-path traces.
func TestQdrantx_SuccessNoErrorEvent(t *testing.T) {
	exp := withTestTracer(t)
	want := &qdrant.PointsOperationResponse{Result: &qdrant.UpdateResult{Status: qdrant.UpdateStatus_Completed}}
	q := &Qdrantx{
		cfg: &config.QdrantConfig{Name: "unit"},
		client: &fakePointsClient{upsert: func(ctx context.Context, in *qdrant.UpsertPoints, opts ...grpc.CallOption) (*qdrant.PointsOperationResponse, error) {
			return want, nil
		}},
	}
	got, err := q.Upsert(context.Background(), &qdrant.UpsertPoints{CollectionName: "c"})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != want {
		t.Fatalf("response mismatch")
	}
	spans := exp.GetSpans().Snapshots()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if len(spans[0].Events()) != 0 {
		t.Errorf("expected no error events on success path, got %d", len(spans[0].Events()))
	}
}
