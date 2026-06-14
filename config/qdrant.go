package config

type QdrantConfig struct {
	Name    string `yaml:"name"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	APIKey  string `yaml:"api_key"`
	GRPC    bool   `yaml:"grpc"`
	TLS     bool   `yaml:"tls"`
	Timeout string `yaml:"timeout"`
}
