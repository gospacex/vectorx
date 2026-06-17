package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ParseVectorXBlock(t *testing.T) {
	dir := t.TempDir()
	yaml := `
vectorx:
  trace:
    enabled: true
    service_name: vectorx-test
    exporter: otlp
    endpoint: localhost:4317
  milvus:
    - name: primary
      address: localhost:19530
  qdrant:
    - name: backup
      host: localhost
      port: 6334
  weaviate:
    - name: audit
      scheme: http
      host: localhost:8080
`
	p := filepath.Join(dir, "mq.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VectorX.Trace.Exporter != "jaeger" {
		// mqx.TracingConfig.Validate normalizes "otlp" → "jaeger" because
		// "otlp" is not in mqx's allow-list. The vectorx exporter.Build
		// accepts both names; the post-Validate canonical name is "jaeger".
		t.Fatalf("trace.exporter after Validate = %q (want %q)", cfg.VectorX.Trace.Exporter, "jaeger")
	}
	if len(cfg.VectorX.Milvus) != 1 || cfg.VectorX.Milvus[0].Name != "primary" {
		t.Fatalf("milvus[0].name = %+v", cfg.VectorX.Milvus)
	}
}

