package qdrantx

import (
	"errors"
	"fmt"
)

// CloseAll closes every cached *Qdrantx and evicts the cache entries on
// success. See milvusx.CloseAll for the shutdown-order rationale. Errors
// are wrapped with %w so errors.Is(closeErr, someSentinel) keeps working
// for caller-defined sentinels — important for the test fixture that
// exercises the join contract.
func CloseAll() error {
	var (
		errs     []error
		closedOK []string
	)
	clientCache.Range(func(k, v any) bool {
		q := v.(*Qdrantx)
		if err := q.Close(); err != nil {
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
