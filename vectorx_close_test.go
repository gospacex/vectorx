package vectorx

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestClose_Idempotent(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := rt.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestClose_AggregatesErrors(t *testing.T) {
	rt := &Runtime{}
	rt.closers = []io.Closer{
		closerFunc(func() error { return nil }),
		closerFunc(func() error { return errors.New("boom-1") }),
		closerFunc(func() error { return errors.New("boom-2") }),
	}
	err := rt.Close()
	if err == nil {
		t.Fatal("expected joined error")
	}
	msg := err.Error()
	if !contains(msg, "boom-1") || !contains(msg, "boom-2") {
		t.Fatalf("joined error missing sub-errors: %q", msg)
	}
}

func TestClose_NoClosers_ReturnsNil(t *testing.T) {
	rt := &Runtime{}
	if err := rt.Close(); err != nil {
		t.Fatalf("Close on empty closers: %v", err)
	}
}

func TestInit_TracingEnabled_RegistersCloser(t *testing.T) {
	dir := t.TempDir()
	yaml := `
vectorx:
  trace:
    enabled: true
    service_name: closer-test
    exporter: otlp
  milvus:
    - name: primary
      address: localhost:19530
`
	yamlPath := filepath.Join(dir, "mq.yaml")
	if err := os.WriteFile(yamlPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	rt, err := Init(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(rt.closers) == 0 {
		t.Fatal("expected at least one closer after Init with tracing enabled")
	}
	// We do NOT call rt.Close() here because the OTLP gRPC batch processor
	// would block on a non-existent collector. The closer registration is
	// what we can verify deterministically; the actual Shutdown path is
	// exercised in production where a real OTLP endpoint is configured.
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// TestInit_NoAdapters_FailsFast verifies that Init returns
// ErrNoAdaptersConfigured when the YAML has the vectorx section but
// no adapter blocks — a common configuration mistake that should be
// caught at startup rather than surfaced as "nil client" later.
func TestInit_NoAdapters_FailsFast(t *testing.T) {
	dir := t.TempDir()
	yaml := `
vectorx:
  trace:
    enabled: false
    service_name: empty-test
    exporter: otlp
`
	yamlPath := filepath.Join(dir, "mq.yaml")
	if err := os.WriteFile(yamlPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	rt, err := Init(yamlPath)
	if !errors.Is(err, ErrNoAdaptersConfigured) {
		t.Fatalf("expected ErrNoAdaptersConfigured, got rt=%v err=%v", rt, err)
	}
	if rt != nil {
		t.Fatal("expected nil Runtime when adapter config is empty")
	}
}

// TestClose_ConcurrentWithAccessors_NoRace is the regression test for the
// TOCTOU race fixed by switching closed from atomic.Bool to sync.RWMutex.
// It launches 100 accessors in parallel with a Close call; the race
// detector is the authoritative check that the lock serializes all
// access to the closed field. The previous atomic.Bool design could in
// principle return a live client from GetXxx after Close returned;
// the RWMutex design holds the write-lock for the entire Close so no
// accessor can be mid-delegation when Close returns.
//
// The post-Close accessor call is a deterministic contract check that
// the closed flag was actually flipped. The mid-race ErrClosed count
// is non-deterministic (depends on whether each accessor beat Close
// to the underlying call), so we only log it — the race detector is
// the real regression test.
func TestClose_ConcurrentWithAccessors_NoRace(t *testing.T) {
	rt, err := Init(testdataPath)
	if err != nil {
		t.Fatal(err)
	}

	const N = 100
	start := make(chan struct{})
	done := make(chan struct{}, N+1)
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			<-start
			_, errs[i] = rt.Milvus("nonexistent")
			done <- struct{}{}
		}()
	}
	// Close is launched from its own goroutine so it runs truly
	// concurrently with the accessors (calling it from the test
	// goroutine would let the scheduler defer it past the accessor
	// burst on a fast machine).
	go func() {
		<-start
		for i := 0; i < 5; i++ {
			_ = rt.Close()
		}
		done <- struct{}{}
	}()
	close(start)

	for i := 0; i < N+1; i++ {
		<-done
	}

	// Diagnostic: report how many accessors observed ErrClosed.
	// We do NOT fail on 0 here — see the comment above.
	sawClosed := 0
	for _, e := range errs {
		if errors.Is(e, ErrClosed) {
			sawClosed++
		}
	}
	t.Logf("mid-race: %d/%d accessors observed ErrClosed (non-deterministic)", sawClosed, N)

	// Deterministic contract: after Close ran at least once, the
	// runtime must report ErrClosed.
	if _, err := rt.Milvus("nonexistent"); !errors.Is(err, ErrClosed) {
		t.Fatalf("post-Close accessor must return ErrClosed, got %v", err)
	}
}
