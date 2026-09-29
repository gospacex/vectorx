// Package qdrant — Qdrant driver config with standalone/cluster modes.
package qdrant

// QdrantStandaloneConfig single-instance configuration.
type QdrantStandaloneConfig struct {
	Address    string `yaml:"address" json:"address"` // e.g. localhost:6333
	Token      string `yaml:"token"   json:"token"`
	Dim        int    `yaml:"dim"     json:"dim"`
	Timeout    int    `yaml:"timeout" json:"timeout"` // seconds
	TLSEnabled bool   `yaml:"tls_enabled" json:"tls_enabled"`
}

// QdrantClusterConfig enterprise-grade cluster configuration.
type QdrantClusterConfig struct {
	QdrantStandaloneConfig `yaml:",inline"`
	MaxOpenConns           int    `yaml:"max_open_conns"    json:"max_open_conns"`
	MaxIdleConns           int    `yaml:"max_idle_conns"    json:"max_idle_conns"`
	ConnMaxLifetime        int    `yaml:"conn_max_lifetime" json:"conn_max_lifetime"` // seconds
	ReadTimeout            int    `yaml:"read_timeout"      json:"read_timeout"`
	WriteTimeout           int    `yaml:"write_timeout"     json:"write_timeout"`
	MaxRetries             int    `yaml:"max_retries"       json:"max_retries"`
	RetryInterval          int    `yaml:"retry_interval"    json:"retry_interval"` // milliseconds
	TLSCertFile            string `yaml:"tls_cert_file"     json:"tls_cert_file"`
	TLSKeyFile             string `yaml:"tls_key_file"      json:"tls_key_file"`
	TLSCAFile              string `yaml:"tls_ca_file"       json:"tls_ca_file"`
	PreferredNode          string `yaml:"preferred_node"    json:"preferred_node"` // cluster preferred node
}
