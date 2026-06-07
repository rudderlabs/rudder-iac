package resourceops_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

// buildManagedRemoteResources returns a RemoteResources collection containing
// a single managed event-stream-source with the given remote ID and external ID.
func buildManagedRemoteResources(remoteID, externalID string) *resources.RemoteResources {
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		remoteID: {ID: remoteID, ExternalID: externalID, Data: map[string]any{"name": "My Source"}},
	})
	return rc
}

// buildStateForSource constructs a state.State with a ResourceState for the
// given external ID, matching the URN pattern "event-stream-source:<externalID>".
func buildStateForSource(externalID string) *state.State {
	st := state.EmptyState()
	st.AddResource(&state.ResourceState{
		ID:   externalID,
		Type: "event-stream-source",
		Output: map[string]any{
			"id":   "src_1",
			"name": "My Source",
		},
	})
	return st
}

func TestDelete_ManagedByExternalID(t *testing.T) {
	t.Parallel()

	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	mp.LoadResourcesFromRemoteVal = buildManagedRemoteResources("src_1", "my-source")

	st := buildStateForSource("my-source")
	mp.MapRemoteToStateVal = st

	err := resourceops.Delete(context.Background(), mp, "event-stream-source", "my-source")
	require.NoError(t, err)

	// The provider should have been called with the external ID and correct type.
	assert.Equal(t, "my-source", mp.DeleteCalledWithArg.ID)
	assert.Equal(t, "event-stream-source", mp.DeleteCalledWithArg.ResourceType)

	// Verify the state data passed to Delete matches what Data() would produce
	// from the resource state (Input+Output merged).
	sr := st.GetResource(resources.URN("my-source", "event-stream-source"))
	require.NotNil(t, sr)
	assert.Equal(t, testutils.DeleteArgs{
		ID:           "my-source",
		ResourceType: "event-stream-source",
		State:        sr.Data(),
	}, mp.DeleteCalledWithArg)
}

func TestDelete_ManagedByRemoteID(t *testing.T) {
	t.Parallel()

	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	mp.LoadResourcesFromRemoteVal = buildManagedRemoteResources("src_1", "my-source")

	st := buildStateForSource("my-source")
	mp.MapRemoteToStateVal = st

	// Resolve by remote ID "src_1" — should still call Delete with the external ID.
	err := resourceops.Delete(context.Background(), mp, "event-stream-source", "src_1")
	require.NoError(t, err)

	assert.Equal(t, "my-source", mp.DeleteCalledWithArg.ID)
	assert.Equal(t, "event-stream-source", mp.DeleteCalledWithArg.ResourceType)
}

func TestDelete_Unmanaged_RejectsWithErrUnmanaged(t *testing.T) {
	t.Parallel()

	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_2": {ID: "src_2", ExternalID: "", Data: map[string]any{"name": "Unmanaged Source"}},
	})
	mp.LoadResourcesFromRemoteVal = rc

	err := resourceops.Delete(context.Background(), mp, "event-stream-source", "src_2")
	require.Error(t, err)
	assert.ErrorIs(t, err, resourceops.ErrUnmanaged)

	// Delete must not have been called on the provider.
	assert.Equal(t, testutils.DeleteArgs{}, mp.DeleteCalledWithArg)
}

func TestDelete_NotFound_RejectsWithErrResourceNotFound(t *testing.T) {
	t.Parallel()

	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	mp.LoadResourcesFromRemoteVal = resources.NewRemoteResources() // empty

	err := resourceops.Delete(context.Background(), mp, "event-stream-source", "ghost")
	require.Error(t, err)
	assert.ErrorIs(t, err, resourceops.ErrResourceNotFound)

	// Delete must not have been called on the provider.
	assert.Equal(t, testutils.DeleteArgs{}, mp.DeleteCalledWithArg)
}
