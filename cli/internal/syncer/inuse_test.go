package syncer

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deletePlan(urns ...string) *planner.Plan {
	plan := &planner.Plan{}
	for _, urn := range urns {
		// URN is "<type>:<id>"; the guard only reads it back out.
		r := resources.NewResource(urn[len("account:"):], "account", resources.ResourceData{}, nil)
		plan.Operations = append(plan.Operations, &planner.Operation{Type: planner.Delete, Resource: r})
	}
	return plan
}

func stateWith(urn, remoteID string) *state.State {
	st := state.EmptyState()
	st.Resources[urn] = &state.ResourceState{ID: remoteID, Type: "account"}
	return st
}

func targetWith(data resources.ResourceData) *resources.Graph {
	g := resources.NewGraph()
	g.AddResource(resources.NewResource("snf-test-source", "retl-source-table", data, nil))
	return g
}

// The repro from DEX-959: the source names the account by raw id, so nothing in
// the graph connects them and the delete used to go through silently.
func TestGuardInUseDeletes_RefusesRawIDReference(t *testing.T) {
	err := guardInUseDeletes(
		deletePlan("account:snf-test"),
		stateWith("account:snf-test", "3JjJul5DLlIlDowCs6CpO1fsOPZ"),
		targetWith(resources.ResourceData{"account_id": "3JjJul5DLlIlDowCs6CpO1fsOPZ", "table": "ALLBIRD"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account:snf-test")
	assert.Contains(t, err.Error(), "retl-source-table:snf-test-source")
	assert.Contains(t, err.Error(), "account_id", "the message must name the field holding the reference")
}

// The reference is gone from the project too, so the delete is what the user asked for.
func TestGuardInUseDeletes_AllowsUnreferencedDelete(t *testing.T) {
	err := guardInUseDeletes(
		deletePlan("account:snf-test"),
		stateWith("account:snf-test", "3JjJul5DLlIlDowCs6CpO1fsOPZ"),
		targetWith(resources.ResourceData{"account_id": "some-other-account", "table": "ALLBIRD"}),
	)
	assert.NoError(t, err)
}

// A nested id must not slip past the scan.
func TestGuardInUseDeletes_FindsNestedReference(t *testing.T) {
	err := guardInUseDeletes(
		deletePlan("account:snf-test"),
		stateWith("account:snf-test", "remote-1"),
		targetWith(resources.ResourceData{
			"config": map[string]any{"sources": []any{map[string]any{"account_id": "remote-1"}}},
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config.sources[0].account_id")
}

// Nothing to compare against: a delete of a resource with no recorded remote id
// must not block the apply.
func TestGuardInUseDeletes_IgnoresResourcesWithoutState(t *testing.T) {
	err := guardInUseDeletes(
		deletePlan("account:snf-test"),
		state.EmptyState(),
		targetWith(resources.ResourceData{"account_id": "3JjJul5DLlIlDowCs6CpO1fsOPZ"}),
	)
	assert.NoError(t, err)
}
