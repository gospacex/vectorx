// Package pinecone — Pinecone driver config (cloud-only, no local docker-compose).
package pinecone

// PineconeStandaloneConfig single-instance configuration.
type PineconeStandaloneConfig struct {
	APIKey    string `yaml:"api_key"     json:"api_key"`
	IndexName string `yaml:"index_name"  json:"index_name"`
	Namespace string `yaml:"namespace"   json:"namespace"`
	Host      string `yaml:"host"        json:"host"` // e.g. index-xxx.svc.xxx.pinecone.io
	Dim       int    `yaml:"dim"         json:"dim"`
	Timeout   int    `yaml:"timeout"     json:"timeout"` // seconds
}

// PineconeClusterConfig enterprise-grade configuration.
type PineconeClusterConfig struct {
	PineconeStandaloneConfig `yaml:",inline"`
	MaxRetries               int `yaml:"max_retries"    json:"max_retries"`
	RetryInterval            int `yaml:"retry_interval" json:"retry_interval"` // milliseconds
	ReadTimeout              int `yaml:"read_timeout"   json:"read_timeout"`
	WriteTimeout             int `yaml:"write_timeout"  json:"write_timeout"`
}
