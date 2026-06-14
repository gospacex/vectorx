package weaviatex

import (
	"fmt"
	"sync"
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
