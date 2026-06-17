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

// Load reads mq.yaml from path and returns parsed Config. After parsing it
// normalizes the tracing config (mqx.Validate is a no-error mutation),
// runs the per-adapter structural validation so a missing Address / Host
// surfaces at startup instead of at first RPC, and resolves ${VAR}
// placeholders in secret fields so k8s/docker-compose deployments can
// inject credentials without templating the YAML first.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	// mqx.TracingConfig.Validate is a normalization step (no error return):
	// it lowercases exporter/protocol/sampler_type, applies sensible defaults
	// for ServiceName/Endpoint/Stream/Topic, and clamps SamplerRatio. It does
	// NOT reject unknown exporters — it falls back to its own default ("jaeger").
	// vectorx's Build function in observability/exporter accepts both mqx's
	// vocabulary (jaeger/redis_stream) and vectorx's (otlp/redis) as aliases.
	c.VectorX.Trace.Validate()
	if err := c.VectorX.Validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", path, err)
	}
	if err := c.ResolveSecrets(); err != nil {
		return nil, fmt.Errorf("resolve secrets in %s: %w", path, err)
	}
	return &c, nil
}
