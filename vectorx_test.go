package vectorx

import (
	"errors"
	"testing"
)

const testdataPath = "vectorx/testdata/mq.yaml"

func TestMustInit_ValidConfig(t *testing.T) {
	rt := MustInit(testdataPath)
	if rt == nil {
		t.Fatal("MustInit returned nil")
	}
	if rt.Cfg.VectorX.Trace.ServiceName != "vectorx-test" {
		t.Fatalf("ServiceName = %q", rt.Cfg.VectorX.Trace.ServiceName)
	}
	if rt.Cfg.VectorX.Trace.Enabled {
		t.Fatal("fixture has Trace.Enabled: false; expected Runtime to reflect that")
	}
}

func TestMustInit_PanicsOnMissingFile(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for missing file")
		}
		msg, ok := r.(error)
		if !ok || !contains(msg.Error(), "vectorx.MustInit:") {
			t.Fatalf("panic message: %v", r)
		}
	}()
	_ = MustInit("/nonexistent/mq.yaml")
}

func TestInit_ReturnsErrorOnMissingFile(t *testing.T) {
	rt, err := Init("/nonexistent/mq.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if rt != nil {
		t.Fatal("expected nil Runtime on error")
	}
}

func TestInit_DisabledTracing_NoOp(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	if rt == nil {
		t.Fatal("nil Runtime")
	}
	if len(rt.closers) != 0 {
		t.Fatalf("expected no closers when tracing disabled, got %d", len(rt.closers))
	}
}

// TestRuntime_Milvus_PropagatesError verifies the accessor delegates
// the underlying milvusx.GetMilvus error unchanged (no extra wrapping).
// Uses a name not in the fixture so we never reach the gRPC dial path.
func TestRuntime_Milvus_PropagatesError(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	m, err := rt.Milvus("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if m != nil {
		t.Fatalf("expected nil Milvusx, got %+v", m)
	}
}

func TestRuntime_MustMilvus_PanicsOnMissing(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = rt.MustMilvus("nonexistent")
}

// TestRuntime_Milvus_ConcurrentErrorPath verifies the accessor is
// concurrency-safe: 100 goroutines hitting the per-adapter error path
// must all return the same error and report no data race. The constructor
// is not reached because the config name is missing.
func TestRuntime_Milvus_ConcurrentErrorPath(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	const N = 100
	errs := make([]error, N)
	done := make(chan struct{})
	for i := 0; i < N; i++ {
		i := i
		go func() {
			_, errs[i] = rt.Milvus("nonexistent")
			done <- struct{}{}
		}()
	}
	for i := 0; i < N; i++ {
		<-done
	}
	first := errs[0]
	if first == nil {
		t.Fatal("expected non-nil error from first call")
	}
	for i, e := range errs {
		if e == nil || e.Error() != first.Error() {
			t.Fatalf("goroutine %d: err=%v (want %v)", i, e, first)
		}
	}
}

// TestRuntime_AccessorsAfterClose_ReturnErrClosed verifies that calling
// accessors after Close returns ErrClosed (the closed gate works).
func TestRuntime_AccessorsAfterClose_ReturnErrClosed(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = rt.Close()
	if _, err := rt.Milvus("primary"); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

// TestRuntime_QdrantAndWeaviate_Propagate verifies the qdrant and
// weaviate accessors delegate errors from their adapters unchanged.
func TestRuntime_QdrantAndWeaviate_Propagate(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Qdrant("nonexistent"); err == nil {
		t.Fatal("expected error for missing qdrant name")
	}
	if _, err := rt.Weaviate("nonexistent"); err == nil {
		t.Fatal("expected error for missing weaviate name")
	}
}

func TestRuntime_MustVariantsPanicOnMissing(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		fn   func()
	}{
		{"MustMilvus", func() { _ = rt.MustMilvus("nonexistent") }},
		{"MustQdrant", func() { _ = rt.MustQdrant("nonexistent") }},
		{"MustWeaviate", func() { _ = rt.MustWeaviate("nonexistent") }},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic")
				}
				// M1 regression: panic value must be a non-nil
				// error so callers can errors.Is / errors.As it
				// uniformly, regardless of whether the failure
				// was the closed gate (ErrClosed) or a missing
				// name (wrapped milvusx error).
				if _, ok := r.(error); !ok {
					t.Fatalf("panic value must be a non-nil error, got %T: %v", r, r)
				}
			}()
			c.fn()
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
