package destination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
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
// the spec leaves out, connects one the spec names, and switches one to
// another. Each shows as a Transformation line; a link the spec names that is
// already the remote one shows none. The spec's reference is resolved from st
// as the apply resolves it, or, for a transformation the same plan imports,
// from adopted (URN to the remote ID its import keeps). A link to a
// transformation the same apply creates never gets here: CheckPlan refuses
// that plan.
func (h *HandlerImpl) previewImport(ctx context.Context, data *DestinationResource, remoteID string, st *state.State, adopted map[string]string) (map[string]differ.PropertyDiff, bool, error) {
	remote, err := h.client.Destinations.Get(ctx, remoteID)
	if err != nil {
		return nil, false, fmt.Errorf("getting destination: %w", err)
	}
	registered, err := h.registry.GetByAPIType(remote.Type, remote.Version)
	if err != nil {
		return nil, false, fmt.Errorf("getting destination definition: %w", err)
	}
	current, err := h.remoteResource(remote, registered, data.ID)
	if err != nil {
		return nil, false, err
	}
	wanted := *data
	wanted.Transformation = nil
	wanted.Config = previewAccountConfig(wanted.Config, st, adopted)

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
	// CompareData counts a secret only the remote holds as one to re-apply, but
	// the adopt's payload leaves it out, so it is removed: the config change is
	// then a real one, and renders as (secret, removed).
	if config, ok := diffs["Config"]; ok && config.SecretOnly && removesSecret(config.SourceValue, config.TargetValue) {
		config.SecretOnly = false
		diffs["Config"] = config
		secretOnly = false
	}

	if line, changed := transformationChange(remoteTransformationID, data.Transformation, st, adopted); changed {
		diffs["Transformation"] = line
		secretOnly = false
	}

	removed, err := h.removedAPIKeys(registered, remote.Config, data)
	if err != nil {
		return nil, false, err
	}
	// One line per removed path, named in full, so the path reads as the
	// README writes it rather than as a quoted map key.
	for path, summary := range removed {
		name := removedFieldsProperty + "." + path
		diffs[name] = differ.PropertyDiff{Property: name, SourceValue: summary}
		secretOnly = false
	}
	return diffs, secretOnly, nil
}

// removedFieldsProperty names the preview entry for remote config the import
// drops that the local comparison cannot show.
const removedFieldsProperty = "Config (API keys the import removes)"

// backendDerivedKeys are config keys the backend sets from connectionMode on
// every write, per source type: useNativeSDK and useNativeSDKToSend. The CLI
// never sends them, yet the destination update e2e snapshots still carry them
// after a CLI write (cli/tests/command_destinations_apply_test.go), so an
// entry whose source type the payload's connectionMode names is rederived, not
// removed.
var backendDerivedKeys = map[string]bool{
	"useNativeSDK":       true,
	"useNativeSDKToSend": true,
}

// collectUnderivable records the entries of a backend-derived key the backend
// cannot rederive after the adopt: those whose source type the payload sends
// no connectionMode for, such as a legacy useNativeSDK on a destination that
// never stored a mode.
func collectUnderivable(key string, value any, payload, removed map[string]any) {
	if _, sent := payload[key]; sent {
		return
	}
	modes, _ := payload["connectionMode"].(map[string]any)
	entries, ok := value.(map[string]any)
	if !ok {
		if len(modes) == 0 {
			removed[key] = summarise(value)
		}
		return
	}
	for sourceType, entry := range entries {
		if _, derived := modes[sourceType]; !derived {
			removed[key+"."+sourceType] = summarise(entry)
		}
	}
}

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
		if backendDerivedKeys[key] {
			collectUnderivable(key, value, payload, removed)
			continue
		}
		collectRemoved(key, value, payload, roundTrip, key, removed)
	}
	return removed, nil
}

// collectRemoved records under path each part of value (found at key in the
// remote parent) that payload leaves out and roundTrip, the parent as local
// config can carry it, does not hold either. A part the payload leaves out but
// the round trip holds is shown by a local line, but its own children may not
// be, so it is walked against an empty payload.
func collectRemoved(key string, value any, payload, roundTrip map[string]any, path string, removed map[string]any) {
	sent, inPayload := payload[key]
	held, inRoundTrip := roundTrip[key]
	if !inPayload && !inRoundTrip {
		removed[path] = summarise(value)
		return
	}
	collectRemovedIn(value, sent, held, path, removed)
}

