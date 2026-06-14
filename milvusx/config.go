package milvusx

import (
	"fmt"
	"sync"

	"github.com/gospacex/vectorx/config"
)

var (
	loadOnce   sync.Once
	globalCfg  *config.Config
	configPath string
)

func SetConfigPath(path string) {
	configPath = path
	loadOnce = sync.Once{}
}

func loadConfig(name string) (*config.MilvusConfig, error) {
	cfg, err := getGlobalConfig()
	if err != nil {
		return nil, err
	}
	for i := range cfg.VectorX.Milvus {
		if cfg.VectorX.Milvus[i].Name == name {
			return &cfg.VectorX.Milvus[i], nil
		}
	}
	return nil, fmt.Errorf("milvus config %q not found in config", name)
}

func getGlobalConfig() (*config.Config, error) {
	var err error
	loadOnce.Do(func() {
		if configPath == "" {
			configPath = "mq.yaml"
		}
		globalCfg, err = config.Load(configPath)
	})
	if err != nil {
		return nil, err
	}
	if globalCfg == nil {
		return nil, fmt.Errorf("config loaded but returned nil")
	}
	return globalCfg, nil
}
