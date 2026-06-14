package qdrantx

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "mq.yaml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGetQdrant_UnknownName_ReturnsError(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	p := writeTestConfig(t, "vectorx:\n  qdrant:\n    - name: primary\n      host: localhost\n      port: 6334\n")
	SetConfigPath(p)

	_, err := GetQdrant("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent config name")
	}
}

func TestMustGetQdrant_UnknownName_Panics(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	p := writeTestConfig(t, "vectorx:\n  qdrant:\n    - name: primary\n      host: localhost\n      port: 6334\n")
	SetConfigPath(p)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustGetQdrant("nonexistent")
}

func TestGetQdrant_CacheHit_AfterLoad(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	sentinel := &Qdrantx{}
	clientCache.Store("cached", sentinel)

	c, err := GetQdrant("cached")
	if err != nil {
		t.Fatal(err)
	}
	if c != sentinel {
		t.Fatal("expected cached instance")
	}
}

func TestGetQdrant_ConcurrentAccess_RaceFree(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = GetQdrant("race-test")
		}()
	}
	wg.Wait()
}

// BenchmarkGetQdrant_CacheHit measures the per-name singleton hot path:
// once a client is constructed, every GetQdrant call is a sync.Map load +
// nil error. The address never resolves (we never call a method on the
// client), so this isolates the SDK-side overhead from any real network I/O.
func BenchmarkGetQdrant_CacheHit(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	sentinel := &Qdrantx{}
	clientCache.Store("bench", sentinel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetQdrant("bench")
	}
}

// BenchmarkGetQdrant_CacheHit_Parallel measures concurrent callers hitting
// the same name. Every goroutine takes the per-key mutex in series; the
// sync.Map Load itself is lock-free, so the benchmark characterizes the
// mutex acquisition cost when many goroutines resolve the same client.
func BenchmarkGetQdrant_CacheHit_Parallel(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	sentinel := &Qdrantx{}
	clientCache.Store("bench", sentinel)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = GetQdrant("bench")
		}
	})
}

// BenchmarkGetQdrant_MultiName_Parallel measures the per-name independence:
// distinct names take distinct per-key mutexes, so the workload parallelizes
// linearly up to GOMAXPROCS. This is the production "multi-tenant" pattern
// where each tenant has its own name in the YAML.
func BenchmarkGetQdrant_MultiName_Parallel(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	const numNames = 32
	for i := 0; i < numNames; i++ {
		clientCache.Store(nameForBench(i), &Qdrantx{})
	}

	var seq atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := seq.Add(1)
			_, _ = GetQdrant(nameForBench(int(idx) % numNames))
		}
	})
}

func nameForBench(i int) string {
	const digits = "0123456789abcdef"
	buf := []byte("bench-")
	buf = append(buf, digits[(i>>4)&0xf], digits[i&0xf])
	return string(buf)
}
