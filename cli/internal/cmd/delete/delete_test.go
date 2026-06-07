package delete_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	deletecmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/delete"
	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

// buildMockProvider returns a MockProvider configured with one managed resource
// (remoteID "src_1", externalID "my-source") and the matching state.
func buildMockProvider() *testutils.MockProvider {
	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})

	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "my-source", Data: map[string]any{"name": "My Source"}},
	})
	mp.LoadResourcesFromRemoteVal = rc

	st := state.EmptyState()
	st.AddResource(&state.ResourceState{
		ID:   "my-source",
		Type: "event-stream-source",
		Output: map[string]any{
			"id":   "src_1",
			"name": "My Source",
		},
	})
	mp.MapRemoteToStateVal = st

	return mp
}

func confirmAlways(_ string) (bool, error) { return true, nil }
func confirmNever(_ string) (bool, error)  { return false, nil }

func TestRunDelete_SkipConfirm_Deletes(t *testing.T) {
	t.Parallel()

	mp := buildMockProvider()
	var buf bytes.Buffer

	err := deletecmd.RunDelete(context.Background(), &buf, mp, "event-stream-source", "my-source", true, confirmNever)
	require.NoError(t, err)

	// Confirm Delete was called with the expected args.
	assert.Equal(t, "my-source", mp.DeleteCalledWithArg.ID)
	assert.Equal(t, "event-stream-source", mp.DeleteCalledWithArg.ResourceType)

	// Output should mention success.
	out := buf.String()
	assert.Contains(t, out, "Deleted")
	assert.Contains(t, out, "my-source")
}

func TestRunDelete_ConfirmTrue_Deletes(t *testing.T) {
	t.Parallel()

	mp := buildMockProvider()
	var buf bytes.Buffer

	err := deletecmd.RunDelete(context.Background(), &buf, mp, "event-stream-source", "my-source", false, confirmAlways)
	require.NoError(t, err)

	assert.Equal(t, "my-source", mp.DeleteCalledWithArg.ID)
	assert.Equal(t, "event-stream-source", mp.DeleteCalledWithArg.ResourceType)

	out := buf.String()
	assert.Contains(t, out, "Deleted")
}

func TestRunDelete_ConfirmFalse_Aborts(t *testing.T) {
	t.Parallel()

	mp := buildMockProvider()
	var buf bytes.Buffer

	err := deletecmd.RunDelete(context.Background(), &buf, mp, "event-stream-source", "my-source", false, confirmNever)
	require.NoError(t, err)

	// Delete must NOT have been called.
	assert.Equal(t, testutils.DeleteArgs{}, mp.DeleteCalledWithArg)

	// Output must mention aborted.
	out := buf.String()
	assert.Contains(t, out, "aborted")
}

func TestRunDelete_Unmanaged_ReturnsErrUnmanaged(t *testing.T) {
	t.Parallel()

	mp := testutils.NewMockProvider(nil, []string{"event-stream-source"})
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_unmanaged": {ID: "src_unmanaged", ExternalID: "", Data: map[string]any{"name": "Unmanaged"}},
	})
	mp.LoadResourcesFromRemoteVal = rc

	var buf bytes.Buffer
	err := deletecmd.RunDelete(context.Background(), &buf, mp, "event-stream-source", "src_unmanaged", true, confirmAlways)
	require.Error(t, err)
	assert.ErrorIs(t, err, resourceops.ErrUnmanaged)

	// Delete must not have been called.
	assert.Equal(t, testutils.DeleteArgs{}, mp.DeleteCalledWithArg)
}

func TestRunDelete_ConfirmFnError_ReturnsError(t *testing.T) {
	t.Parallel()

	mp := buildMockProvider()
	var buf bytes.Buffer
	sentinel := errors.New("prompt error")

	confirmErr := func(_ string) (bool, error) { return false, sentinel }

	err := deletecmd.RunDelete(context.Background(), &buf, mp, "event-stream-source", "my-source", false, confirmErr)
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)

	// Delete must not have been called.
	assert.Equal(t, testutils.DeleteArgs{}, mp.DeleteCalledWithArg)
}

func TestNewCmdDelete_Registered(t *testing.T) {
	t.Parallel()

	cmd := deletecmd.NewCmdDelete()
	require.NotNil(t, cmd)

	assert.Equal(t, "delete", cmd.Name())

	// Exactly 2 args required.
	assert.Error(t, cmd.Args(cmd, []string{}), "zero args must be rejected")
	assert.Error(t, cmd.Args(cmd, []string{"a"}), "one arg must be rejected")
	assert.NoError(t, cmd.Args(cmd, []string{"a", "b"}), "two args must be accepted")
	assert.Error(t, cmd.Args(cmd, []string{"a", "b", "c"}), "three args must be rejected")

	// --confirm flag must be present and default to false.
	confirmFlag := cmd.Flags().Lookup("confirm")
	require.NotNil(t, confirmFlag, "--confirm flag must be registered")
	assert.Equal(t, "false", confirmFlag.DefValue)
}
