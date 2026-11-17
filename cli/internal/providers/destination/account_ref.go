package destination

import (
	"fmt"
	"maps"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// accountConfigKey is the local config key holding the integration account a
// destination authenticates with. Three destination types declare it today
// (google_adwords_offline_conversions, linkedin_ads,
// bingads_offline_conversions), all under this one name, so the reference
// support keys on the config key itself rather than on a per-definition flag:
// a destination without the key is never touched by any of this.
const accountConfigKey = "rudder_account_id"

// accountRefPrefix is the scalar reference form, matching the "#<kind>:<id>"
// shape RETL connection specs already use for warehouse accounts.
var accountRefPrefix = "#" + accounts.AccountSpecKind + ":"

// resolveAccountIDFromState reads the remote account id out of account state.
func resolveAccountIDFromState(state *accounts.AccountState) (string, error) {
	if state.ID == "" {
		return "", fmt.Errorf("account state has empty ID")
	}
	return state.ID, nil
}

// createAccountRef stamps the "id" property so the differ's ref comparison sees
// the same shape from a spec and from remote state. Mirrors
// createTransformationRef.
func createAccountRef(urn string) *resources.PropertyRef {
	ref := handler.CreatePropertyRef(urn, resolveAccountIDFromState)
	ref.Property = "id"
	return ref
}

// parseAccountRef turns a scalar "#account:<id>" value into a PropertyRef. A
// value without the "#" prefix is a literal remote account id and is left
// alone, which is what keeps every spec written before references existed
// working unchanged.
func parseAccountRef(value string) (*resources.PropertyRef, bool, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "#") {
		return nil, false, nil
	}
	id, found := strings.CutPrefix(trimmed, accountRefPrefix)
	if !found || id == "" {
		return nil, false, fmt.Errorf(
			"invalid account reference %q in '%s': expected format %s<id>",
			value, accountConfigKey, accountRefPrefix)
	}
	return createAccountRef(resources.URN(id, accounts.AccountResourceType)), true, nil
}

// withAccountRef replaces a "#account:<id>" value in the config with a
// PropertyRef the apply framework resolves. It mutates config in place and
// reports whether it replaced anything.
//
// The resolution happens inside a map value, which the reflection walker only
// reaches through its interface case — see state.DereferenceByReflection.
func withAccountRef(config map[string]any) error {
	raw, ok := config[accountConfigKey].(string)
	if !ok {
		return nil
	}
	ref, isRef, err := parseAccountRef(raw)
	if err != nil {
		return err
	}
	if isRef {
		config[accountConfigKey] = ref
	}
	return nil
}

// remoteAccountRef turns the remote destination's literal account id back into
// the same PropertyRef a spec produces, so a managed account compares equal on
// both sides instead of diffing a ref against an id forever.
//
// An account the CLI does not manage keeps its literal id on both sides, which
// is equally drift-free and is the only option: there is no URN to point at.
func remoteAccountRef(config map[string]any, urnResolver handler.URNResolver) error {
	remoteID, ok := config[accountConfigKey].(string)
	if !ok || remoteID == "" {
		return nil
	}
	urn, err := urnResolver.GetURNByID(accounts.AccountResourceType, remoteID)
	if err != nil {
		if err == resources.ErrRemoteResourceExternalIdNotFound {
			return nil
		}
		return fmt.Errorf("resolving account URN for %q: %w", remoteID, err)
	}
	config[accountConfigKey] = createAccountRef(urn)
	return nil
}

// flattenAccountRef puts the resolved remote id back into the config as a plain
// string, for the API payload. Called at the API boundary, after the framework
// has resolved the ref.
func flattenAccountRef(config map[string]any) (map[string]any, error) {
	ref, ok := config[accountConfigKey].(*resources.PropertyRef)
	if !ok {
		return config, nil
	}
	if !ref.IsResolved {
		return nil, fmt.Errorf(
			"account reference %q in '%s' was not resolved: the account it names is not in this project's state",
			ref.URN, accountConfigKey)
	}
	// Copied rather than mutated: the caller's map is the graph's own config,
	// and the apply cycle reads it again after this.
	flattened := make(map[string]any, len(config))
	maps.Copy(flattened, config)
	flattened[accountConfigKey] = ref.Value
	return flattened, nil
}

// accountRefID reports the account id a spec's config references, for the
// semantic rule. It reads the raw spec value, so it runs before extraction has
// turned the value into a PropertyRef. A malformed reference is the syntactic
// rule's to report, so it is not one here.
func accountRefID(config map[string]any) (string, bool) {
	raw, ok := config[accountConfigKey].(string)
	if !ok {
		return "", false
	}
	id, found := strings.CutPrefix(strings.TrimSpace(raw), accountRefPrefix)
	if !found || id == "" {
		return "", false
	}
	return id, true
}
