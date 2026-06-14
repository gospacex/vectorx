package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level vectorx config. Mirrors the mqx+vectorx shape.
type Config struct {
	MQX     MQXSection     `yaml:"mqx"`
	VectorX VectorXSection `yaml:"vectorx"`
}

type MQXSection struct {
	Raw yaml.Node `yaml:",inline"`
}

type VectorXSection struct {
	Trace    TracingConfig     `yaml:"trace"`
	Milvus   []MilvusConfig    `yaml:"milvus"`
	Qdrant   []QdrantConfig    `yaml:"qdrant"`
	Weaviate []WeaviateConfig  `yaml:"weaviate"`
}

// Load reads mq.yaml from path and returns parsed Config.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := c.VectorX.Trace.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}
