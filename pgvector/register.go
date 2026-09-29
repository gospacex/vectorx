// vectorx/pgvector - 驱动注册
package pgvector

import "github.com/gospacex/vectorx"

func init() {
	vectorx.RegisterCreator("pgvector", func(cfg *vectorx.Config) (vectorx.CollectionClient, error) {
		return New(cfg)
	})
}
