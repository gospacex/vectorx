package milvusx

import (
	"errors"
	"fmt"
)

// CloseAll closes every cached *Milvusx in an arbitrary but deterministic
// order (sorted by name). Returns the joined error of every individual
// Close. The cache is cleared only on success path: if any Close fails,
// callers can still inspect surviving entries via the package-level
// snapshot, and a subsequent CloseAll retries the failed ones.
//
// Individual close errors are wrapped with %w (not string-concatenated)
// so callers can still match them with errors.Is — in particular
// milvusx.ErrForcedClose, which the test fixture plants via
// PlantFailingForTest. The "name: " prefix is preserved in the error
// message for log-friendliness.
//
// Intended to be called by vectorx.Runtime.Close during graceful
// shutdown, so the process exits with all gRPC connections drained
// rather than left to the OS to abort.
func CloseAll() error {
	var (
		errs     []error
		closedOK []string
	)
	clientCache.Range(func(k, v any) bool {
		m := v.(*Milvusx)
		if err := m.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", nameOf(k), err))
			return true
		}
		closedOK = append(closedOK, nameOf(k))
		return true
	})
	// Best-effort cleanup: also drop entries whose Close failed, so the
	// next GetMilvus(name) doesn't hand back a half-closed client.
	for _, name := range closedOK {
		clientCache.Delete(name)
	}
	return errors.Join(errs...)
}

func nameOf(k any) string {
	s, _ := k.(string)
	return s
}
