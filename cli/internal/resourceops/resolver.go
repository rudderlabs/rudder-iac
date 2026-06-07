package resourceops

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

var (
	ErrResourceNotFound = errors.New("resource not found")
	ErrVerbNotSupported = errors.New("operation not supported for resource type")
)

// Resolver routes resource operations to the appropriate provider and provides
// common lookup and capability-assertion helpers used by verb commands.
type Resolver struct{ router provider.TypeRouter }

func New(router provider.TypeRouter) *Resolver { return &Resolver{router: router} }

// ProviderFor returns the provider responsible for the given resource type.
func (r *Resolver) ProviderFor(resourceType string) (provider.Provider, error) {
	return r.router.ProviderForType(resourceType)
}

// FindRemote loads remote resources of resourceType and returns the one matching
// id (external-id first, then remote-id).
func (r *Resolver) FindRemote(ctx context.Context, resourceType, id string) (*resources.RemoteResource, error) {
	all, err := r.loadAll(ctx, resourceType)
	if err != nil {
		return nil, err
	}

	for _, res := range all {
		if res.ExternalID == id {
			return res, nil
		}
	}

	if res, ok := all[id]; ok {
		return res, nil
	}

	return nil, fmt.Errorf("%s %q: %w", resourceType, id, ErrResourceNotFound)
}

// ExternalIDSetterFor asserts the optional ExternalIDSetter capability on the provider
// for resourceType, or returns ErrVerbNotSupported when the provider doesn't implement it.
func (r *Resolver) ExternalIDSetterFor(resourceType string) (provider.ExternalIDSetter, error) {
	p, err := r.ProviderFor(resourceType)
	if err != nil {
		return nil, err
	}

	setter, ok := p.(provider.ExternalIDSetter)
	if !ok {
		return nil, fmt.Errorf("set-external-id on %q: %w", resourceType, ErrVerbNotSupported)
	}

	return setter, nil
}

// loadAll is the single load path for remote resources: resolves the provider,
// fetches managed resources, and returns the type-keyed map (keyed by remote ID).
// Task 5 will extend this with the unmanaged merge.
func (r *Resolver) loadAll(ctx context.Context, resourceType string) (map[string]*resources.RemoteResource, error) {
	prov, err := r.ProviderFor(resourceType)
	if err != nil {
		return nil, err
	}

	collection, err := prov.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources for %q: %w", resourceType, err)
	}

	return collection.GetAll(resourceType), nil
}
