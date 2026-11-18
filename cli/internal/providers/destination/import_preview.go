package destination

import (
	"context"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// previewImport compares a spec with the remote destination it adopts, field by
// field as the plan compares an update, and returns what the import writes.
// Import pushes the whole spec config over the remote one, so the remote side
// is read exactly as MapRemoteToState reads a managed destination: a value the
// spec leaves out, empty or not, shows as removed.
//
// The transformation link is left out: the spec names it by URN and the remote
// by ID, so the two cannot be compared before the apply resolves it.
func (h *HandlerImpl) previewImport(ctx context.Context, data *DestinationResource, remoteID string) (map[string]differ.PropertyDiff, bool, error) {
	remote, err := h.client.Destinations.Get(ctx, remoteID)
	if err != nil {
		return nil, false, fmt.Errorf("getting destination: %w", err)
	}
	registered, err := h.registry.GetByAPIType(remote.Type, remote.Version)
	if err != nil {
		return nil, false, fmt.Errorf("getting destination definition: %w", err)
	}
	localConfig, err := h.apiConfigToLocal(registered.Type, remote.Version, remote.Config)
	if err != nil {
		return nil, false, err
	}

	current := &DestinationResource{
		ID:                data.ID,
		DisplayName:       remote.Name,
		Type:              registered.Type,
		Enabled:           remote.IsEnabled,
		DefinitionVersion: remote.Version,
		Config:            secret.WrapUnknownSecrets(localConfig, registered.SecretKeys()),
	}
	wanted := *data
	wanted.Transformation = nil

	var source, target map[string]any
	if err := mapstructure.Decode(current, &source); err != nil {
		return nil, false, fmt.Errorf("decoding remote destination: %w", err)
	}
	if err := mapstructure.Decode(&wanted, &target); err != nil {
		return nil, false, fmt.Errorf("decoding destination spec: %w", err)
	}
	diffs, secretOnly := differ.CompareData(source, target)
	return diffs, secretOnly, nil
}
