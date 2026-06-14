// Package vectorx provides a one-line init entry point that bootstraps the
// entire vectorx SDK: config loading, observability tracing, and per-adapter
// configuration paths. The returned *Runtime exposes thin accessor methods
// (Milvus / Qdrant / Weaviate) that delegate to each adapter's existing
// lazy-singleton API, so no client is constructed until first use.
package vectorx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/milvusx"
	"github.com/gospacex/vectorx/observability"
	"github.com/gospacex/vectorx/qdrantx"
	"github.com/gospacex/vectorx/weaviatex"

	"go.opentelemetry.io/otel"
)

// ErrClosed is returned by Runtime accessor methods after Close has been
// called. Accessors check this sentinel under an RWMutex read-lock; adapter
// constructors that return errors propagate them unchanged.
var ErrClosed = errors.New("vectorx: runtime already closed")

// ErrNoAdaptersConfigured is returned by Init when the YAML has a vectorx
// section but no adapter blocks. This is almost always a configuration
// mistake (the application would have no way to use the SDK), so we
// fail fast at startup rather than returning a Runtime that does
// nothing useful.
var ErrNoAdaptersConfigured = errors.New("vectorx.Init: no adapter blocks configured (vectorx.milvus, vectorx.qdrant, vectorx.weaviate)")

// Runtime is the value object returned by Init / MustInit. It holds the
// loaded *config.Config (exported for inspection) and a list of close hooks
// (e.g. the OTel TracerProvider shutdown function) executed in LIFO order
// when Close is called.
//
// The mu/closed fields protect accessor delegation. Close acquires the
// write-lock to ensure that no accessor can still be running its
// delegated GetXxx call after Close returns; accessors acquire the
// read-lock to allow concurrent reads.
type Runtime struct {
	// Cfg is the parsed configuration. Read-only; do not mutate.
	Cfg *config.Config

	closers []io.Closer

	mu     sync.RWMutex
	closed bool
}

// Init loads mq.yaml from path, initializes the global OTel TracerProvider
// when vectorx.trace.enabled is true, and registers the config path on each
// adapter so subsequent GetXxx calls can find their config block. Each step
// wraps errors with the stage name to make debugging easier.
func Init(path string) (*Runtime, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("vectorx.Init: config.Load(%q): %w", path, err)
	}

	// Fail fast: an empty SDK config (no adapters) is almost always a
	// mistake, and returning a Runtime that does nothing useful makes
	// the bug harder to discover later.
	if len(cfg.VectorX.Milvus) == 0 &&
		len(cfg.VectorX.Qdrant) == 0 &&
		len(cfg.VectorX.Weaviate) == 0 {
		return nil, ErrNoAdaptersConfigured
	}

	rt := &Runtime{Cfg: cfg}

	if cfg.VectorX.Trace.Enabled {
		if err := observability.InitTracing(&cfg.VectorX.Trace); err != nil {
			return nil, fmt.Errorf("vectorx.Init: observability.InitTracing: %w", err)
		}
		// Register a closer that flushes + shuts down the SDK TracerProvider.
		// We type-assert the OTel global back to the SDK concrete type so we
		// do not need to modify the observability package. If the assertion
		// fails (e.g. someone replaced the provider), the runtime still
		// works; only the shutdown path is a no-op. This is best-effort by
		// design — observability owns the global and we cannot force a
		// shutdown on an unknown implementation.
		if tp, ok := otel.GetTracerProvider().(interface {
			Shutdown(ctx context.Context) error
		}); ok {
			rt.closers = append(rt.closers, tpCloser{provider: tp})
		}
	}

	milvusx.SetConfigPath(path)
	qdrantx.SetConfigPath(path)
	weaviatex.SetConfigPath(path)

	return rt, nil
}

// MustInit is the panic-on-error variant of Init. Useful at application
// startup where a missing or malformed config is fatal.
func MustInit(path string) *Runtime {
	rt, err := Init(path)
	if err != nil {
		panic(fmt.Errorf("vectorx.MustInit: %w", err))
	}
	return rt
}

// Milvus returns the cached *milvusx.Milvusx for name, constructing on first
// call. The construction happens inside milvusx (sync.Map + per-key mutex);
// the Runtime adds no caching of its own. Returns ErrClosed if Close has
// been called.
func (r *Runtime) Milvus(name string) (*milvusx.Milvusx, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return nil, ErrClosed
	}
	return milvusx.GetMilvus(name)
}

// MustMilvus panics if Milvus returns an error. The panic value is always
// a non-nil error (the closed-state path uses ErrClosed; the missing-name
// path wraps the underlying error).
func (r *Runtime) MustMilvus(name string) *milvusx.Milvusx {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		panic(ErrClosed)
	}
	c, err := milvusx.GetMilvus(name)
	if err != nil {
		panic(fmt.Errorf("vectorx.MustMilvus(%q): %w", name, err))
	}
	return c
}

// Qdrant mirrors Milvus for the qdrantx adapter.
func (r *Runtime) Qdrant(name string) (*qdrantx.Qdrantx, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return nil, ErrClosed
	}
	return qdrantx.GetQdrant(name)
}

func (r *Runtime) MustQdrant(name string) *qdrantx.Qdrantx {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		panic(ErrClosed)
	}
	c, err := qdrantx.GetQdrant(name)
	if err != nil {
		panic(fmt.Errorf("vectorx.MustQdrant(%q): %w", name, err))
	}
	return c
}

// Weaviate mirrors Milvus for the weaviatex adapter.
func (r *Runtime) Weaviate(name string) (*weaviatex.Weaviatex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return nil, ErrClosed
	}
	return weaviatex.GetWeaviate(name)
}

func (r *Runtime) MustWeaviate(name string) *weaviatex.Weaviatex {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		panic(ErrClosed)
	}
	c, err := weaviatex.GetWeaviate(name)
	if err != nil {
		panic(fmt.Errorf("vectorx.MustWeaviate(%q): %w", name, err))
	}
	return c
}

// Close flushes and shuts down the OTel TracerProvider (if observability was
// enabled) and calls any registered per-adapter close hooks. Safe to call
// multiple times: subsequent calls return nil without re-invoking closers.
//
// The write-lock is held for the entire duration of Close. This blocks all
// concurrent accessor calls until Close returns, eliminating the TOCTOU
// race where an accessor could pass the closed check and then call the
// per-adapter GetXxx after Close has run.
func (r *Runtime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	var errs []error
	for i := len(r.closers) - 1; i >= 0; i-- {
		if err := r.closers[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// tpCloser wraps an OTel SDK TracerProvider's Shutdown method as an io.Closer.
type tpCloser struct {
	provider interface{ Shutdown(context.Context) error }
}

func (t tpCloser) Close() error {
	return t.provider.Shutdown(context.Background())
}
