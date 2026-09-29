// vectorx/qdrant - 驱动注册
package qdrant

import "github.com/gospacex/vectorx"

func init() {
	vectorx.RegisterCreator("qdrant", func(cfg *vectorx.Config) (vectorx.CollectionClient, error) {
		return New(cfg)
	})
}
