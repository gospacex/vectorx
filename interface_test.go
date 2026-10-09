package vectorx

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// ===== Config.FillDefaults =====

func TestConfigFillDefaults(t *testing.T) {
	c := &Config{}
	c.FillDefaults()

	if c.Address != "localhost:19530" {
		t.Errorf("Address = %q, want %q", c.Address, "localhost:19530")
	}
	if c.Name != "default" {
		t.Errorf("Name = %q, want %q", c.Name, "default")
	}
	if c.Dim != 128 {
		t.Errorf("Dim = %d, want 128", c.Dim)
	}
}

func TestConfigFillDefaultsKeepsExplicitValues(t *testing.T) {
	c := &Config{Address: "milvus:19530", Name: "prod", Dim: 768}
	c.FillDefaults()

	if c.Address != "milvus:19530" || c.Name != "prod" || c.Dim != 768 {
		t.Errorf("显式配置被覆盖: %+v", *c)
	}
}

// ===== ConfigFromYAML：纯逻辑，钉住「store 名注入 Config.Name」这一契约 =====
//
// 契约要点：ConfigFromYAML 接收的是**已取出的 vectorx 段**（顶层即 stores），
// 不是带 "vectorx:" 前缀的完整配置文档 —— 后者由 configx 的 Unmarshal 负责剥离。
// 传错层级不会报错，只会静默解析出 0 个 store，属于易踩的坑，故在此钉住。

func TestConfigFromYAML(t *testing.T) {
	data := []byte(`
stores:
  main:
    driver: milvus
    address: milvus:19530
    dim: 768
    enable_trace: true
  backup:
    driver: qdrant
    address: qdrant:6334
`)
	cfgs, err := ConfigFromYAML(data)
	if err != nil {
		t.Fatalf("ConfigFromYAML: %v", err)
	}
	if len(cfgs) != 2 {
		t.Fatalf("解析出 %d 个 store, want 2", len(cfgs))
	}

	byName := map[string]*Config{}
	for _, c := range cfgs {
		byName[c.Name] = c
	}

	main, ok := byName["main"]
	if !ok {
		t.Fatalf("缺少 store %q，实际: %v", "main", byName)
	}
	// 契约：store 名必须回填到 Config.Name，否则后续按名字寻址会失效
	if main.Driver != "milvus" || main.Dim != 768 || !main.EnableTrace {
		t.Errorf("main = %+v, want driver=milvus dim=768 enable_trace=true", *main)
	}

	backup, ok := byName["backup"]
	if !ok {
		t.Fatalf("缺少 store %q", "backup")
	}
	if backup.Driver != "qdrant" {
		t.Errorf("backup.Driver = %q, want qdrant", backup.Driver)
	}
}

// TestConfigFromYAMLInvalid stores 类型不匹配（期望 map，给了字符串）时
// yaml 解码会报错，ConfigFromYAML 必须把它透传出去而不是吞掉返回空列表。
func TestConfigFromYAMLInvalid(t *testing.T) {
	if _, err := ConfigFromYAML([]byte(`stores: "not-a-map"`)); err == nil {
		t.Fatal("stores 类型错误时应返回 error，实际 nil")
	}
}

// ===== 工厂 =====

func TestNewUnregisteredDriver(t *testing.T) {
	_, err := New(&Config{Driver: "definitely-not-registered"})
	if err == nil {
		t.Fatal("未注册 driver 应返回 error，实际 nil")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("错误信息应含 %q，实际: %v", "not registered", err)
	}
}

type fakeCollectionClient struct{}

func (f *fakeCollectionClient) CreateCollection(ctx context.Context, name string, dim int) error {
	return nil
}
func (f *fakeCollectionClient) DropCollection(ctx context.Context, name string) error { return nil }
func (f *fakeCollectionClient) Insert(ctx context.Context, name string, embs []Embedding) error {
	return nil
}
func (f *fakeCollectionClient) Search(ctx context.Context, name string, q Vector, limit int) (*SearchResult, error) {
	return &SearchResult{}, nil
}
func (f *fakeCollectionClient) Delete(ctx context.Context, name string, ids []string) error {
	return nil
}
func (f *fakeCollectionClient) HealthCheck() error { return nil }
func (f *fakeCollectionClient) Close() error       { return nil }
func (f *fakeCollectionClient) Driver() string     { return "fake" }

func TestFactoryCreateAfterRegister(t *testing.T) {
	const drv = "test-factory-vectorx"
	RegisterCreator(drv, func(c *Config) (CollectionClient, error) {
		return &fakeCollectionClient{}, nil
	})

	got, err := New(&Config{Driver: drv})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got == nil {
		t.Fatal("New 返回的 CollectionClient 为 nil")
	}
}

// TestFactoryConcurrentAccess 钉住 creators map 的并发安全性。
//
// 与 coordx / lockx 同源缺陷：REVIEW-REPORT.md 第二优先级第 6 条点名 vectorx
// 要给 factory map 加锁，第八节已修复清单未覆盖。未加锁时并发注册 + 并发查找
// 会触发 "fatal error: concurrent map read and map write"（不可 recover）。
// 读写两端都走受保护 API，不直接访问裸 map。
func TestFactoryConcurrentAccess(t *testing.T) {
	const n = 32
	var wg sync.WaitGroup
	wg.Add(n * 2)

	for i := 0; i < n; i++ {
		drv := "concurrent-vectorx-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		go func(d string) {
			defer wg.Done()
			RegisterCreator(d, func(c *Config) (CollectionClient, error) {
				return &fakeCollectionClient{}, nil
			})
		}(drv)

		// 命中与否都允许（注册与查找并发，顺序不定），只要不 panic / 不 race
		go func(d string) {
			defer wg.Done()
			_, _ = New(&Config{Driver: d})
		}(drv)
	}
	wg.Wait()
}
