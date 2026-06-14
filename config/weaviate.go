package config

type WeaviateConfig struct {
	Name   string `yaml:"name"`
	Scheme string `yaml:"scheme"`
	Host   string `yaml:"host"`
	APIKey string `yaml:"api_key"`
	Class  string `yaml:"class"`
}
