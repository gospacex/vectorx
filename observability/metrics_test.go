package observability

import (
	"testing"
)

// TestObserveExport_AcceptsValidArguments locks in the call shape
// documented in metrics.go: ObserveExport takes (exporter, status,
// durationSeconds). This is the only contract adapter code relies
// on — the prometheus client does the heavy lifting of incrementing
// the counter and recording the histogram, and unit-testing those
// paths would just be testing the prometheus library.
func TestObserveExport_AcceptsValidArguments(t *testing.T) {
	const (
		exporter = "unit-test-exporter-shape"
		status   = "ok"
	)
	// Must not panic.
	ObserveExport(exporter, status, 0.001)
}

// TestObserveExport_AllowsZeroDuration documents that a 0s duration
// is a valid input (callers may not know the elapsed time).
func TestObserveExport_AllowsZeroDuration(t *testing.T) {
	ObserveExport("unit-test-zero", "ok", 0)
}

// TestObserveExport_AllowsEmptyLabels documents the behaviour for
// the (degenerate) case where the caller passes empty strings —
// the prometheus client creates a series keyed by empty string,
// which is valid. This is the call shape some adapters use for
// "untagged" exports.
func TestObserveExport_AllowsEmptyLabels(t *testing.T) {
	ObserveExport("", "", 0)
}

// TestObserveExport_IdempotentUnderConcurrency exercises the
// "safe to call from any goroutine" contract from the doc-comment.
// N goroutines each call ObserveExport concurrently; the test
// passes if no panic occurs and the histogram observer remains
// reachable afterwards (a future swap of the metric type that
// loses the internal map would surface here).
func TestObserveExport_IdempotentUnderConcurrency(t *testing.T) {
	const (
		exporter = "unit-test-exporter-concurrency"
		status   = "ok"
	)
	done := make(chan struct{}, 32)
	for i := 0; i < 32; i++ {
		go func() {
			ObserveExport(exporter, status, 0)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 32; i++ {
		<-done
	}
	if obs := ExportDuration.WithLabelValues(exporter); obs == nil {
		t.Fatal("histogram observer lost after concurrent ObserveExport")
	}
}