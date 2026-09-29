// vectorx/milvus - 驱动注册
package milvus

import "github.com/gospacex/vectorx"

func init() {
	vectorx.RegisterCreator("milvus", func(cfg *vectorx.Config) (vectorx.CollectionClient, error) {
		return New(cfg)
	})
}
