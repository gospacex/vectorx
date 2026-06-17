package milvusx

import (
	"fmt"
	"sync"

	"github.com/gospacex/vectorx/config"
)

var (
	clientCache sync.Map
	clientLocks sync.Map
)

func GetMilvus(name string) (*Milvusx, error) {
	lockAny, _ := clientLocks.LoadOrStore(name, &sync.Mutex{})
	lock := lockAny.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	if v, ok := clientCache.Load(name); ok {
		return v.(*Milvusx), nil
	}

	cfg, err := loadConfig(name)
	if err != nil {
		return nil, fmt.Errorf("milvusx %q: %w", name, err)
	}

	c, err := newClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("milvusx %q: %w", name, err)
	}

	clientCache.Store(name, c)
	return c, nil
}

func MustGetMilvus(name string) *Milvusx {
	c, err := GetMilvus(name)
	if err != nil {
		panic(fmt.Sprintf("milvusx %q: %v", name, err))
	}
	return c
}

// New constructs a *Milvusx directly from a config, bypassing the
// package-level cache. Intended for hubx-style injection where the
// caller already owns a parsed config map. The returned client is not
// stored in clientCache and must be Closed by the caller.
//
// The underlying newClient handles dialing the gRPC connection, so the
// same error semantics as GetMilvus apply (network errors surface
// here, not in Close).
func New(cfg *config.MilvusConfig) (*Milvusx, error) {
	if cfg == nil {
		return nil, fmt.Errorf("milvusx: nil config")
	}
	return newClient(cfg)
}
