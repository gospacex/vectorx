package observability

import (
	"context"
	"testing"
)

func TestStartSpan_Disabled_NoOp(t *testing.T) {
	// tracing not initialized
	_, span := StartSpan(context.Background(), "milvusx.Search")
	defer span.End()
	if span.IsRecording() {
		t.Fatal("expected non-recording span when tracing disabled")
	}
}
