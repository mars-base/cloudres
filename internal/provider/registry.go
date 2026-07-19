// Package provider implements the provider registry and cloud provider detectors.
package provider

import (
	"context"
	"fmt"

	"github.com/mars-base/cloudres/internal/core"
)

// Registry holds all known provider detectors.
type Registry struct {
	detectors []core.ProviderDetector
}

// NewRegistry creates a registry with all built-in detectors.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a detector to the registry.
func (r *Registry) Register(d core.ProviderDetector) {
	r.detectors = append(r.detectors, d)
}

// DetectAll scans for all installed cloud providers.
func (r *Registry) DetectAll(ctx context.Context) []*core.Provider {
	var providers []*core.Provider
	for _, d := range r.detectors {
		p, err := d.Detect(ctx)
		if err != nil || p == nil {
			continue
		}
		providers = append(providers, p)
	}
	return providers
}

// Get returns a specific provider by name, or an error if not found.
func (r *Registry) Get(ctx context.Context, name string) (*core.Provider, error) {
	for _, d := range r.detectors {
		if d.Name() != name {
			continue
		}
		p, err := d.Detect(ctx)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, fmt.Errorf("provider %q not detected", name)
		}
		return p, nil
	}
	return nil, fmt.Errorf("unknown provider %q", name)
}

// FetchersFor returns all resource fetchers for a named provider.
func (r *Registry) FetchersFor(name string) []core.ResourceFetcher {
	for _, d := range r.detectors {
		if d.Name() == name {
			return d.Fetchers()
		}
	}
	return nil
}

// ProviderNames returns all registered provider names.
func (r *Registry) ProviderNames() []string {
	names := make([]string, len(r.detectors))
	for i, d := range r.detectors {
		names[i] = d.Name()
	}
	return names
}

// Detector returns the detector for a named provider.
func (r *Registry) Detector(name string) core.ProviderDetector {
	for _, d := range r.detectors {
		if d.Name() == name {
			return d
		}
	}
	return nil
}
