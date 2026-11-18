package destination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
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
// Import also reconciles the transformation link: it disconnects a remote link
// the spec leaves out and connects one the spec names. Those show as a
// Transformation line. When both sides name one, the spec's URN cannot be
// matched to the remote ID before the apply resolves it, so no line is shown.
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

	link, err := h.client.Destinations.GetTransformation(ctx, remoteID)
	if err != nil && !errors.Is(err, client.ErrResourceNotFound) {
		return nil, false, fmt.Errorf("getting destination transformation: %w", err)
	}
	var remoteTransformationID string
	if link != nil {
		remoteTransformationID = link.TransformationID
	}

	var source, target map[string]any
	if err := mapstructure.Decode(current, &source); err != nil {
		return nil, false, fmt.Errorf("decoding remote destination: %w", err)
	}
	if err := mapstructure.Decode(&wanted, &target); err != nil {
		return nil, false, fmt.Errorf("decoding destination spec: %w", err)
	}
	diffs, secretOnly := differ.CompareData(source, target)

	switch {
	case remoteTransformationID != "" && data.Transformation == nil:
		diffs["Transformation"] = differ.PropertyDiff{Property: "Transformation", SourceValue: remoteTransformationID + " (remote id)"}
		secretOnly = false
	case remoteTransformationID == "" && data.Transformation != nil:
		diffs["Transformation"] = differ.PropertyDiff{Property: "Transformation", TargetValue: data.Transformation}
		secretOnly = false
	}

	removed, err := h.removedAPIKeys(registered, remote.Config, data)
	if err != nil {
		return nil, false, err
	}
	if len(removed) > 0 {
		diffs[removedFieldsProperty] = differ.PropertyDiff{
			Property:    removedFieldsProperty,
			SourceValue: removed,
			TargetValue: map[string]any{},
		}
		secretOnly = false
	}
	return diffs, secretOnly, nil
}

// removedFieldsProperty names the preview entry for remote config the import
// drops that the local comparison cannot show.
const removedFieldsProperty = "Config (API keys the import removes)"

// removedAPIKeys returns the parts of the remote API config that the import's
// payload leaves out and that local config cannot hold, keyed by their path in
// the API's spelling (consentManagement.amp, headers[0].description). Local
// config holds what survives an API -> local -> API round trip, and a removal
// there already shows as a local line; anything else would vanish unseen: keys
// the definition does not model (the legacy oneTrustCookieCategories block, a
// consent entry for a source type the definition omits), fields of an array
// element it does not map, and an unselected event filter list. Maps and array
// elements are compared all the way down.
//
// Values are summarised rather than printed, since an unmodelled key can hold
// a credential the definition does not know is one. Keys the API never
// returns, such as secret keys, cannot be named.
func (h *HandlerImpl) removedAPIKeys(registered *definitions.RegisteredDefinition, raw json.RawMessage, data *DestinationResource) (map[string]any, error) {
	var remote map[string]any
	if err := json.Unmarshal(raw, &remote); err != nil {
		return nil, fmt.Errorf("unmarshalling destination config: %w", err)
	}
	payload, err := registered.LocalToAPI(secret.RevealSecrets(data.Config, registered.SecretKeys()))
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
	}
	local, err := registered.APIToLocal(remote)
	if err != nil {
		return nil, fmt.Errorf("converting API config to local: %w", err)
	}
	roundTrip, err := registered.LocalToAPI(local)
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
	}

	// A discriminator naming no member says the same as its absence, so the
	// backend keeping or defaulting it is no change worth a line.
	neutral := map[string]bool{}
	for _, prop := range registered.Properties {
		if prop.Selector == nil {
			continue
		}
		if selected, present := prop.Selector.LocalKeyFor(remote); present && selected == "" {
			neutral[prop.Selector.APIKey] = true
		}
	}

	removed := map[string]any{}
	for key, value := range remote {
		if neutral[key] {
			continue
		}
		collectRemoved(key, value, payload, roundTrip, key, removed)
	}
	return removed, nil
}

// collectRemoved records under path each part of value (found at key in the
// remote parent) that payload leaves out and roundTrip, the parent as local
// config can carry it, does not hold either.
func collectRemoved(key string, value any, payload, roundTrip map[string]any, path string, removed map[string]any) {
	sent, inPayload := payload[key]
	held, inRoundTrip := roundTrip[key]
	if !inPayload {
		if !inRoundTrip {
			removed[path] = summarise(value)
		}
		return
	}
	collectRemovedIn(value, sent, held, path, removed)
}

func collectRemovedIn(value, sent, held any, path string, removed map[string]any) {
	switch remote := value.(type) {
	case map[string]any:
		sentMap, ok := sent.(map[string]any)
		if !ok {
			return
		}
		heldMap, _ := held.(map[string]any)
		for key, item := range remote {
			collectRemoved(key, item, sentMap, heldMap, path+"."+key, removed)
		}
	case []any:
		sentList, ok := sent.([]any)
		if !ok {
			return
		}
		heldList, _ := held.([]any)
		for i, item := range remote {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if i >= len(sentList) {
				if i >= len(heldList) {
					removed[itemPath] = summarise(item)
				}
				continue
			}
			var heldItem any
			if i < len(heldList) {
				heldItem = heldList[i]
			}
			collectRemovedIn(item, sentList[i], heldItem, itemPath, removed)
		}
	}
}

// summarise describes a removed value without printing it: an empty one as it
// is, anything else by its shape.
func summarise(value any) any {
	if isEmptyConfigValue(value) {
		return value
	}
	switch typed := value.(type) {
	case map[string]any:
		return fmt.Sprintf("(object, %d %s)", len(typed), plural(len(typed), "key"))
	case []any:
		return fmt.Sprintf("(list, %d %s)", len(typed), plural(len(typed), "item"))
	default:
		return "(value)"
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
