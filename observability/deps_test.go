package observability

import (
	"os/exec"
	"strings"
	"testing"
)

// TestObservability_Deps_MustNotContainAdapterPackages enforces the
// "static decoupling invariant" — observability/ must not transitively
// depend on any adapter package or any redis/kafka SDK. If a future
// refactor accidentally pulls in such a dependency, this test FAILS
// (it used to silently t.Skipf on go list errors, which defeated the
// purpose of the invariant).
func TestObservability_Deps_MustNotContainAdapterPackages(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "github.com/gospacex/vectorx/observability/...").Output()
	if err != nil {
		// Fail loudly — the invariant is not optional. If go list itself
		// can't run, the test infrastructure is broken and we should not
		// pass.
		t.Fatalf("go list -deps failed (cannot enforce invariant): %v", err)
	}
	deps := string(out)
	forbidden := []string{
		"vectorx/milvusx",
		"vectorx/qdrantx",
		"vectorx/weaviatex",
		"github.com/redis/go-redis",
		"github.com/confluentinc/confluent-kafka-go",
		"/redis",
		"/kafka",
	}
	for _, pkg := range forbidden {
		if strings.Contains(deps, pkg) {
			t.Errorf("observability must not depend on %q, but found in deps", pkg)
		}
	}
}
