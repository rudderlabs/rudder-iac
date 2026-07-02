package resourceops

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

var (
	ErrResourceNotFound = errors.New("resource not found")
	ErrVerbNotSupported = errors.New("operation not supported for resource type")
)

// ValidateType returns a helpful error listing supported types when resourceType
// is not among them. supportedTypes is the full set of registered types returned
// by the composite provider — all four verb commands (get, delete, describe,
// set-external-id) route through this so users always get the same actionable message.
func ValidateType(supportedTypes []string, resourceType string) error {
	for _, t := range supportedTypes {
		if t == resourceType {
			return nil
		}
	}
	sorted := make([]string, len(supportedTypes))
	copy(sorted, supportedTypes)
	sort.Strings(sorted)
	return fmt.Errorf("unknown resource type %q; valid types: %s",
		resourceType, strings.Join(sorted, ", "))
}

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

	res := findInMap(all, id)
	if res == nil {
		return nil, fmt.Errorf("%s %q: %w", resourceType, id, ErrResourceNotFound)
	}
	return res, nil
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
// fetches managed and (when supported) unmanaged resources, and returns the
// merged map keyed by remote ID (managed entries win on collision).
func (r *Resolver) loadAll(ctx context.Context, resourceType string) (map[string]*resources.RemoteResource, error) {
	prov, err := r.ProviderFor(resourceType)
	if err != nil {
		return nil, err
	}

	merged, err := mergedRemote(ctx, prov, resourceType)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources for %q: %w", resourceType, err)
	}

	return merged, nil
}

// findInMap returns the resource matching id (external-id first, then remote-id), or nil.
func findInMap(all map[string]*resources.RemoteResource, id string) *resources.RemoteResource {
	for _, res := range all {
		if res.ExternalID == id {
			return res
		}
	}
	if res, ok := all[id]; ok {
		return res
	}
	return nil
}
