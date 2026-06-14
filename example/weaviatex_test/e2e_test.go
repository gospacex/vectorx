//go:build integration
// +build integration

package weaviatex_test

import (
	"context"
	"os"
	"testing"

	"github.com/gospacex/vectorx/weaviatex"
)

func TestWeaviateLiveE2E(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	weaviatex.SetConfigPath("mq.yaml")

	_, err := weaviatex.GetWeaviate("primary")
	if err != nil {
		t.Fatalf("GetWeaviate: %v", err)
	}

	_ = context.Background()
	t.Log("weaviate client created")
}
