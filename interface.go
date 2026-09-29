// vectorx: 向量数据库统一接口，支持多驱动实现
package vectorx

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gospacex/configx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
	"gopkg.in/yaml.v3"
)

// Vector 向量
type Vector []float32

// Embedding 含向量的文档
type Embedding struct {
	ID     string
	Vector Vector
	Fields map[string]interface{}
}

// SearchResult 搜索返回结果
type SearchResult struct {
	IDs       []string
	Distances []float32
	Fields    []map[string]interface{}
}

// CollectionClient 向量集合客户端接口
type CollectionClient interface {
	// CreateCollection 创建集合（建表/索引）
	CreateCollection(ctx context.Context, name string, dim int) error
	// DropCollection 删除集合
	DropCollection(ctx context.Context, name string) error
	// Insert 插入向量
	Insert(ctx context.Context, name string, embeddings []Embedding) error
	// Search 向量相似度搜索
	Search(ctx context.Context, name string, query Vector, limit int) (*SearchResult, error)
	// Delete 删除向量
	Delete(ctx context.Context, name string, ids []string) error
	// HealthCheck 健康检查
	HealthCheck() error
	// Close 关闭连接
	Close() error
	// Driver 返回驱动名称
	Driver() string
}

// Config 向量数据库配置。
//
// tag 说明：三套 tag 齐备，分别对应 configx 的三条读取通道——
// yaml 本地文件 / json 远程配置中心 / mapstructure 本地 viper 通道。
// 多词字段（enable_trace …）缺 mapstructure tag 时经 viper 通道解码
// 会静默为零值。
type Config struct {
	Driver      string `yaml:"driver" json:"driver" mapstructure:"driver"`
	Name        string `yaml:"name" json:"name" mapstructure:"name"`
	Mode        string `yaml:"mode" json:"mode" mapstructure:"mode"` // standalone | cluster
	Address     string `yaml:"address" json:"address" mapstructure:"address"`
	Token       string `yaml:"token" json:"token" mapstructure:"token"`
	Namespace   string `yaml:"namespace" json:"namespace" mapstructure:"namespace"`          // qdrant: cluster_name; pinecone: index name; pgvector: schema
	Collection  string `yaml:"collection" json:"collection" mapstructure:"collection"`       // 默认集合名（兼容 milvus）
	Dim         int    `yaml:"dim" json:"dim" mapstructure:"dim"`                            // 向量维度
	EnableTrace bool   `yaml:"enable_trace" json:"enable_trace" mapstructure:"enable_trace"` // 统一调用链开关
}

// FillDefaults 填充默认值
func (c *Config) FillDefaults() {
	if c.Address == "" {
		c.Address = "localhost:19530"
	}
	if c.Name == "" {
		c.Name = "default"
	}
	if c.Dim == 0 {
		c.Dim = 128
	}
}

// ===== Factory =====

var creators = map[string]func(*Config) (CollectionClient, error){}

// RegisterCreator 注册向量数据库驱动创建函数
func RegisterCreator(name string, creator func(*Config) (CollectionClient, error)) {
	creators[name] = creator
}

// New 创建向量数据库客户端
func New(cfg *Config) (CollectionClient, error) {
	cfg.FillDefaults()
	// 接入 hubx 统一调用链开关：组件自身配置为主开关，
	// 全局总闸 traceflag.SetEnabled(false) 可一键熔断 vectorx。
	// 各驱动侧的 *traceflag.Tracer 惰性判定，此处收敛结果即时生效。
	cfg.EnableTrace = traceflag.Apply("vectorx", cfg.EnableTrace)

	creator, ok := creators[cfg.Driver]
	if !ok {
		return nil, fmt.Errorf("vectorx: driver %q not registered; make sure to import the driver package", cfg.Driver)
	}

	client, err := creator(cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorx: create client %q: %w", cfg.Driver, err)
	}
	return client, nil
}

// Duration 返回时间（用于健康检查）
func Duration(start time.Time) time.Duration { return time.Since(start) }

// ======================= ConfigX 集成 =======================

// VectorDBSection 对应 configx.yaml 中的 vectorx 配置节
type VectorDBSection struct {
	Stores map[string]*Config `yaml:"stores"`
}

// ReadConfig 加载统一配置句柄（所有 x 系列 SDK 共享同一配置源）
func ReadConfig(opts ...configx.ConfigOptions) error {
	_, err := configx.ReadConfig(opts...)
	return err
}

// ReadAndCreate 从 configx 读取 vectorx 配置并创建指定名称的客户端
func ReadAndCreate(name string, opts ...configx.ConfigOptions) (CollectionClient, error) {
	c, err := configx.ReadConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("vectorx: failed to read config: %w", err)
	}

	section := &VectorDBSection{}
	if err := c.Unmarshal("vectorx", section); err != nil {
		return nil, fmt.Errorf("vectorx: failed to unmarshal vectorx section: %w", err)
	}

	stores := section.Stores
	if stores == nil {
		return nil, fmt.Errorf("vectorx: no vectorx.stores configured in config")
	}

	cfg, ok := stores[name]
	if !ok {
		return nil, fmt.Errorf("vectorx: store %q not found in vectorx.stores", name)
	}

	cfg.Name = name
	client, err := New(cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorx: %w", err)
	}
	return client, nil
}

// MustReadAndCreate 类似 ReadAndCreate，失败时 panic
func MustReadAndCreate(name string, opts ...configx.ConfigOptions) CollectionClient {
	client, err := ReadAndCreate(name, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vectorx: failed to initialize %q: %v\n", name, err)
		os.Exit(1)
	}
	return client
}

// ConfigFromYAML 从 YAML 字节解析 Config 列表
func ConfigFromYAML(data []byte) ([]*Config, error) {
	var section VectorDBSection
	if err := yaml.Unmarshal(data, &section); err != nil {
		return nil, fmt.Errorf("vectorx: failed to parse yaml: %w", err)
	}
	cfgs := make([]*Config, 0, len(section.Stores))
	for name, cfg := range section.Stores {
		cfg.Name = name
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}
