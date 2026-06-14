package weaviatex

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

func TestGetWeaviate_UnknownName_ReturnsError(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	p := writeTestConfig(t, "vectorx:\n  weaviate:\n    - name: primary\n      scheme: http\n      host: localhost:8080\n")
	SetConfigPath(p)

	_, err := GetWeaviate("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent config name")
	}
}

func TestMustGetWeaviate_UnknownName_Panics(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	p := writeTestConfig(t, "vectorx:\n  weaviate:\n    - name: primary\n      scheme: http\n      host: localhost:8080\n")
	SetConfigPath(p)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustGetWeaviate("nonexistent")
}

func TestGetWeaviate_CacheHit_AfterLoad(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	sentinel := &Weaviatex{}
	clientCache.Store("cached", sentinel)

	c, err := GetWeaviate("cached")
	if err != nil {
		t.Fatal(err)
	}
	if c != sentinel {
		t.Fatal("expected cached instance")
	}
}

func TestGetWeaviate_ConcurrentAccess_RaceFree(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = GetWeaviate("race-test")
		}()
	}
	wg.Wait()
}

// BenchmarkGetWeaviate_CacheHit measures the per-name singleton hot path.
// Mirrors qdrantx/milvusx so the three adapters can be compared.
func BenchmarkGetWeaviate_CacheHit(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	sentinel := &Weaviatex{}
	clientCache.Store("bench", sentinel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetWeaviate("bench")
	}
}

// BenchmarkGetWeaviate_CacheHit_Parallel measures concurrent callers
// hitting the same name. Per-key mutex serializes cache hits; this
// characterizes the contention cost.
func BenchmarkGetWeaviate_CacheHit_Parallel(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	sentinel := &Weaviatex{}
	clientCache.Store("bench", sentinel)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = GetWeaviate("bench")
		}
	})
}

// BenchmarkGetWeaviate_MultiName_Parallel measures per-name independence.
// Multi-tenant production pattern — scales linearly up to GOMAXPROCS.
func BenchmarkGetWeaviate_MultiName_Parallel(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	const numNames = 32
	for i := 0; i < numNames; i++ {
		clientCache.Store(weaviateBenchName(i), &Weaviatex{})
	}

	var seq atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := seq.Add(1)
			_, _ = GetWeaviate(weaviateBenchName(int(idx) % numNames))
		}
	})
}

func weaviateBenchName(i int) string {
	const digits = "0123456789abcdef"
	buf := []byte("bench-")
	buf = append(buf, digits[(i>>4)&0xf], digits[i&0xf])
	return string(buf)
}
