package observability

import (
	"os/exec"
	"strings"
	"testing"
)

func TestObservability_Deps_MustNotContainAdapterPackages(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "./...")
	cmd.Dir = t.TempDir()
	// Run from module root to get proper resolution
	out, err := exec.Command("go", "list", "-deps", "github.com/gospacex/vectorx/observability/...").Output()
	if err != nil {
		t.Skipf("go list -deps failed: %v", err)
	}
	deps := string(out)
	forbidden := []string{
		"vectorx/milvusx",
		"vectorx/qdrantx",
		"vectorx/weaviatex",
		"github.com/redis/go-redis",
		"github.com/confluentinc/confluent-kafka-go",
	}
	for _, pkg := range forbidden {
		if strings.Contains(deps, pkg) {
			t.Errorf("observability must not depend on %q, but found in deps: %s", pkg, deps)
		}
	}
}
