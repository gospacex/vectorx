package vectorx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gospacex/vectorx/milvusx"
	"github.com/gospacex/vectorx/qdrantx"
	"github.com/gospacex/vectorx/weaviatex"
)

// TestClose_CascadesToAdapterCaches is the regression test for the
// "Runtime.Close never propagated to adapter caches" gap. Before this
// test existed, the only thing Runtime.Close did was flush OTel — the
// cached *Milvusx / *Qdrantx / *Weaviatex instances kept their
// connections open until process exit. After the fix, every adapter
// package exposes CloseAll() and Runtime.Close calls each one.
//
// The test populates the per-adapter caches directly (we don't actually
// dial a real server) and asserts they all drain on Runtime.Close.
func TestClose_CascadesToAdapterCaches(t *testing.T) {
	// Reset package-level caches in case a previous test left entries.
	milvusx.CloseAll()
	qdrantx.CloseAll()
	weaviatex.CloseAll()

	// Plant one entry in each cache. Using sentinel-only entries means
	// no real gRPC dial — we're verifying the eviction step, not the
	// underlying network teardown.
	milvusx.PlantForTest("m-cascade")
	qdrantx.PlantForTest("q-cascade")
	weaviatex.PlantForTest("w-cascade")

	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Caches must be drained.
	if _, ok := milvusx.LookupForTest("m-cascade"); ok {
		t.Fatal("milvusx cache not drained after Runtime.Close")
	}
	if _, ok := qdrantx.LookupForTest("q-cascade"); ok {
		t.Fatal("qdrantx cache not drained after Runtime.Close")
	}
	if _, ok := weaviatex.LookupForTest("w-cascade"); ok {
		t.Fatal("weaviatex cache not drained after Runtime.Close")
	}
}

// TestClose_AggregatesAdapterErrors verifies the new aggregator path:
// if any CloseAll returns a non-nil error, Runtime.Close's joined error
// wraps it so callers can still inspect it via errors.Is / errors.As.
func TestClose_AggregatesAdapterErrors(t *testing.T) {
	milvusx.CloseAll()
	qdrantx.CloseAll()
	weaviatex.CloseAll()

	// Wrap the sentinel so CloseAll's %w join preserves both layers:
	// errors.Is(returned, milvusx.ErrForcedClose) matches the inner
	// sentinel, while errors.Is(returned, inner) matches the custom
	// cause. This is the contract Runtime.Close callers depend on.
	milvusx.PlantFailingForTest("m-fail", fmt.Errorf("forced-milvus-close-error: %w", milvusx.ErrForcedClose))

	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	closeErr := rt.Close()
	if closeErr == nil {
		t.Fatal("expected non-nil error from Close when adapter close fails")
	}
	if !errors.Is(closeErr, milvusx.ErrForcedClose) {
		t.Fatalf("err = %v, want errors.Is(milvusx.ErrForcedClose)", closeErr)
	}

	// Clean up the test sentinel so subsequent tests don't see it.
	milvusx.CloseAll()
}

// TestClose_IdempotentWithAdapterCaches verifies that double-Close is
// safe even when adapter caches were drained on the first call.
func TestClose_IdempotentWithAdapterCaches(t *testing.T) {
	milvusx.CloseAll()
	qdrantx.CloseAll()
	weaviatex.CloseAll()
	milvusx.PlantForTest("m-idempotent")

	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := rt.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
