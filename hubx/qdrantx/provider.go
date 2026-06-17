// Package qdrantx implements hubx.ClientProvider for "vectorx.qdrant".
//
// Build decodes cfg["config"] into config.QdrantConfig via mapstructure
// (TagName: "yaml", ErrorUnset / ErrorUnused both enabled) and then
// calls qdrantx.New, which dials the Qdrant gRPC connection.
//
// Errors are wrapped with the appropriate hubx sentinel:
//
//   - missing or invalid "config" key → hubx.ErrConfigInvalid
//   - qdrantx.New failure              → hubx.ErrBuildFailed
//
// Qdrant uses gRPC long-lived connections, so Close delegates to
// Qdrantx.Close which releases the *grpc.ClientConn and evicts the
// cache entry. HealthCheck is a no-op because the Qdrant SDK has no
// dedicated Ping endpoint; production callers should observe health
// through the application-level RPCs.
package qdrantx

import (
	"context"
	"fmt"

	hubx "github.com/gospacex/hubx"
	"github.com/mitchellh/mapstructure"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/qdrantx"
)

// Provider implements hubx.ClientProvider for the "vectorx.qdrant" driver.
type Provider struct{}

// New returns a new vectorx.qdrant Provider.
func New() *Provider { return &Provider{} }

// Name returns the registry name.
func (p *Provider) Name() string { return "vectorx.qdrant" }

// Build decodes cfg["config"] → config.QdrantConfig and calls qdrantx.New.
// The returned client wraps the underlying *Qdrantx so hubx.Client.Close
// releases the gRPC connection cleanly.
func (p *Provider) Build(instanceName string, cfg map[string]any) (hubx.Client, error) {
	raw, ok := cfg["config"]
	if !ok {
		return nil, fmt.Errorf("%w: vectorx.qdrant/%s: missing 'config' key", hubx.ErrConfigInvalid, instanceName)
	}

	var qc config.QdrantConfig
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "yaml",
		ErrorUnset:  true,
		ErrorUnused: true,
		Result:      &qc,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: vectorx.qdrant/%s: decoder: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: vectorx.qdrant/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}

	cli, err := qdrantx.New(&qc)
	if err != nil {
		return nil, fmt.Errorf("%w: vectorx.qdrant/%s: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	return &client{c: cli}, nil
}

// HealthCheck is a no-op for the provider itself — the provider owns
// no connection state.
func (p *Provider) HealthCheck(context.Context) error { return nil }

// Close is a no-op for the provider itself.
func (p *Provider) Close() error { return nil }

// client wraps *qdrantx.Qdrantx as a hubx.Client.
type client struct{ c *qdrantx.Qdrantx }

// HealthCheck is a no-op — the Qdrant SDK exposes no Ping endpoint.
// The application-level RPCs (Get/Count/Search) double as live
// signals; if the underlying conn has been closed, the next RPC
// returns "grpc: the client connection is closing".
func (c *client) HealthCheck(ctx context.Context) error {
	_ = ctx
	return nil
}

// Close releases the gRPC connection and evicts the cache entry.
func (c *client) Close() error {
	if c.c == nil {
		return nil
	}
	return c.c.Close()
}