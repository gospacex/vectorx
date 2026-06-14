//go:build integration
// +build integration

package milvusx_test

import (
	"context"
	"os"
	"testing"

	"github.com/gospacex/vectorx/milvusx"
)

func TestMilvusSearchE2E(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	milvusx.SetConfigPath("mq.yaml")

	c, err := milvusx.GetMilvus("primary")
	if err != nil {
		t.Fatalf("GetMilvus: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	has, err := c.HasCollection(ctx, "vectorx_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("has collection vectorx_test: %v", has)
}
