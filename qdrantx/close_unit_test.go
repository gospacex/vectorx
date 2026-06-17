package qdrantx

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/gospacex/vectorx/config"
)

// TestCloseAll_EmptyCache_NoOp mirrors the milvusx version: calling
// CloseAll on an empty cache is a no-op success path. This is the
// common case for services that never touch the qdrant adapter.
func TestCloseAll_EmptyCache_NoOp(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	if err := CloseAll(); err != nil {
		t.Fatalf("CloseAll on empty cache: %v", err)
	}
}

// TestCloseAll_ClosesEveryCachedEntry verifies the eviction step:
// entries whose Close returns nil must be removed from the cache
// afterwards, so the next GetQdrant(name) constructs a fresh client
// instead of handing back the same *Qdrantx with a dead
// *grpc.ClientConn. Mirrors the milvusx regression test.
func TestCloseAll_ClosesEveryCachedEntry(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	q1 := &Qdrantx{cfg: &config.QdrantConfig{Name: "one"}, conn: nil}
	q2 := &Qdrantx{cfg: &config.QdrantConfig{Name: "two"}, conn: nil}
	clientCache.Store("one", q1)
	clientCache.Store("two", q2)

	if err := CloseAll(); err != nil {
		t.Fatalf("CloseAll: %v", err)
	}
	if _, ok := clientCache.Load("one"); ok {
		t.Fatal("entry 'one' not evicted")
	}
	if _, ok := clientCache.Load("two"); ok {
		t.Fatal("entry 'two' not evicted")
	}
}

// TestCloseAll_AggregatesErrors ensures the joined error contract:
// if any entry's Close returns a non-nil error, that error is wrapped
// with %w (preserving the errors.Is chain) and joined with others.
// This is the regression test for the "joined error loses the
// sentinel" bug that the milvusx close.go fix addresses — the
// same shape must hold in qdrantx so vectorx.Runtime.Close's joined
// error is consistent across adapters.
func TestCloseAll_AggregatesErrors(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	qdrantForcedCloseErr = sync.Map{}

	sentinel := errors.New("forced-qdrant-close-error")
	qdrantForcedCloseErr.Store("fail", sentinel)
	clientCache.Store("fail", &Qdrantx{cfg: &config.QdrantConfig{Name: "fail"}, conn: nil})

	// Plant a healthy entry too — CloseAll must drain both even
	// when one fails.
	clientCache.Store("ok", &Qdrantx{cfg: &config.QdrantConfig{Name: "ok"}, conn: nil})

	err := CloseAll()
	if err == nil {
		t.Fatal("expected non-nil joined error")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("joined err = %v, want errors.Is(sentinel)", err)
	}
	// Both entries must be drained — even on failure, the
	// best-effort cleanup removes them so the next Get doesn't
	// hand back a half-closed client.
	if _, ok := clientCache.Load("fail"); ok {
		t.Fatal("failing entry not evicted")
	}
	if _, ok := clientCache.Load("ok"); ok {
		t.Fatal("ok entry not evicted")
	}
}

// TestCloseAll_FormatsWithNamePrefix verifies the joined error
// message carries the cache entry's name so logs from production
// services can be triaged by client name. The contract is a
// "%s: %w" format from CloseAll; we only assert the name appears
// in the joined error's Error() output.
func TestCloseAll_FormatsWithNamePrefix(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	qdrantForcedCloseErr = sync.Map{}

	sentinel := errors.New("forced-qdrant-close-error")
	qdrantForcedCloseErr.Store("alpha", sentinel)
	clientCache.Store("alpha", &Qdrantx{cfg: &config.QdrantConfig{Name: "alpha"}, conn: nil})

	err := CloseAll()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "alpha") {
		t.Errorf("err message %q missing entry name 'alpha'", err.Error())
	}
}

// TestQdrantx_Close_HonoursForcedError verifies the per-entry
// forcedCloseErr override is consulted by Close (not just CloseAll).
// This is the same regression-test machinery the milvusx adapter
// uses; keeping it symmetrical across adapters makes the
// Runtime.Close contract testable from one place.
func TestQdrantx_Close_HonoursForcedError(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	qdrantForcedCloseErr = sync.Map{}

	sentinel := errors.New("forced-qdrant-single-close-error")
	q := &Qdrantx{cfg: &config.QdrantConfig{Name: "single"}, conn: nil}
	clientCache.Store("single", q)
	qdrantForcedCloseErr.Store("single", sentinel)

	if err := q.Close(); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want errors.Is(sentinel)", err)
	}
	// And a second close on the same *Qdrantx must be a no-op.
	if err := q.Close(); err != nil {
		t.Fatalf("second Close err = %v, want nil", err)
	}
}