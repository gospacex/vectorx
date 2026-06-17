package observability

// SetEnabledForTesting flips the package-level `enabled` flag without
// going through InitTracing. InitTracing is gated by sync.Once and
// would leak the configured TracerProvider into subsequent tests in
// the same binary. Tests that want to assert on spans produced by
// adapter methods (which call observability.StartSpan) need the flag
// on for the duration of the test, then off again so the noop path
// is restored.
//
// Usage:
//
//	func TestX(t *testing.T) {
//	    observability.SetEnabledForTesting(t, true)
//	    ...
//	}
//
// The t.Cleanup hook restores the previous value, so the flag is
// always reset to whatever it was before the test started — including
// the common case where it was already false.
func SetEnabledForTesting(t interface{ Cleanup(func()) }, on bool) {
	prev := enabled
	enabled = on
	t.Cleanup(func() { enabled = prev })
}
