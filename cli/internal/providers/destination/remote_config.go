package destination

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// remoteConfig is a remote destination's config seen two ways: the local
// config a spec is compared against, and the API keys an update must send back
// because a spec cannot carry them.
type remoteConfig struct {
	local     map[string]any
	preserved map[string]any
}

// readRemoteConfig converts a remote destination's API config for the plan.
//
// A spec cannot carry every key the remote stores. Keys the definition does not
// model never reach local config, and empty values (the webapp persists a
// cleared field rather than unsetting it) are pruned on export. Every update
// replaces the whole config, so sending only what the spec carries would erase
// them without the plan having shown it. They are kept in preserved instead,
// and dropped from local, so a spec that leaves them out matches the remote and
// an update writes them back unchanged.
//
// An empty value equal to the key's declared default stays in local, since the
// spec side has its defaults applied and carries it too. So does the empty
// member an exclusive group selects, whose emptiness is the setting.
func readRemoteConfig(registered *definitions.RegisteredDefinition, raw json.RawMessage) (*remoteConfig, error) {
	var apiConfig map[string]any
	if err := json.Unmarshal(raw, &apiConfig); err != nil {
		return nil, fmt.Errorf("unmarshalling destination config: %w", err)
	}
	if apiConfig == nil {
		apiConfig = map[string]any{}
	}

	local, err := registered.APIToLocal(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("converting API config to local: %w", err)
	}
	roundTrip, err := registered.LocalToAPI(local)
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
	}

	// Secret keys stay too: they are compared as unknown whatever their value,
	// so the plan keeps re-applying them as it always has.
	protected := append(registered.SelectedKeyRoots(apiConfig), registered.SecretKeys()...)
	defaults := registered.ConfigDefaults()
	for key, value := range local {
		if def, ok := defaults[key]; ok && sameValue(def, value) {
			protected = append(protected, key)
		}
	}
	pruneEmptyValues(local, protected...)

	preserved := unexpressedKeys(apiConfig, roundTrip)
	if len(preserved) == 0 {
		preserved = nil
	}
	return &remoteConfig{local: local, preserved: preserved}, nil
}

// unexpressedKeys returns the parts of apiConfig that a spec cannot express:
// keys the round trip through local config loses, and empty values. Maps are
// walked so a lost key nested in a modelled block, such as a connectionMode
// entry for a source type the CLI does not manage, is kept on its own.
func unexpressedKeys(apiConfig, roundTrip map[string]any) map[string]any {
	kept := map[string]any{}
	for key, value := range apiConfig {
		expressed, ok := roundTrip[key]
		if !ok {
			kept[key] = value
			continue
		}
		nested, isMap := value.(map[string]any)
		expressedMap, expressedIsMap := expressed.(map[string]any)
		if isMap && expressedIsMap && len(nested) > 0 {
			if inner := unexpressedKeys(nested, expressedMap); len(inner) > 0 {
				kept[key] = inner
			}
			continue
		}
		if isEmptyConfigValue(value) {
			kept[key] = value
		}
	}
	return kept
}

// withPreserved returns apiConfig with each preserved key it lacks added back.
// A key the spec sets wins: preserved values only fill what the spec leaves out.
func withPreserved(apiConfig, preserved map[string]any) map[string]any {
	if len(preserved) == 0 {
		return apiConfig
	}
	merged := maps.Clone(apiConfig)
	if merged == nil {
		merged = map[string]any{}
	}
	for _, key := range slices.Sorted(maps.Keys(preserved)) {
		value := preserved[key]
		current, exists := merged[key]
		if !exists {
			merged[key] = value
			continue
		}
		currentMap, currentIsMap := current.(map[string]any)
		valueMap, valueIsMap := value.(map[string]any)
		if currentIsMap && valueIsMap {
			merged[key] = withPreserved(currentMap, valueMap)
		}
	}
	return merged
}

// sameValue compares two config values the way the plan does, so a default
// decoded from a struct tag matches the same value decoded from JSON.
func sameValue(a, b any) bool {
	diffs, _ := differ.CompareData(map[string]any{"v": a}, map[string]any{"v": b})
	return len(diffs) == 0
}

// remoteLocalConfig is the remote config as the plan compares it: preserved
// keys left out and secret keys still present wrapped as unknown.
func (h *HandlerImpl) remoteLocalConfig(registered *definitions.RegisteredDefinition, raw json.RawMessage) (*remoteConfig, error) {
	rc, err := readRemoteConfig(registered, raw)
	if err != nil {
		return nil, err
	}
	rc.local = secret.WrapUnknownSecrets(rc.local, registered.SecretKeys())
	return rc, nil
}

// previewImport compares a spec with the remote destination it adopts, field by
// field as the plan compares an update, and returns what the import writes.
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
	rc, err := h.remoteLocalConfig(registered, remote.Config)
	if err != nil {
		return nil, false, err
	}

	current := &DestinationResource{
		ID:                data.ID,
		DisplayName:       remote.Name,
		Type:              registered.Type,
		Enabled:           remote.IsEnabled,
		DefinitionVersion: remote.Version,
		Config:            rc.local,
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

// requiredAbsentSecrets returns the top-level secret keys config leaves out
// that the definition's own validation requires for it, such as an HTTP bearer
// token when auth is bearerTokenAuth. Deriving them from validation keeps a
// conditional requirement conditional: S3 access keys are asked for only when
// role-based auth is off.
func requiredAbsentSecrets(registered *definitions.RegisteredDefinition, config map[string]any) []string {
	absent := map[string]bool{}
	for _, key := range registered.SecretKeys() {
		if _, present := config[key]; !present && !strings.Contains(key, ".") {
			absent[key] = true
		}
	}
	if len(absent) == 0 {
		return nil
	}

	var required []string
	for _, configErr := range registered.ValidateConfig(config) {
		key := strings.TrimPrefix(configErr.Path, "/")
		if absent[key] && !slices.Contains(required, key) {
			required = append(required, key)
		}
	}
	slices.Sort(required)
	return required
}
