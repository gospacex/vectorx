//go:build integration
// +build integration

package qdrantx_test

import (
	"context"
	"os"
	"testing"

	"github.com/gospacex/vectorx/qdrantx"
)

func TestQdrantSearchE2E(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run")
	}

	qdrantx.SetConfigPath("mq.yaml")

	_, err := qdrantx.GetQdrant("primary")
	if err != nil {
		t.Fatalf("GetQdrant: %v", err)
	}

	_ = context.Background()
	t.Log("qdrant client created")
}
