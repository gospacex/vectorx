package weaviatex

import (
	"sync"

	"github.com/gospacex/vectorx/config"
)

// PlantForTest stores a sentinel *Weaviatex under name. Used by
// vectorx_close_test.go to verify Runtime.Close cascades to the
// weaviatex cache. The sentinel has no SDK client (weaviate.NewClient
// needs a real host) — the regression we're guarding is the
// *eviction* step.
func PlantForTest(name string) {
	clientCache.Store(name, &Weaviatex{cfg: &config.WeaviateConfig{Name: name}})
}

// LookupForTest returns the cached entry for name and whether it was
// present. Used by the same parent-package test to assert the cache
// was drained by Runtime.Close.
func LookupForTest(name string) (*Weaviatex, bool) {
	v, ok := clientCache.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*Weaviatex), true
}

// weaviateForcedCloseErr is the per-name error override consulted by
// Close(). Tests in close_unit_test.go store a sentinel here, and
// the next CloseAll (or single Close) reports it via errors.Is.
//
// The variable is a value (not a pointer) because the test setup
// reassigns it with `weaviateForcedCloseErr = sync.Map{}` to get
// isolation between subtests — every test starts with a fresh map.
var weaviateForcedCloseErr sync.Map
