package config

type MilvusConfig struct {
	Name       string `yaml:"name"`
	Address    string `yaml:"address"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	DBName     string `yaml:"db_name"`
	Collection string `yaml:"collection"`
	APIKey     string `yaml:"api_key"`
	TLS        bool   `yaml:"tls"`
}
