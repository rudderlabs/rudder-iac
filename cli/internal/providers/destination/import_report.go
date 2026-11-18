package destination

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importreport"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

var (
	_ provider.ImportReporter = &Provider{}
	_ provider.ImportGuard    = &Provider{}
)

// unmodelledConfig returns the parts of a remote destination config that
// local config cannot hold, so adopting the destination removes them.
//
// Local config holds what survives an API to local to API round trip, so a part
// of the remote config missing from the round trip is one the definition does
// not model: a legacy block, a consent entry for a source type the definition
// omits, a field of an array element it does not map. A modelled key the spec
// leaves out is a plain diff, and an empty value is the webapp's noise, so
// neither is listed.
//
// Values are summarised, never returned.
func unmodelledConfig(registered *definitions.RegisteredDefinition, raw json.RawMessage) ([]importreport.Lost, error) {
	var remote map[string]any
	if err := json.Unmarshal(raw, &remote); err != nil {
		return nil, fmt.Errorf("unmarshalling destination config: %w", err)
	}
	local, err := registered.APIToLocal(remote)
	if err != nil {
		return nil, fmt.Errorf("converting API config to local: %w", err)
	}
	roundTrip, err := registered.LocalToAPI(local)
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
	}

	// A discriminator naming no member says what its absence says, so the round
	// trip dropping it loses nothing.
	neutral := map[string]bool{}
	for _, prop := range registered.Properties {
		if prop.Selector == nil {
			continue
		}
		if selected, present := prop.Selector.LocalKeyFor(remote); present && selected == "" {
			neutral[prop.Selector.APIKey] = true
		}
	}

	var lost []importreport.Lost
	for key, value := range remote {
		if neutral[key] {
			continue
		}
		collectUnmodelled(value, roundTrip[key], key, roundTripHas(roundTrip, key), &lost)
	}
	slices.SortFunc(lost, func(a, b importreport.Lost) int { return strings.Compare(a.Path, b.Path) })
	return lost, nil
}

func roundTripHas(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

// collectUnmodelled walks value against kept, the same part of the round trip,
// down maps by key and lists by index. held says kept exists at all.
func collectUnmodelled(value, kept any, path string, held bool, lost *[]importreport.Lost) {
	if !held {
		if !isEmptyConfigValue(value) {
			*lost = append(*lost, importreport.Lost{Path: path, Value: summariseLost(value)})
		}
		return
	}

	switch remote := value.(type) {
	case map[string]any:
		keptMap, _ := kept.(map[string]any)
		for key, item := range remote {
			collectUnmodelled(item, keptMap[key], path+"."+key, roundTripHas(keptMap, key), lost)
		}
	case []any:
		keptList, _ := kept.([]any)
		for i, item := range remote {
			var keptItem any
			if i < len(keptList) {
				keptItem = keptList[i]
			}
			collectUnmodelled(item, keptItem, fmt.Sprintf("%s[%d]", path, i), i < len(keptList), lost)
		}
	}
}

// summariseLost describes a value by its shape, since an unmodelled key can hold
// a credential the definition does not know is one.
func summariseLost(value any) string {
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

// ImportReport lists, for each destination the import writes as a spec, the
// remote config the spec cannot carry. See provider.ImportReporter.
func (p *Provider) ImportReport(collection *resources.RemoteResources, entries []importmanifest.ImportEntry) ([]importreport.Entry, error) {
	urns := map[string]string{}
	for _, entry := range entries {
		urns[entry.RemoteID] = entry.URN
	}

	var report []importreport.Entry
	for _, res := range collection.GetAll(DestinationResourceType) {
		remote, ok := res.Data.(*RemoteDestination)
		if !ok || urns[remote.ID] == "" {
			continue
		}
		registered, err := p.registry.GetByAPIType(remote.Type, remote.Version)
		if err != nil {
			return nil, fmt.Errorf("getting definition for destination %s: %w", remote.ID, err)
		}
		lost, err := unmodelledConfig(registered, remote.Config)
		if err != nil {
			return nil, fmt.Errorf("reading destination %s config: %w", remote.ID, err)
		}
		if len(lost) == 0 {
			continue
		}
		report = append(report, importreport.Entry{
			URN:      urns[remote.ID],
			RemoteID: remote.ID,
			Name:     remote.Name,
			Lost:     lost,
		})
	}
	return report, nil
}

// CheckImports refuses to adopt a destination whose remote config holds parts
// the spec cannot carry, unless the report acknowledges them. The remote config
// is read now, not taken from the report, so a key added in the dashboard after
// the import is caught. See provider.ImportGuard.
func (p *Provider) CheckImports(ctx context.Context, plan *planner.Plan, _ *state.State, report *importreport.Report) error {
	var refused []string
	for _, op := range plan.Operations {
		if op.Type != planner.Import || op.Resource.Type() != DestinationResourceType || op.Resource.ImportMetadata() == nil {
			continue
		}
		remoteID := op.Resource.ImportMetadata().RemoteId

		remote, err := p.handler.client.Destinations.Get(ctx, remoteID)
		if err != nil {
			return fmt.Errorf("getting destination %s to check its import: %w", op.Resource.URN(), err)
		}
		registered, err := p.registry.GetByAPIType(remote.Type, remote.Version)
		if err != nil {
			return fmt.Errorf("getting definition for destination %s: %w", op.Resource.URN(), err)
		}
		lost, err := unmodelledConfig(registered, remote.Config)
		if err != nil {
			return fmt.Errorf("reading destination %s config: %w", op.Resource.URN(), err)
		}

		uncovered := report.Uncovered(remoteID, lost)
		if len(uncovered) == 0 {
			continue
		}
		refused = append(refused, op.Resource.URN())
		for _, item := range uncovered {
			refused = append(refused, fmt.Sprintf("    %s %s", item.Path, item.Value))
		}
	}
	if len(refused) == 0 {
		return nil
	}

	return fmt.Errorf("adopting these destinations removes remote config the spec cannot hold, and the "+
		"import report does not acknowledge it. Read the paths, then set \"acknowledged\": true on the "+
		"destination's entry in %s (written by `import workspace`; add the entry by hand if there is none). "+
		"Nothing has been changed.\n  %s", importreport.FileName, strings.Join(refused, "\n  "))
}