// collectRemovedIn walks value against sent (nil when the payload leaves it
// out) and held, down maps by key and lists by index.
func collectRemovedIn(value, sent, held any, path string, removed map[string]any) {
	switch remote := value.(type) {
	case map[string]any:
		sentMap, ok := sent.(map[string]any)
		if !ok && sent != nil {
			return // a changed type is a value change, shown by a local line
		}
		heldMap, _ := held.(map[string]any)
		for key, item := range remote {
			collectRemoved(key, item, sentMap, heldMap, path+"."+key, removed)
		}
	case []any:
		sentList, ok := sent.([]any)
		if !ok && sent != nil {
			return
		}
		heldList, _ := held.([]any)
		for i, item := range remote {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if i >= len(heldList) && i >= len(sentList) {
				removed[itemPath] = summarise(item)
				continue
			}
			var sentItem, heldItem any
			if i < len(sentList) {
				sentItem = sentList[i]
			}
			if i < len(heldList) {
				heldItem = heldList[i]
			}
			collectRemovedIn(item, sentItem, heldItem, itemPath, removed)
		}
	}
}

// removesSecret reports whether source holds a secret target lacks, at any
// depth: a secret the remote has that the adopt's payload leaves out.
func removesSecret(source, target any) bool {
	switch typed := source.(type) {
	case secret.String, *secret.String:
		return target == nil
	case map[string]any:
		targetMap, _ := target.(map[string]any)
		for key, item := range typed {
			if removesSecret(item, targetMap[key]) {
				return true
			}
		}
	case []any:
		targetList, _ := target.([]any)
		for i, item := range typed {
			var targetItem any
			if i < len(targetList) {
				targetItem = targetList[i]
			}
			if removesSecret(item, targetItem) {
				return true
			}
		}
	}
	return false
}

// summarise describes a removed value without printing it: an empty one as it
// is, anything else by its shape.
func summarise(value any) any {
	// A null removed would read as absence, and absence is no line at all.
	if value == nil {
		return "(null)"
	}
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

// previewAccountConfig returns config with an account reference replaced by the
// remote ID it resolves to, so it compares with the literal the remote holds.
// MapRemoteToState maps a managed account to a reference, but the preview reads
// the remote through remoteResource, which does not, because the account being
// adopted has no URN to point at yet.
//
// An account already in state resolves from st; one the same plan imports keeps
// its remote ID (adopted). Neither having it means the account is created by
// this apply, and the unresolved reference is then what the comparison should
// show.
func previewAccountConfig(config map[string]any, st *state.State, adopted map[string]string) map[string]any {
	ref, ok := config[accountConfigKey].(*resources.PropertyRef)
	if !ok {
		return config
	}
	id, resolved := resolvedRefID(ref, st)
	if !resolved {
		if id, resolved = adopted[ref.URN]; !resolved {
			return config
		}
	}
	// Copied rather than mutated: config is the graph's own, and the apply
	// reads it again after the preview.
	flattened := maps.Clone(config)
	flattened[accountConfigKey] = id
	return flattened
}

// transformationChange compares the remote transformation link with the one the
// spec names, as Import's syncTransformationLink will: by remote ID.
func transformationChange(remoteID string, ref *resources.PropertyRef, st *state.State, adopted map[string]string) (differ.PropertyDiff, bool) {
	line := differ.PropertyDiff{Property: "Transformation"}
	if remoteID != "" {
		line.SourceValue = remoteID + " (remote id)"
	}
	if ref == nil {
		return line, remoteID != ""
	}

	wantedID, resolved := resolvedRefID(ref, st)
	if resolved {
		line.TargetValue = wantedID + " (" + ref.URN + ")"
		return line, wantedID != remoteID
	}
	adoptedID, isAdopted := adopted[ref.URN]
	if !isAdopted {
		// Not in state nor imported: created by this apply, which CheckPlan
		// refuses before the preview runs.
		return line, false
	}
	line.TargetValue = adoptedID + " (" + ref.URN + ", adopted in this apply)"
	return line, adoptedID != remoteID
}

// resolvedRefID resolves a copy of ref against st, leaving the plan's
// reference untouched for the apply to resolve.
func resolvedRefID(ref *resources.PropertyRef, st *state.State) (string, bool) {
	if ref.IsResolved {
		return ref.Value, ref.Value != ""
	}
	if st == nil || st.GetResource(ref.URN) == nil {
		return "", false
	}
	resolved := *ref
	if err := state.DereferenceByReflection(&resolved, st); err != nil || !resolved.IsResolved {
		return "", false
	}
	return resolved.Value, resolved.Value != ""
}
