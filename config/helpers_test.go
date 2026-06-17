package config

import "os"

// writeFile is a small helper used by validate_test.go; co-located here
// so test files don't need to import "os" individually. Wrapping is the
// same as os.WriteFile but takes a string path for shorter test bodies.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
