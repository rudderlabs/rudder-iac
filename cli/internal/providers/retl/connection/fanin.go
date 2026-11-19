package connection

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

// CheckFanIn vets all planned rETL writes against the entire workspace, not
// just the rows mapped into CLI state. Importing an existing row without moving
// its endpoints is allowed even if the workspace already has fan-in. Writes
// must not introduce a new source into a destination's fan-in, including when
// an existing connection keeps its remote ID but repoints its source. The offline
// semantic rule handles event-stream edges. This check precedes apply and dry run.
func (h *Handler) CheckFanIn(ctx context.Context, plan *planner.Plan, st *state.State) error {
	var writes []*planner.Operation
	for _, op := range plan.Operations {
		if op.Resource.Type() != ResourceType || op.Type == planner.Delete {
			continue
		}
		if op.Type == planner.Create || op.Type == planner.Import || op.Type == planner.Update {
			writes = append(writes, op)
		}
	}
	if len(writes) == 0 {
		return nil
	}

	// The unfiltered list matters: unsupported and unmanaged remote rows still
	// occupy the destination. Fetch each import by its declared remote id; a
	// connection may already be CLI-managed even when it is missing from state.
	remote, err := h.listAll(ctx, nil)
	if err != nil {
		return fmt.Errorf("checking rETL connection fan-in: %w", err)
	}
	byID := make(map[string]struct{ source, destination string }, len(remote))
	byDestination := make(map[string][]int)
	for index, row := range remote {
		byID[row.ID] = struct{ source, destination string }{row.SourceID, row.DestinationID}
		byDestination[row.DestinationID] = append(byDestination[row.DestinationID], index)
	}

	// Resolve endpoint IDs from state or declared imports; newly created
	// endpoints have no remote ID yet and are compared by local URN instead.
	imports := make(map[string]string)
	creates := make(map[string]bool)
	for _, op := range plan.Operations {
		if op.Type == planner.Create {
			creates[op.Resource.URN()] = true
		}
		if op.Type == planner.Import {
			if metadata := op.Resource.ImportMetadata(); metadata != nil {
				imports[op.Resource.URN()] = metadata.RemoteId
			}
		}
	}
	sources, destinations, err := h.endpoints(ctx)
	if err != nil {
		return fmt.Errorf("checking rETL connection endpoints: %w", err)
	}
	endpointID := func(urn string, catalogContains func(string) bool) (string, error) {
		id := imports[urn]
		if previous := st.GetResource(urn); previous != nil {
			if stored, ok := previous.Output[IDKey].(string); ok && stored != "" {
				id = stored
			}
			if typed, ok := previous.OutputRaw.(*destination.DestinationState); ok && typed != nil && typed.ID != "" {
				id = typed.ID
			}
		}
		if id == "" {
			if creates[urn] {
				return "", nil
			}
			return "", fmt.Errorf("endpoint %q has no remote id or planned create", urn)
		}
		if !catalogContains(id) {
			return "", fmt.Errorf("endpoint %q (remote id %q) was not found in this workspace", urn, id)
		}
		return id, nil
	}

	// Source and destination refs on the plan are local URNs. Compare known
	// remote IDs to catch aliases; fall back to URNs for planned creates.
	// Include unchanged imports/updates: they remain occupants of the planned
	// destination, even though they need no replacement themselves.
	planned := make(map[string]map[string]bool)
	changed := make(map[string]bool)
	for _, op := range writes {
		urn := op.Resource.URN()
		data := op.Resource.Data()
		sourceURN, err := plannedEndpointURN(data, SourceKey)
		if err != nil {
			return fmt.Errorf("checking %s: %w", urn, err)
		}
		destinationURN, err := plannedEndpointURN(data, DestinationKey)
		if err != nil {
			return fmt.Errorf("checking %s: %w", urn, err)
		}
		sourceID, err := endpointID(sourceURN, func(id string) bool { _, ok := sources[id]; return ok })
		if err != nil {
			return fmt.Errorf("checking %s: %w", urn, err)
		}
		remoteDestination, err := endpointID(destinationURN, func(id string) bool { _, ok := destinations[id]; return ok })
		if err != nil {
			return fmt.Errorf("checking %s: %w", urn, err)
		}
		var existingID string
		switch op.Type {
		case planner.Import:
			metadata := op.Resource.ImportMetadata()
			if metadata == nil || metadata.RemoteId == "" {
				return fmt.Errorf("checking %s: missing remote import id", urn)
			}
			existingID = metadata.RemoteId
		case planner.Update:
			if previous := st.GetResource(urn); previous != nil {
				existingID, _ = previous.Output[IDKey].(string)
			}
			if existingID == "" {
				return fmt.Errorf("checking %s: missing connection id in state", urn)
			}
		}
		unchanged := false
		if existingID != "" {
			old, ok := byID[existingID]
			if !ok {
				return fmt.Errorf("checking %s: remote connection %q was not found", urn, existingID)
			}
			// Compare remote endpoint IDs, just as replacementNeeded does
			// after dereferencing at apply time. An unresolved new endpoint
			// necessarily moves the pair; never grandfather it.
			unchanged = sourceID != "" && remoteDestination != "" && sourceID == old.source && remoteDestination == old.destination
		}
		destinationKey, sourceKey := destinationURN, sourceURN
		if remoteDestination != "" {
			destinationKey = "remote:" + remoteDestination
		}
		if sourceID != "" {
			sourceKey = "remote:" + sourceID
		}
		if planned[destinationKey] == nil {
			planned[destinationKey] = make(map[string]bool)
		}
		planned[destinationKey][sourceKey] = true
		if unchanged {
			continue
		}
		changed[destinationKey] = true

		// Check existing remote occupants even if they are not in the CLI
		// state. Exclude the row being replaced (which will be deleted).
		// A newly created destination has no remote occupants yet. Its
		// planned-group check below still catches two planned sources.
		for _, index := range byDestination[remoteDestination] {
			row := remote[index]
			if row.ID == existingID {
				continue
			}
			if sourceID != "" && row.SourceID == sourceID && existingID == "" {
				// New imports may already exist remotely under the same pair;
				// replacement writes must refuse it before deleting the old row.
				continue
			}
			return fmt.Errorf("cannot create or replace %s: destination %q already has rETL connection %q from source %q; this write would introduce rETL fan-in or replace an occupied pair", urn, destinationURN, row.ID, row.SourceID)
		}
	}
	for destinationURN, sources := range planned {
		if changed[destinationURN] && len(sources) > 1 {
			return fmt.Errorf("cannot create or replace rETL connections: destination %q has multiple planned sources [%s]", destinationURN, strings.Join(slices.Sorted(maps.Keys(sources)), ", "))
		}
	}
	return nil
}

func plannedEndpointURN(data resources.ResourceData, key string) (string, error) {
	ref, ok := data[key].(*resources.PropertyRef)
	if !ok || ref == nil || ref.URN == "" {
		return "", fmt.Errorf("missing %s reference", key)
	}
	return ref.URN, nil
}
