package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Secret resolution is applied AFTER yaml.Unmarshal + Validate so the
// structural rules (Name required, etc.) see the placeholder text and
// catch missing values. The placeholder syntax intentionally matches
// what shells (and most config libraries) use, so the same YAML works
// in docker-compose / k8s / helm without translation.
//
// Two patterns are supported inside any string field:
//
//   ${VAR}             — replaced with os.Getenv("VAR"); empty if unset.
//                        Useful when the deployment platform guarantees
//                        the env var is always set (e.g. k8s Secret
//                        mounted via envFrom).
//
//   ${VAR:-default}    — replaced with os.Getenv("VAR") if set, else
//                        "default". Useful for dev/staging overrides.
//
// The underscore-suffixed form ${VAR_FILE} is treated as a *file path* —
// the env var is read, the file is slurped, and the result replaces
// the placeholder. This matches the docker-compose / k8s secret-mount
// pattern where the secret is on disk rather than in the environment.

var envVarPattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)(?:::?-([^}]*))?\}`)

// ErrSecretNotFound is returned when a ${VAR} placeholder (without
// default and without _FILE suffix) references an env var that is not
// set. Callers can errors.Is against it to decide whether to fail-fast
// or log-and-skip. The default behaviour in ResolveSecrets is fail-fast
// because silent-empty passwords are exactly the bug we want to surface.
var ErrSecretNotFound = errors.New("config: referenced env var is not set")

// ResolveSecrets walks every string field in the config and substitutes
// ${VAR}, ${VAR:-default}, and ${VAR_FILE} placeholders. Returns the
// first error found; subsequent fields are not touched. This matches
// Validate's "first-error-wins" policy so a single config-load reports
// a single, actionable problem.
//
// Applied automatically by Load, but exposed publicly so callers that
// build a Config programmatically (tests, hot-reload paths) can re-run
// resolution after mutating fields.
func (c *Config) ResolveSecrets() error {
	for i := range c.VectorX.Milvus {
		s, err := expandSecrets(c.VectorX.Milvus[i].Username)
		if err != nil {
			return fmt.Errorf("milvus[%s].username: %w", c.VectorX.Milvus[i].Name, err)
		}
		c.VectorX.Milvus[i].Username = s

		s, err = expandSecrets(c.VectorX.Milvus[i].Password)
		if err != nil {
			return fmt.Errorf("milvus[%s].password: %w", c.VectorX.Milvus[i].Name, err)
		}
		c.VectorX.Milvus[i].Password = s

		s, err = expandSecrets(c.VectorX.Milvus[i].APIKey)
		if err != nil {
			return fmt.Errorf("milvus[%s].api_key: %w", c.VectorX.Milvus[i].Name, err)
		}
		c.VectorX.Milvus[i].APIKey = s
	}
	for i := range c.VectorX.Qdrant {
		s, err := expandSecrets(c.VectorX.Qdrant[i].APIKey)
		if err != nil {
			return fmt.Errorf("qdrant[%s].api_key: %w", c.VectorX.Qdrant[i].Name, err)
		}
		c.VectorX.Qdrant[i].APIKey = s
	}
	for i := range c.VectorX.Weaviate {
		s, err := expandSecrets(c.VectorX.Weaviate[i].APIKey)
		if err != nil {
			return fmt.Errorf("weaviate[%s].api_key: %w", c.VectorX.Weaviate[i].Name, err)
		}
		c.VectorX.Weaviate[i].APIKey = s
	}
	return nil
}

// expandSecrets is the per-field workhorse. Empty input is left alone
// (so the YAML may legitimately set password: "" for an anonymous cluster
// without a noisy ${} error).
//
// The pattern matches ${NAME} or ${NAME:-default}. NAME ending in _FILE
// triggers the file-reference path: os.Getenv("NAME_FILE") is read as a
// path and the file contents replace the placeholder.
func expandSecrets(s string) (string, error) {
	if s == "" {
		return s, nil
	}
	if !strings.Contains(s, "${") {
		return s, nil
	}

	var expandErr error
	out := envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := envVarPattern.FindStringSubmatch(match)
		name, def := sub[1], sub[2]

		// File reference.
		if strings.HasSuffix(name, "_FILE") {
			path, ok := os.LookupEnv(name)
			if !ok {
				if def != "" {
					return def
				}
				expandErr = fmt.Errorf("%w: %s (file reference, env var unset)", ErrSecretNotFound, name)
				return match
			}
			data, err := os.ReadFile(path)
			if err != nil {
				expandErr = fmt.Errorf("read secret file %q (env %s): %w", path, name, err)
				return match
			}
			return strings.TrimRight(string(data), "\r\n")
		}

		v, ok := os.LookupEnv(name)
		if !ok {
			if def != "" {
				return def
			}
			expandErr = fmt.Errorf("%w: %s", ErrSecretNotFound, name)
			return match
		}
		return v
	})
	return out, expandErr
}
