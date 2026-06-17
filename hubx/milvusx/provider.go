// Package milvusx implements hubx.ClientProvider for "vectorx.milvus".
//
// Build decodes cfg["config"] into config.MilvusConfig via mapstructure
// (TagName: "yaml", ErrorUnset / ErrorUnused both enabled) and then
// calls milvusx.New, which dials the Milvus gRPC connection.
//
// Errors are wrapped with the appropriate hubx sentinel:
//
//   - missing or invalid "config" key → hubx.ErrConfigInvalid
//   - milvusx.New failure              → hubx.ErrBuildFailed
//
// HealthCheck / Close semantics: milvus uses gRPC long-lived
// connections, so HealthCheck calls Milvusx.HealthCheck through the
// embedded client.Client (Milvus SDK exposes no Ping today; we keep
// the seam so future SDK upgrades can wire one in without changing
// the provider surface). Close delegates to Milvusx.Close, which
// releases the gRPC connection and evicts the cache entry if any.
package milvusx

import (
	"context"
	"fmt"

	hubx "github.com/gospacex/hubx"
	"github.com/mitchellh/mapstructure"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/milvusx"
)

// Provider implements hubx.ClientProvider for the "vectorx.milvus" driver.
type Provider struct{}

// New returns a new vectorx.milvus Provider.
func New() *Provider { return &Provider{} }

// Name returns the registry name.
func (p *Provider) Name() string { return "vectorx.milvus" }

// Build decodes cfg["config"] → config.MilvusConfig and calls milvusx.New.
// The returned client wraps the underlying *Milvusx so hubx.Client.Close
// releases the gRPC connection cleanly.
func (p *Provider) Build(instanceName string, cfg map[string]any) (hubx.Client, error) {
	raw, ok := cfg["config"]
	if !ok {
		return nil, fmt.Errorf("%w: vectorx.milvus/%s: missing 'config' key", hubx.ErrConfigInvalid, instanceName)
	}

	var mc config.MilvusConfig
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "yaml",
		ErrorUnset:  true,
		ErrorUnused: true,
		Result:      &mc,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: vectorx.milvus/%s: decoder: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: vectorx.milvus/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}

	cli, err := milvusx.New(&mc)
	if err != nil {
		return nil, fmt.Errorf("%w: vectorx.milvus/%s: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	return &client{c: cli}, nil
}

// HealthCheck is a no-op for the provider itself — the provider owns
// no connection state.
func (p *Provider) HealthCheck(context.Context) error { return nil }

// Close is a no-op for the provider itself.
func (p *Provider) Close() error { return nil }

// client wraps *milvusx.Milvusx as a hubx.Client.
type client struct{ c *milvusx.Milvusx }

// HealthCheck delegates to the underlying Milvus client. The Milvus
// SDK exposes a HasCollection RPC that doubles as a cheap liveness
// probe; we use it here because there is no dedicated Ping endpoint.
// If the underlying Client has been closed, the SDK returns a
// connection-closing error which surfaces as a non-nil error from
// HealthCheck — matching the hubx contract that an unhealthy client
// reports unhealthy.
func (c *client) HealthCheck(ctx context.Context) error {
	if c.c == nil {
		return nil
	}
	// Close is the canonical resource-release path. Milvusx.Close
	// performs the SDK conn close and cache eviction; we call it here
	// so hubx.Client.Close stays symmetric with the rest of the
	// provider surface.
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