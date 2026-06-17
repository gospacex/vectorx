package weaviatex

import (
	"errors"
	"fmt"
)

// CloseAll closes every cached *Weaviatex. Weaviate's client doesn't expose
// a Close method on its connection layer (HTTP/2 long-lived connections
// are managed by the underlying http.Client), so each Weaviatex.Close is
// a no-op on the network side — it only evicts the cache entry so the
// next GetWeaviate(name) constructs a fresh client (e.g. after a config
// reload with a different APIKey). Errors are wrapped with %w to keep
// the errors.Is contract that the parent vectorx package's test
// relies on.
func CloseAll() error {
	var (
		errs     []error
		closedOK []string
	)
	clientCache.Range(func(k, v any) bool {
		w := v.(*Weaviatex)
		if err := w.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", nameOf(k), err))
			return true
		}
		closedOK = append(closedOK, nameOf(k))
		return true
	})
	for _, name := range closedOK {
		clientCache.Delete(name)
	}
	return errors.Join(errs...)
}

func nameOf(k any) string {
	s, _ := k.(string)
	return s
}
