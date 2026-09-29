// vectorx/pinecone - 驱动注册
package pinecone

import "github.com/gospacex/vectorx"

func init() {
	vectorx.RegisterCreator("pinecone", func(cfg *vectorx.Config) (vectorx.CollectionClient, error) {
		return New(cfg)
	})
}
