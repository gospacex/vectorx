package vectorx

import (
	"os/exec"
	"strings"
	"testing"
)

func TestTopLevel_AdaptersDoNotImportTopLevel(t *testing.T) {
	for _, pkg := range []string{"./milvusx/...", "./qdrantx/...", "./weaviatex/..."} {
		out, err := exec.Command("go", "list", "-deps", pkg).Output()
		if err != nil {
			t.Fatalf("go list -deps %s: %v", pkg, err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			if line == "github.com/gospacex/vectorx" {
				t.Fatalf("%s must not depend on top-level vectorx package; found: %s", pkg, line)
			}
		}
	}
}

// TestTopLevel_NoRedisOrKafkaSDK uses substring matching to catch any
// redis or kafka client SDK, including newer ones not on the original
// allow-list (twmb/franz-go, redis/rueidis, etc.).
func TestTopLevel_NoRedisOrKafkaSDK(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps .: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "/redis") || strings.Contains(lower, "/kafka") {
			t.Fatalf("top-level vectorx must not import %s", line)
		}
	}
}
