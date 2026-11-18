package destination

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
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

	removed, err := h.removedAPIKeys(registered, remote.Config, data, diffs)
	if err != nil {
		return nil, false, err
	}
	if len(removed) > 0 {
		diffs[removedAPIKeysProperty] = differ.PropertyDiff{
			Property:    removedAPIKeysProperty,
			SourceValue: removed,
			TargetValue: map[string]any{},
		}
		secretOnly = false
	}
	return diffs, secretOnly, nil
}

// removedAPIKeysProperty names the preview entry for remote API keys the
// import drops that the local comparison cannot show.
const removedAPIKeysProperty = "Config (API keys the import removes)"

// removedAPIKeys returns the top-level keys of the remote API config that the
// import's payload leaves out and that no local config line already shows:
// keys the definition does not model, such as the legacy
// oneTrustCookieCategories block, and members the converter drops on reading,
// such as an unselected event filter list. The adopt erases them, so the dry
// run names them in the API's own spelling. Keys the API never returns, such as
// secret keys, cannot be named.
func (h *HandlerImpl) removedAPIKeys(registered *definitions.RegisteredDefinition, raw json.RawMessage, data *DestinationResource, localDiffs map[string]differ.PropertyDiff) (map[string]any, error) {
	var remote map[string]any
	if err := json.Unmarshal(raw, &remote); err != nil {
		return nil, fmt.Errorf("unmarshalling destination config: %w", err)
	}
	payload, err := registered.LocalToAPI(secret.RevealSecrets(data.Config, registered.SecretKeys()))
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
	}

	shown := map[string]bool{}
	if configDiff, ok := localDiffs["Config"]; ok {
		before, _ := configDiff.SourceValue.(map[string]any)
		after, _ := configDiff.TargetValue.(map[string]any)
		changed, _ := differ.CompareData(before, after)
		for key := range changed {
			shown[key] = true
		}
	}
	secretRoots := map[string]bool{}
	for _, key := range registered.SecretKeys() {
		root, _, _ := strings.Cut(key, ".")
		secretRoots[root] = true
	}
	localRoot := map[string]string{}
	for _, prop := range registered.Properties {
		apiRoot, _, _ := strings.Cut(prop.APIKey, ".")
		local, _, _ := strings.Cut(prop.LocalKey, ".")
		if apiRoot != "" && local != "" {
			localRoot[apiRoot] = local
		}
	}

	removed := map[string]any{}
	for key, value := range remote {
		if _, kept := payload[key]; kept {
			continue
		}
		local, modelled := localRoot[key]
		if modelled && shown[local] {
			continue
		}
		if modelled && secretRoots[local] {
			value = secret.NewUnknown()
		}
		removed[key] = value
	}
	return removed, nil
}
