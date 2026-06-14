package weaviatex

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

func loadConfig(name string) (*config.WeaviateConfig, error) {
	cfg, err := getGlobalConfig()
	if err != nil {
		return nil, err
	}
	for i := range cfg.VectorX.Weaviate {
		if cfg.VectorX.Weaviate[i].Name == name {
			return &cfg.VectorX.Weaviate[i], nil
		}
	}
	return nil, fmt.Errorf("weaviate config %q not found in config", name)
}

func getGlobalConfig() (*config.Config, error) {
	var err error
	loadOnce.Do(func() {
		if configPath == "" {
			configPath = "mq.yaml"
		}
		globalCfg, err = config.Load(configPath)
	})
	return globalCfg, err
}
