package config

import (
	"errors"
	"strings"
	"testing"
)

func TestMilvusConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     MilvusConfig
		wantErr error // nil → expect success
	}{
		{"valid", MilvusConfig{Name: "primary", Address: "localhost:19530"}, nil},
		{"missing name", MilvusConfig{Address: "localhost:19530"}, ErrNameRequired},
		{"missing address", MilvusConfig{Name: "primary"}, ErrAddressRequired},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want errors.Is(%v)", err, tc.wantErr)
			}
		})
	}
}

func TestQdrantConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     QdrantConfig
		wantErr error
	}{
		{"valid", QdrantConfig{Name: "primary", Host: "localhost", Port: 6334}, nil},
		{"valid TLS", QdrantConfig{Name: "primary", Host: "q.example.com", Port: 6334, TLS: true}, nil},
		{"missing name", QdrantConfig{Host: "localhost", Port: 6334}, ErrNameRequired},
		{"missing host", QdrantConfig{Name: "primary", Port: 6334}, ErrHostRequired},
		{"port 0", QdrantConfig{Name: "primary", Host: "localhost", Port: 0}, ErrPortInvalid},
		{"port too high", QdrantConfig{Name: "primary", Host: "localhost", Port: 70000}, ErrPortInvalid},
		{"port negative", QdrantConfig{Name: "primary", Host: "localhost", Port: -1}, ErrPortInvalid},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want errors.Is(%v)", err, tc.wantErr)
			}
		})
	}
}

func TestWeaviateConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     WeaviateConfig
		wantErr error
	}{
		{"valid http", WeaviateConfig{Name: "primary", Host: "localhost", Scheme: "http"}, nil},
		{"valid https", WeaviateConfig{Name: "primary", Host: "localhost", Scheme: "https"}, nil},
		{"valid empty scheme", WeaviateConfig{Name: "primary", Host: "localhost"}, nil}, // SDK defaults
		{"missing name", WeaviateConfig{Host: "localhost", Scheme: "http"}, ErrNameRequired},
		{"missing host", WeaviateConfig{Name: "primary", Scheme: "http"}, ErrHostRequired},
		{"bad scheme", WeaviateConfig{Name: "primary", Host: "localhost", Scheme: "ftp"}, ErrSchemeInvalid},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want errors.Is(%v)", err, tc.wantErr)
			}
		})
	}
}

func TestVectorXSection_Validate_DuplicateNameInSameKind(t *testing.T) {
	s := &VectorXSection{
		Milvus: []MilvusConfig{
			{Name: "primary", Address: "a:19530"},
			{Name: "primary", Address: "b:19530"},
		},
	}
	err := s.Validate()
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("err = %v, want errors.Is(ErrDuplicateName)", err)
	}
	if !strings.Contains(err.Error(), "milvus") {
		t.Fatalf("err = %v, want it to mention milvus", err)
	}
}

func TestVectorXSection_Validate_AllKindsValid(t *testing.T) {
	s := &VectorXSection{
		Milvus:   []MilvusConfig{{Name: "primary", Address: "a:19530"}},
		Qdrant:   []QdrantConfig{{Name: "primary", Host: "b", Port: 6334}},
		Weaviate: []WeaviateConfig{{Name: "primary", Host: "c", Scheme: "http"}},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVectorXSection_Validate_EmptyIsAllowed(t *testing.T) {
	// The "no adapters" case is rejected one level up by
	// vectorx.ErrNoAdaptersConfigured; here we just check the section
	// itself is structurally happy.
	s := &VectorXSection{}
	if err := s.Validate(); err != nil {
		t.Fatalf("unexpected error on empty section: %v", err)
	}
}

func TestLoad_ValidatesOnLoad(t *testing.T) {
	dir := t.TempDir()
	yaml := `
vectorx:
  trace:
    enabled: false
    service_name: validate-on-load-test
  milvus:
    - name: primary
`
	path := dir + "/mq.yaml"
	if err := writeFile(path, yaml); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for milvus missing address")
	}
	if !errors.Is(err, ErrAddressRequired) {
		t.Fatalf("err = %v, want errors.Is(ErrAddressRequired)", err)
	}
}
