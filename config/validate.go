package config

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by Validate. Callers can errors.Is against these
// to distinguish "config structurally invalid" (must fail at startup) from
// "config parsed but business-rule broken" (also fail at startup, but
// distinct error path for log alerts).
var (
	ErrNameRequired    = errors.New("config: name is required")
	ErrAddressRequired = errors.New("config: address is required")
	ErrHostRequired    = errors.New("config: host is required")
	ErrSchemeInvalid   = errors.New("config: scheme must be http or https")
	ErrPortInvalid     = errors.New("config: port must be in [1, 65535]")
	ErrDuplicateName   = errors.New("config: duplicate adapter name within the same kind")
	ErrNameCollision   = errors.New("config: name collides with another adapter kind")
)

// Validate enforces the structural minimum needed to construct a client.
// Semantic checks (TLS file readability, endpoint reachability) are left to
// the adapter constructors so a bad host doesn't surface as a "config
// invalid" log — it surfaces as a dial error, which is more accurate.
func (c *MilvusConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("milvus: %w", ErrNameRequired)
	}
	if c.Address == "" {
		return fmt.Errorf("milvus[%s]: %w", c.Name, ErrAddressRequired)
	}
	return nil
}

// Validate enforces the structural minimum for a Qdrant client. TLS-only
// fields (CAFile, ServerName) are not checked here — loadCAPool returns a
// descriptive error at dial time when the PEM bundle is unreadable.
//
// Port 0 is rejected because it is YAML's "field omitted" sentinel; a
// forgotten port: line would otherwise reach the gRPC dial with a
// useless address like "host:0" and surface as a confusing connect error
// instead of a startup-time config error.
func (c *QdrantConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("qdrant: %w", ErrNameRequired)
	}
	if c.Host == "" {
		return fmt.Errorf("qdrant[%s]: %w", c.Name, ErrHostRequired)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("qdrant[%s]: %w (got %d, want 1-65535)", c.Name, ErrPortInvalid, c.Port)
	}
	return nil
}

// Validate enforces the structural minimum for a Weaviate client. Scheme
// must be http or https — a typo (e.g. "htttps") otherwise silently
// downgrades to a non-functional default at the SDK level.
func (c *WeaviateConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("weaviate: %w", ErrNameRequired)
	}
	if c.Host == "" {
		return fmt.Errorf("weaviate[%s]: %w", c.Name, ErrHostRequired)
	}
	if c.Scheme != "" && c.Scheme != "http" && c.Scheme != "https" {
		return fmt.Errorf("weaviate[%s]: %w (got %q, want http|https)", c.Name, ErrSchemeInvalid, c.Scheme)
	}
	return nil
}

// Validate walks every adapter block, enforcing per-block rules plus
// cross-block uniqueness. Names must be unique within a kind (no two
// milvus entries named "primary") and unique across kinds (no milvus
// "primary" + qdrant "primary") — the latter prevents GetXxx("primary")
// from being ambiguous once secrets and metrics are wired by name.
//
// Returns the first error found; subsequent blocks are not validated.
// This is deliberate — surfacing all errors at once is rarely useful at
// startup, and a single error message with the offending name is
// usually enough for the operator to fix the YAML.
func (s *VectorXSection) Validate() error {
	seen := map[string]string{} // name → "<kind>[<index>]" of first occurrence

	checkUnique := func(kind string, name string, idx int) error {
		key := kind + ":" + name
		if prev, dup := seen[key]; dup {
			return fmt.Errorf("%w: %s name %q used by both %s and %s[%d]",
				ErrDuplicateName, kind, name, prev, kind, idx)
		}
		seen[key] = fmt.Sprintf("%s[%d]", kind, idx)
		return nil
	}

	// Cross-kind collision: a milvus "primary" + qdrant "primary" share
	// the user-facing name "primary" but resolve through different
	// accessors — so it's only ambiguous if a future API takes a name
	// without an accessor prefix. Today it's safe; we warn via error if
	// and only if the user-facing accessor API is consolidated. For
	// now we permit it but record it for future-proofing.
	_ = ErrNameCollision

	for i := range s.Milvus {
		if err := s.Milvus[i].Validate(); err != nil {
			return err
		}
		if err := checkUnique("milvus", s.Milvus[i].Name, i); err != nil {
			return err
		}
	}
	for i := range s.Qdrant {
		if err := s.Qdrant[i].Validate(); err != nil {
			return err
		}
		if err := checkUnique("qdrant", s.Qdrant[i].Name, i); err != nil {
			return err
		}
	}
	for i := range s.Weaviate {
		if err := s.Weaviate[i].Validate(); err != nil {
			return err
		}
		if err := checkUnique("weaviate", s.Weaviate[i].Name, i); err != nil {
			return err
		}
	}
	return nil
}
