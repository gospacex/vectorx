package qdrantx

import (
	"fmt"
	"sync"

	"github.com/gospacex/vectorx/config"
)

var (
	clientCache sync.Map
	clientLocks sync.Map
)

func GetQdrant(name string) (*Qdrantx, error) {
	lockAny, _ := clientLocks.LoadOrStore(name, &sync.Mutex{})
	lock := lockAny.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	if v, ok := clientCache.Load(name); ok {
		return v.(*Qdrantx), nil
	}

	cfg, err := loadConfig(name)
	if err != nil {
		return nil, fmt.Errorf("qdrantx %q: %w", name, err)
	}

	c, err := newClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("qdrantx %q: %w", name, err)
	}

	clientCache.Store(name, c)
	return c, nil
}

func MustGetQdrant(name string) *Qdrantx {
	c, err := GetQdrant(name)
	if err != nil {
		panic(fmt.Sprintf("qdrantx %q: %v", name, err))
	}
	return c
}

// New constructs a *Qdrantx directly from a config, bypassing the
// package-level cache. Intended for hubx-style injection where the
// caller already owns a parsed config map. The returned client is not
// stored in clientCache and must be Closed by the caller.
//
// newClient opens the gRPC connection eagerly; the same error
// semantics as GetQdrant apply.
func New(cfg *config.QdrantConfig) (*Qdrantx, error) {
	if cfg == nil {
		return nil, fmt.Errorf("qdrantx: nil config")
	}
	return newClient(cfg)
}
