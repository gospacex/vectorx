// Package pgvector — Pgvector driver config with standalone/cluster modes.
package pgvector

// PgvectorStandaloneConfig single-instance configuration.
type PgvectorStandaloneConfig struct {
	DSN      string `yaml:"dsn"      json:"dsn"` // e.g. postgres://user:pass@localhost:5432/db
	Host     string `yaml:"host"     json:"host"`
	Port     int    `yaml:"port"     json:"port"`
	User     string `yaml:"user"     json:"user"`
	Password string `yaml:"password" json:"password"`
	DBName   string `yaml:"dbname"   json:"dbname"`
	Dim      int    `yaml:"dim"      json:"dim"`
	Timeout  int    `yaml:"timeout"  json:"timeout"` // seconds
}

// PgvectorClusterConfig enterprise-grade cluster configuration.
type PgvectorClusterConfig struct {
	PgvectorStandaloneConfig `yaml:",inline"`
	MaxOpenConns             int    `yaml:"max_open_conns"    json:"max_open_conns"`
	MaxIdleConns             int    `yaml:"max_idle_conns"    json:"max_idle_conns"`
	ConnMaxLifetime          int    `yaml:"conn_max_lifetime" json:"conn_max_lifetime"` // seconds
	ReadTimeout              int    `yaml:"read_timeout"      json:"read_timeout"`
	WriteTimeout             int    `yaml:"write_timeout"     json:"write_timeout"`
	MaxRetries               int    `yaml:"max_retries"       json:"max_retries"`
	RetryInterval            int    `yaml:"retry_interval"    json:"retry_interval"` // milliseconds
	SSLMode                  string `yaml:"sslmode"           json:"sslmode"`        // disable | require | verify-ca | verify-full
	SSLCertFile              string `yaml:"ssl_cert_file"     json:"ssl_cert_file"`
	SSLKeyFile               string `yaml:"ssl_key_file"      json:"ssl_key_file"`
	SSLRootCert              string `yaml:"ssl_root_cert"     json:"ssl_root_cert"`
}
