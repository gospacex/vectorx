package weaviatex

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/gospacex/vectorx/config"
)

// TestCloseAll_EmptyCache_NoOp mirrors milvusx/qdrantx: calling
// CloseAll on an empty cache is a no-op success path.
func TestCloseAll_EmptyCache_NoOp(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	if err := CloseAll(); err != nil {
		t.Fatalf("CloseAll on empty cache: %v", err)
	}
}

// TestCloseAll_ClosesEveryCachedEntry verifies the eviction step:
// every entry must be removed from the cache afterwards, so the
// next GetWeaviate(name) constructs a fresh client instead of
// handing back a stale *Weaviatex (e.g. one whose APIKey was
// rotated and the in-memory copy is now wrong).
func TestCloseAll_ClosesEveryCachedEntry(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	w1 := &Weaviatex{cfg: &config.WeaviateConfig{Name: "one"}}
	w2 := &Weaviatex{cfg: &config.WeaviateConfig{Name: "two"}}
	clientCache.Store("one", w1)
	clientCache.Store("two", w2)

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
// if any entry's Close returns a non-nil error, that error is
// wrapped with %w (preserving the errors.Is chain) and joined with
// others. Mirrors the qdrantx/milvusx contracts — vectorx.Runtime.Close
// depends on a uniform shape across adapters.
func TestCloseAll_AggregatesErrors(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	weaviateForcedCloseErr = sync.Map{}

	sentinel := errors.New("forced-weaviate-close-error")
	weaviateForcedCloseErr.Store("fail", sentinel)
	clientCache.Store("fail", &Weaviatex{cfg: &config.WeaviateConfig{Name: "fail"}})

	// Plant a healthy entry too — CloseAll must drain both.
	clientCache.Store("ok", &Weaviatex{cfg: &config.WeaviateConfig{Name: "ok"}})

	err := CloseAll()
	if err == nil {
		t.Fatal("expected non-nil joined error")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("joined err = %v, want errors.Is(sentinel)", err)
	}
	if _, ok := clientCache.Load("fail"); ok {
		t.Fatal("failing entry not evicted")
	}
	if _, ok := clientCache.Load("ok"); ok {
		t.Fatal("ok entry not evicted")
	}
}

// TestCloseAll_FormatsWithNamePrefix verifies the joined error
// message carries the cache entry's name. See qdrantx/milvusx
// versions for the rationale.
func TestCloseAll_FormatsWithNamePrefix(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	weaviateForcedCloseErr = sync.Map{}

	sentinel := errors.New("forced-weaviate-close-error")
	weaviateForcedCloseErr.Store("alpha", sentinel)
	clientCache.Store("alpha", &Weaviatex{cfg: &config.WeaviateConfig{Name: "alpha"}})

	err := CloseAll()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "alpha") {
		t.Errorf("err message %q missing entry name 'alpha'", err.Error())
	}
}

// TestWeaviatex_Close_HonoursForcedError verifies the per-entry
// forcedCloseErr override is consulted by Close. Mirrors the
// qdrantx/milvusx regression test fixtures.
func TestWeaviatex_Close_HonoursForcedError(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}
	weaviateForcedCloseErr = sync.Map{}

	sentinel := errors.New("forced-weaviate-single-close-error")
	w := &Weaviatex{cfg: &config.WeaviateConfig{Name: "single"}}
	clientCache.Store("single", w)
	weaviateForcedCloseErr.Store("single", sentinel)

	if err := w.Close(); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want errors.Is(sentinel)", err)
	}
	// And a second close on the same *Weaviatex must be a no-op.
	if err := w.Close(); err != nil {
		t.Fatalf("second Close err = %v, want nil", err)
	}
}

// TestWeaviatex_Close_NilCfg verifies the no-config Close path
// returns nil instead of panic-ing. This is the path the zero-value
// *Weaviatex fixture (used by parent-package tests) takes.
func TestWeaviatex_Close_NilCfg(t *testing.T) {
	w := &Weaviatex{}
	if err := w.Close(); err != nil {
		t.Fatalf("Close with nil cfg err = %v, want nil", err)
	}
}