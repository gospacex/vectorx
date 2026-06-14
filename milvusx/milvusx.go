package milvusx

import (
	"fmt"
	"sync"
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
