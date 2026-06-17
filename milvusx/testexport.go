package milvusx

import (
	"sync"

	"github.com/gospacex/vectorx/config"
)

// The *ForTest helpers below are exported so the parent vectorx package's
// tests can plant/inspect cache entries without dialing a real Milvus.
// They are intentionally tiny and clearly named so a grep for "ForTest"
// surfaces all of them in code review.
//
// PlantForTest stores a sentinel *Milvusx under name. Used by
// vectorx_close_test.go to verify Runtime.Close cascades to the
// milvusx cache.
func PlantForTest(name string) {
	clientCache.Store(name, &Milvusx{cfg: &config.MilvusConfig{Name: name}})
}

// LookupForTest returns the cached entry for name and whether it was
// present. Used by the same parent-package test to assert the cache
// was drained by Runtime.Close.
func LookupForTest(name string) (*Milvusx, bool) {
	v, ok := clientCache.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*Milvusx), true
}

// forcedCloseErr is the per-name error override consulted by Close().
// PlantFailingForTest stores a sentinel here, and the next CloseAll —
// and thus the next Runtime.Close — reports it via errors.Is. The
// sync.Map keeps the override zero-cost on the hot path (a single
// Load for every Close) and lets tests plant multiple failing
// entries without races.
var forcedCloseErr sync.Map

// ErrForcedClose is the sentinel error returned by Milvusx.Close when
// the entry was planted by PlantFailingForTest. Used to verify that
// Runtime.Close's joined error preserves it via errors.Is.
var ErrForcedClose = errForcedClose

// errForcedClose is the underlying value; exposed as a separate var
// (not a const) so tests in other packages can compare with ==
// without going through errors.Is. The exported ErrForcedClose is the
// errors.Is-friendly sentinel.
var errForcedClose = forcedCloseErrSentinel{}

type forcedCloseErrSentinel struct{}

func (forcedCloseErrSentinel) Error() string { return "milvusx: forced close error (test)" }

// PlantFailingForTest stores a *Milvusx whose Close returns err. The
// next CloseAll — and thus the next Runtime.Close — will report err
// via errors.Is / errors.As. After this returns, errors.Is(returned
// error, ErrForcedClose) is true for any err that itself equals
// ErrForcedClose or wraps it; for an arbitrary error (e.g.
// errors.New("forced-milvus-close-error")) callers compare with
// errors.Is using the *Milvusx entry's name as the lookup key — see
// the Runtime.Close join semantics for the joined-error contract.
func PlantFailingForTest(name string, err error) {
	clientCache.Store(name, &Milvusx{cfg: &config.MilvusConfig{Name: name}})
	forcedCloseErr.Store(name, err)
}
