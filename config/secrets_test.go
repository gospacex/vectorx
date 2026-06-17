package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandSecrets_Empty(t *testing.T) {
	got, err := expandSecrets("")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestExpandSecrets_NoPlaceholders(t *testing.T) {
	got, err := expandSecrets("plain-value")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "plain-value" {
		t.Fatalf("got %q, want plain-value", got)
	}
}

func TestExpandSecrets_EnvSet(t *testing.T) {
	t.Setenv("VECTORX_TEST_VAR", "secret-value")
	got, err := expandSecrets("${VECTORX_TEST_VAR}")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "secret-value" {
		t.Fatalf("got %q, want secret-value", got)
	}
}

func TestExpandSecrets_EnvUnset_NoDefault_Errors(t *testing.T) {
	os.Unsetenv("VECTORX_TEST_UNSET")
	_, err := expandSecrets("${VECTORX_TEST_UNSET}")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("err = %v, want errors.Is(ErrSecretNotFound)", err)
	}
}

func TestExpandSecrets_EnvUnset_WithDefault(t *testing.T) {
	os.Unsetenv("VECTORX_TEST_UNSET")
	got, err := expandSecrets("${VECTORX_TEST_UNSET:-fallback}")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "fallback" {
		t.Fatalf("got %q, want fallback", got)
	}
}

func TestExpandSecrets_EnvSet_DefaultIgnored(t *testing.T) {
	t.Setenv("VECTORX_TEST_VAR", "real")
	got, err := expandSecrets("${VECTORX_TEST_VAR:-fallback}")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "real" {
		t.Fatalf("got %q, want real", got)
	}
}

func TestExpandSecrets_FileReference(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("s3cret-from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MILVUS_PASSWORD_FILE", secretPath)

	got, err := expandSecrets("${MILVUS_PASSWORD_FILE}")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "s3cret-from-file" {
		t.Fatalf("got %q, want s3cret-from-file (trailing newline stripped)", got)
	}
}

func TestExpandSecrets_FileReference_DefaultWhenUnset(t *testing.T) {
	os.Unsetenv("MILVUS_PASSWORD_FILE")
	got, err := expandSecrets("${MILVUS_PASSWORD_FILE:-fallback}")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "fallback" {
		t.Fatalf("got %q, want fallback", got)
	}
}

func TestExpandSecrets_FileReference_MissingFile(t *testing.T) {
	t.Setenv("MILVUS_PASSWORD_FILE", "/nonexistent/path/secret")
	_, err := expandSecrets("${MILVUS_PASSWORD_FILE}")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExpandSecrets_MultiplePlaceholders(t *testing.T) {
	t.Setenv("USER", "alice")
	t.Setenv("PASS", "hunter2")
	got, err := expandSecrets("postgres://${USER}:${PASS}@host/db")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "postgres://alice:hunter2@host/db" {
		t.Fatalf("got %q, want postgres://alice:hunter2@host/db", got)
	}
}

func TestResolveSecrets_Milvus(t *testing.T) {
	t.Setenv("MILVUS_USER", "root")
	t.Setenv("MILVUS_PASS", "milvus-pwd")
	c := &Config{
		VectorX: VectorXSection{
			Milvus: []MilvusConfig{{
				Name:     "primary",
				Address:  "host:19530",
				Username: "${MILVUS_USER}",
				Password: "${MILVUS_PASS}",
			}},
		},
	}
	if err := c.ResolveSecrets(); err != nil {
		t.Fatalf("ResolveSecrets: %v", err)
	}
	if got := c.VectorX.Milvus[0].Username; got != "root" {
		t.Fatalf("username = %q, want root", got)
	}
	if got := c.VectorX.Milvus[0].Password; got != "milvus-pwd" {
		t.Fatalf("password = %q, want milvus-pwd", got)
	}
}

func TestResolveSecrets_Weaviate(t *testing.T) {
	t.Setenv("WEAVIATE_KEY", "wv-key-123")
	c := &Config{
		VectorX: VectorXSection{
			Weaviate: []WeaviateConfig{{
				Name:   "audit",
				Host:   "host",
				Scheme: "https",
				APIKey: "${WEAVIATE_KEY}",
			}},
		},
	}
	if err := c.ResolveSecrets(); err != nil {
		t.Fatalf("ResolveSecrets: %v", err)
	}
	if got := c.VectorX.Weaviate[0].APIKey; got != "wv-key-123" {
		t.Fatalf("api_key = %q, want wv-key-123", got)
	}
}

func TestResolveSecrets_Qdrant(t *testing.T) {
	t.Setenv("QDRANT_KEY", "qd-key-456")
	c := &Config{
		VectorX: VectorXSection{
			Qdrant: []QdrantConfig{{
				Name:   "backup",
				Host:   "host",
				Port:   6334,
				APIKey: "${QDRANT_KEY}",
			}},
		},
	}
	if err := c.ResolveSecrets(); err != nil {
		t.Fatalf("ResolveSecrets: %v", err)
	}
	if got := c.VectorX.Qdrant[0].APIKey; got != "qd-key-456" {
		t.Fatalf("api_key = %q, want qd-key-456", got)
	}
}

func TestLoad_ResolvesSecretsEndToEnd(t *testing.T) {
	t.Setenv("E2E_USER", "e2e-user")
	t.Setenv("E2E_PASS", "e2e-pass")

	dir := t.TempDir()
	yaml := `
vectorx:
  trace:
    enabled: false
    service_name: secrets-e2e
  milvus:
    - name: primary
      address: localhost:19530
      username: "${E2E_USER}"
      password: "${E2E_PASS}"
`
	path := filepath.Join(dir, "mq.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.VectorX.Milvus[0].Username; got != "e2e-user" {
		t.Fatalf("username = %q, want e2e-user", got)
	}
	if got := cfg.VectorX.Milvus[0].Password; got != "e2e-pass" {
		t.Fatalf("password = %q, want e2e-pass", got)
	}
}

func TestLoad_MissingEnvWithoutDefault_FailsFast(t *testing.T) {
	os.Unsetenv("MISSING_SECRET_FOR_LOAD_TEST")

	dir := t.TempDir()
	yaml := `
vectorx:
  trace:
    enabled: false
    service_name: missing-secret
  milvus:
    - name: primary
      address: localhost:19530
      password: "${MISSING_SECRET_FOR_LOAD_TEST}"
`
	path := filepath.Join(dir, "mq.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("err = %v, want errors.Is(ErrSecretNotFound)", err)
	}
}
