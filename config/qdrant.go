package config

// QdrantConfig drives the gRPC client connection to a Qdrant cluster.
//
// Security posture (defaults to insecure for backward compatibility, but
// every production deployment should set TLS: true):
//
//   - TLS == false → plaintext gRPC; Insecure credentials. Local dev only.
//   - TLS == true  → server-authenticated TLS. ServerName defaults to Host
//                    when unset. InsecureSkipVerify must be set explicitly
//                    to bypass chain verification (self-signed clusters).
//   - CAFile       → PEM bundle to add to the trust pool. Required when
//                    the server cert is signed by a private CA that is
//                    not in the system trust store.
//
// APIKey is sent as a gRPC metadata header on every RPC when non-empty.
// GRPC is kept for parity with the legacy field even though the SDK
// only speaks gRPC — leaving the door open for a future REST fallback.
type QdrantConfig struct {
	Name               string `yaml:"name"`
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	APIKey             string `yaml:"api_key"`
	GRPC               bool   `yaml:"grpc"`
	TLS                bool   `yaml:"tls"`
	ServerName         string `yaml:"server_name"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	CAFile             string `yaml:"ca_file"`
	Timeout            string `yaml:"timeout"`
}
