package milvusx

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

func TestGetMilvus_UnknownName_ReturnsError(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	p := writeTestConfig(t, "vectorx:\n  milvus:\n    - name: primary\n      address: localhost:19530\n")
	SetConfigPath(p)

	_, err := GetMilvus("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent config name")
	}
}

func TestMustGetMilvus_UnknownName_Panics(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	p := writeTestConfig(t, "vectorx:\n  milvus:\n    - name: primary\n      address: localhost:19530\n")
	SetConfigPath(p)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustGetMilvus("nonexistent")
}

func TestGetMilvus_CacheHit_AfterLoad(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	// With a valid config, GetMilvus should attempt connection and fail
	// (no real server). We test the cache hit logic by storing a sentinel.
	loadOnce = sync.Once{}
	configPath = ""

	// Store a sentinel with a controlled name to test cache path
	sentinel := &Milvusx{}
	clientCache.Store("cached", sentinel)

	c, err := GetMilvus("cached")
	if err != nil {
		t.Fatal(err)
	}
	if c != sentinel {
		t.Fatal("expected cached instance")
	}
}

func TestGetMilvus_ConcurrentAccess_RaceFree(t *testing.T) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	// concurrent calls with a name that is not in cache should all return
	// the same error (config not found), but not race
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = GetMilvus("race-test")
		}()
	}
	wg.Wait()
}

// BenchmarkGetMilvus_CacheHit measures the per-name singleton hot path:
// once a client is constructed, every GetMilvus call is a sync.Map load +
// nil error. Mirrors the qdrantx/weaviatex benchmarks so the three
// adapters can be compared apples-to-apples.
func BenchmarkGetMilvus_CacheHit(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	sentinel := &Milvusx{}
	clientCache.Store("bench", sentinel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetMilvus("bench")
	}
}

// BenchmarkGetMilvus_CacheHit_Parallel measures concurrent callers hitting
// the same name. Per-key mutex serializes cache hits; this characterizes
// the contention cost.
func BenchmarkGetMilvus_CacheHit_Parallel(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	sentinel := &Milvusx{}
	clientCache.Store("bench", sentinel)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = GetMilvus("bench")
		}
	})
}

// BenchmarkGetMilvus_MultiName_Parallel measures the per-name independence:
// distinct names take distinct per-key mutexes. Multi-tenant production
// pattern — scales linearly up to GOMAXPROCS.
func BenchmarkGetMilvus_MultiName_Parallel(b *testing.B) {
	clientCache = sync.Map{}
	clientLocks = sync.Map{}

	const numNames = 32
	for i := 0; i < numNames; i++ {
		clientCache.Store(milvusBenchName(i), &Milvusx{})
	}

	var seq atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := seq.Add(1)
			_, _ = GetMilvus(milvusBenchName(int(idx) % numNames))
		}
	})
}

func milvusBenchName(i int) string {
	const digits = "0123456789abcdef"
	buf := []byte("bench-")
	buf = append(buf, digits[(i>>4)&0xf], digits[i&0xf])
	return string(buf)
}
