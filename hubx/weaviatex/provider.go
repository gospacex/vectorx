// Package weaviatex implements hubx.ClientProvider for "vectorx.weaviate".
//
// Build decodes cfg["config"] into config.WeaviateConfig via mapstructure
// (TagName: "yaml", ErrorUnset / ErrorUnused both enabled) and then
// calls weaviatex.New, which constructs the Weaviate SDK client.
//
// Errors are wrapped with the appropriate hubx sentinel:
//
//   - missing or invalid "config" key → hubx.ErrConfigInvalid
//   - weaviatex.New failure            → hubx.ErrBuildFailed
//
// Weaviate uses HTTP/2 long-lived connections managed by the SDK's
// internal http.Client; there is no network-level Close. Weaviatex.Close
// only evicts the cache entry. HealthCheck delegates to Weaviatex.IsLive
// which performs a cheap GET against /v1/.well-known/live — the SDK's
// canonical liveness probe.
package weaviatex

import (
	"context"
	"fmt"

	hubx "github.com/gospacex/hubx"
	"github.com/mitchellh/mapstructure"

	"github.com/gospacex/vectorx/config"
	"github.com/gospacex/vectorx/weaviatex"
)

// Provider implements hubx.ClientProvider for the "vectorx.weaviate" driver.
type Provider struct{}

// New returns a new vectorx.weaviate Provider.
func New() *Provider { return &Provider{} }

// Name returns the registry name.
func (p *Provider) Name() string { return "vectorx.weaviate" }

// Build decodes cfg["config"] → config.WeaviateConfig and calls weaviatex.New.
// The returned client wraps the underlying *Weaviatex so hubx.Client.Close
// triggers cache eviction cleanly.
func (p *Provider) Build(instanceName string, cfg map[string]any) (hubx.Client, error) {
	raw, ok := cfg["config"]
	if !ok {
		return nil, fmt.Errorf("%w: vectorx.weaviate/%s: missing 'config' key", hubx.ErrConfigInvalid, instanceName)
	}

	var wc config.WeaviateConfig
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "yaml",
		ErrorUnset:  true,
		ErrorUnused: true,
		Result:      &wc,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: vectorx.weaviate/%s: decoder: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: vectorx.weaviate/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}

	cli, err := weaviatex.New(&wc)
	if err != nil {
		return nil, fmt.Errorf("%w: vectorx.weaviate/%s: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	return &client{c: cli}, nil
}

// HealthCheck is a no-op for the provider itself — the provider owns
// no connection state.
func (p *Provider) HealthCheck(context.Context) error { return nil }

// Close is a no-op for the provider itself.
func (p *Provider) Close() error { return nil }

// client wraps *weaviatex.Weaviatex as a hubx.Client.
type client struct{ c *weaviatex.Weaviatex }

// HealthCheck delegates to Weaviatex.IsLive which issues a cheap
// HTTP GET against the Weaviate liveness endpoint. Returns the SDK
// error verbatim so callers can errors.Is / errors.As against
// transport-level failures.
func (c *client) HealthCheck(ctx context.Context) error {
	if c.c == nil {
		return nil
	}
	_, err := c.c.IsLive(ctx)
	return err
}

// Close evicts the cache entry. Weaviate's SDK client does not expose
// a network-level Close (HTTP/2 long-lived connections are managed
// by the underlying http.Client), so this is a one-line eviction.
func (c *client) Close() error {
	if c.c == nil {
		return nil
	}
	return c.c.Close()
}