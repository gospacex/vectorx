package milvusx

import (
	"sync"
	"testing"

	"github.com/gospacex/vectorx/config"
)

// TestCloseAll_ClosesEveryCachedEntry is the regression test for the
// "Runtime.Close never propagated to adapter caches" gap. Before this
// code existed, Runtime.Close flushed OTel and that's all — the cached
// *Milvusx instances kept their gRPC connections open until the OS
// reaped the process. After the fix, Runtime.Close calls
// milvusx.CloseAll which evicts every entry.
func TestCloseAll_ClosesEveryCachedEntry(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	// Two entries, neither has a real conn (cfg-only) — CloseAll must
	// evict them anyway. We don't test the gRPC close path here
	// because the regression we're guarding is the *eviction* step.
	m1 := &Milvusx{cfg: &config.MilvusConfig{Name: "one"}}
	m2 := &Milvusx{cfg: &config.MilvusConfig{Name: "two"}}
	clientCache.Store("one", m1)
	clientCache.Store("two", m2)

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

// TestCloseAll_EmptyCache_NoError asserts CloseAll on an empty cache is
// a no-op success — this is the common case for services that never
// reach a vector adapter (e.g. an HTTP server that only uses
// observability).
func TestCloseAll_EmptyCache_NoError(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	if err := CloseAll(); err != nil {
		t.Fatalf("CloseAll on empty cache: %v", err)
	}
}
