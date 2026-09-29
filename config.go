// vectorx/config.go — configx 集成 helpers
package vectorx

import (
	"fmt"

	"github.com/gospacex/configx"
)

// ReadStoreConfig 从 configx 读取单个 store 的 Config
func ReadStoreConfig(cfg configx.Config, storeName string) (*Config, error) {
	section := &VectorDBSection{}
	if err := cfg.Unmarshal("vectorx", section); err != nil {
		return nil, fmt.Errorf("vectorx: unmarshal vectorx section: %w", err)
	}
	if section.Stores == nil {
		return nil, fmt.Errorf("vectorx: no vectorx.stores configured")
	}
	store, ok := section.Stores[storeName]
	if !ok {
		return nil, fmt.Errorf("vectorx: store %q not found in vectorx.stores", storeName)
	}
	store.Name = storeName
	return store, nil
}

// ReadConfigAndCreate 从已有的 configx.Config 读取配置并创建客户端
func ReadConfigAndCreate(cfg configx.Config, storeName string) (CollectionClient, error) {
	storeCfg, err := ReadStoreConfig(cfg, storeName)
	if err != nil {
		return nil, err
	}
	return New(storeCfg)
}

// ReadConfigAndCreateWithOpts 通过 ConfigOptions 读取配置并创建客户端
func ReadConfigAndCreateWithOpts(storeName string, opts ...configx.ConfigOptions) (CollectionClient, error) {
	cfg, err := configx.ReadConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("vectorx: read config: %w", err)
	}
	return ReadConfigAndCreate(cfg, storeName)
}
