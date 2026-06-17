package qdrantx

import (
	"sync"

	"github.com/gospacex/vectorx/config"
)

// PlantForTest stores a sentinel *Qdrantx under name. Used by
// vectorx_close_test.go to verify Runtime.Close cascades to the
// qdrantx cache. The sentinel has no real gRPC connection — the
// regression we're guarding is the *eviction* step, not the
// underlying network teardown.
func PlantForTest(name string) {
	clientCache.Store(name, &Qdrantx{cfg: &config.QdrantConfig{Name: name}})
}

// LookupForTest returns the cached entry for name and whether it was
// present. Used by the same parent-package test to assert the cache
// was drained by Runtime.Close.
func LookupForTest(name string) (*Qdrantx, bool) {
	v, ok := clientCache.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*Qdrantx), true
}

// qdrantForcedCloseErr is the per-name error override consulted by
// Close(). plantQdrantFailing stores a sentinel here, and the next
// CloseAll reports it via errors.Is. sync.Map keeps the override
// zero-cost on the hot path (a single Load for every Close) and lets
// tests plant multiple failing entries without races.
//
// The test file (close_unit_test.go) reassigns this to a fresh
// sync.Map{} in its setup so tests are isolated. Production code does
// not touch it.
var qdrantForcedCloseErr sync.Map
