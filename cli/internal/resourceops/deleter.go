package resourceops

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ErrUnmanaged is returned when a delete is attempted on a resource that has
// no external ID — unmanaged resources were never claimed by IaC and cannot be
// deleted through it.
var ErrUnmanaged = errors.New("resource is not managed (no external id); nothing to delete via IaC")

// Delete removes a managed resource from the remote backend via the provider's
// LifecycleManager. It resolves id (external-id first, then remote-id); an
// unmanaged resource (no external id) is rejected with ErrUnmanaged.
func Delete(ctx context.Context, prov provider.Provider, resourceType, id string) error {
	all, err := mergedRemote(ctx, prov, resourceType)
	if err != nil {
		return fmt.Errorf("loading remote resources for %q: %w", resourceType, err)
	}

	found := findInMap(all, id)
	if found == nil {
		return fmt.Errorf("%s %q: %w", resourceType, id, ErrResourceNotFound)
	}

	if found.ExternalID == "" {
		return fmt.Errorf("%s %q: %w", resourceType, id, ErrUnmanaged)
	}

	coll := resources.NewRemoteResources()
	coll.Set(resourceType, map[string]*resources.RemoteResource{found.ID: found})

	st, err := prov.MapRemoteToState(coll)
	if err != nil {
		return fmt.Errorf("mapping remote to state for %s %q: %w", resourceType, id, err)
	}

	urn := resources.URN(found.ExternalID, resourceType)
	sr := st.GetResource(urn)
	if sr == nil {
		return fmt.Errorf("state missing resource %s %q (urn %q)", resourceType, id, urn)
	}

	if err := prov.Delete(ctx, found.ExternalID, resourceType, sr.Data()); err != nil {
		return fmt.Errorf("deleting %s %q: %w", resourceType, id, err)
	}

	return nil
}
