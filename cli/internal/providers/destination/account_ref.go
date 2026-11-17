package destination

import (
	"fmt"
	"maps"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
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
// PropertyRef the apply framework resolves. It mutates config in place.
//
// The resolution happens inside a map value, which the reflection walker only
// reaches through its interface case — see state.DereferenceByReflection.
//
// A reference on a type that cannot support one is refused rather than
// accepted: nothing would ever resolve it, and remote state is mapped as a
// literal for such a type, so accepting it would mean a diff on every apply.
func withAccountRef(destinationType string, config map[string]any) error {
	raw, ok := config[accountConfigKey].(string)
	if !ok {
		return nil
	}
	ref, isRef, err := parseAccountRef(raw)
	if err != nil {
		return err
	}
	if !isRef {
		return nil
	}
	if !accounts.SupportsReference(destinationType) {
		return fmt.Errorf(
			"'%s' for destination type %q takes the account's ID, not a reference: "+
				"rudder-cli cannot adopt an account for this type, so there is no account spec to point at",
			accountConfigKey, destinationType)
	}
	config[accountConfigKey] = ref
	return nil
}

// remoteAccountRef turns the remote destination's literal account id back into
// the same PropertyRef a spec produces, so a managed account compares equal on
// both sides instead of diffing a ref against an id forever.
//
// Gated on the type's capability, not on what the local spec happens to say:
// remote state is mapped without sight of the spec, so the form has to be
// decided the same way on both sides. For a type that supports references,
// import writes a reference too (toExportSpecMap), so the two agree. A literal
// left in a spec for such a type is caught at plan time by
// checkAdoptedAccountLiterals rather than diffing silently.
//
// An account the CLI does not manage keeps its literal id on both sides, which
// is equally drift-free and is the only option: there is no URN to point at.
func remoteAccountRef(destinationType string, config map[string]any, urnResolver handler.URNResolver) error {
	remoteID, ok := config[accountConfigKey].(string)
	if !ok || remoteID == "" || !accounts.SupportsReference(destinationType) {
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

// exportAccountRef rewrites the config's literal account id to a
// "#account:<id>" reference for a type that supports one, so an imported
// project names the account spec that import writes beside it. A type without
// the capability, or an account the CLI does not manage, keeps the literal.
func exportAccountRef(destinationType string, config map[string]any, inputResolver resolver.ReferenceResolver) error {
	remoteID, ok := config[accountConfigKey].(string)
	if !ok || remoteID == "" || !accounts.SupportsReference(destinationType) {
		return nil
	}
	ref, err := inputResolver.ResolveToReference(accounts.AccountResourceType, remoteID)
	if err != nil {
		// The account is not in this import and not managed, so there is
		// nothing to reference. The literal is correct and still applies.
		return nil
	}
	config[accountConfigKey] = ref
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

// adoptedAccountExternalIDs maps each managed account's remote id to the
// external id its spec claims.
func adoptedAccountExternalIDs(st *state.State) map[string]string {
	if st == nil {
		return nil
	}
	byRemoteID := map[string]string{}
	for _, resource := range st.Resources {
		if resource.Type != accounts.AccountResourceType {
			continue
		}
		if accountState, ok := resource.OutputRaw.(*accounts.AccountState); ok && accountState.ID != "" {
			byRemoteID[accountState.ID] = resource.ID
		}
	}
	return byRemoteID
}

// checkAdoptedAccountLiterals refuses a plan where a destination names an
// account this project already manages by the account's remote id.
//
// Remote state maps such an account as a reference (remoteAccountRef), so a
// literal in the spec is not equal to it and the destination would be rewritten
// with an identical config on every apply. The two spellings mean the same
// account, so the fix is a one-line spec edit rather than anything the CLI can
// decide; saying so beats a diff that never goes away.
func checkAdoptedAccountLiterals(plan *planner.Plan, st *state.State) error {
	adopted := adoptedAccountExternalIDs(st)
	if len(adopted) == 0 {
		return nil
	}

	var refused []string
	for _, op := range plan.Operations {
		dest, ok := op.Resource.RawData().(*DestinationResource)
		if !ok || !accounts.SupportsReference(dest.Type) {
			continue
		}
		remoteID, isLiteral := dest.Config[accountConfigKey].(string)
		if !isLiteral {
			continue
		}
		if externalID, isAdopted := adopted[remoteID]; isAdopted {
			refused = append(refused, fmt.Sprintf(
				"%s names account %q; write %s%s instead",
				op.Resource.URN(), remoteID, accountRefPrefix, externalID))
		}
	}
	if len(refused) == 0 {
		return nil
	}
	return fmt.Errorf("a destination names an account this project manages by its remote ID. "+
		"Reference the account spec instead, or the destination is rewritten with an identical "+
		"config on every apply:\n  %s", strings.Join(refused, "\n  "))
}
