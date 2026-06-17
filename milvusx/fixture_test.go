package milvusx

import "github.com/gospacex/vectorx/config"

// configMilvusFixture is a tiny test-only helper to build a MilvusConfig
// with just a Name (the only field CloseAll reads). Defined here so the
// close_test file stays focused on the eviction behaviour.
type configMilvusFixture struct{ Name string }

func (f *configMilvusFixture) toCfg() *config.MilvusConfig {
	return &config.MilvusConfig{Name: f.Name}
}
