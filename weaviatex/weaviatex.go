package weaviatex

import (
	"fmt"
	"sync"

	"github.com/gospacex/vectorx/config"
)

var (
	clientCache sync.Map
	clientLocks sync.Map
)

func GetWeaviate(name string) (*Weaviatex, error) {
	lockAny, _ := clientLocks.LoadOrStore(name, &sync.Mutex{})
	lock := lockAny.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	if v, ok := clientCache.Load(name); ok {
		return v.(*Weaviatex), nil
	}

	cfg, err := loadConfig(name)
	if err != nil {
		return nil, fmt.Errorf("weaviatex %q: %w", name, err)
	}

	c, err := newClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("weaviatex %q: %w", name, err)
	}

	clientCache.Store(name, c)
	return c, nil
}

func MustGetWeaviate(name string) *Weaviatex {
	c, err := GetWeaviate(name)
	if err != nil {
		panic(fmt.Sprintf("weaviatex %q: %v", name, err))
	}
	return c
}

// New constructs a *Weaviatex directly from a config, bypassing the
// package-level cache. Intended for hubx-style injection where the
// caller already owns a parsed config map. The returned client is not
// stored in clientCache and must be Closed by the caller.
//
// Weaviate's SDK builds the client eagerly but only opens HTTP
// connections on the first request, so this call rarely fails — the
// closest failure mode is an invalid scheme/host combination.
func New(cfg *config.WeaviateConfig) (*Weaviatex, error) {
	if cfg == nil {
		return nil, fmt.Errorf("weaviatex: nil config")
	}
	return newClient(cfg)
}
