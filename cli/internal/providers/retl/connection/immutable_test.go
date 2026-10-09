package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

const planURN = "retl-connection:sf-to-amp"

// planDiff builds the diff the syncer computes for a connection: the stored
// entry on the source side, the desired entry on the target side, compared by the
// real differ. A nested config change lands under the top-level "config" key,
// which is what the check has to cope with.
func planDiff(t *testing.T, stored, desired resources.ResourceData) *differ.Diff {
	t.Helper()

	diffs, secretOnly := differ.CompareData(stored, desired)
	require.NotEmpty(t, diffs)
	return &differ.Diff{UpdatedResources: map[string]differ.ResourceDiff{
		planURN: {URN: planURN, Diffs: diffs, SecretOnly: secretOnly},
	}}
}

func TestCheckImmutableChanges(t *testing.T) {
	t.Parallel()

	withConfig := func(change func(*ConfigSpec)) ConfigSpec {
		config := jsonMapperConfig()
		change(&config)
		return config
	}

	tests := []struct {
		name    string
		desired ConfigSpec
		wantErr string
	}{
		{
			name:    "refuses a sync_behaviour change",
			desired: withConfig(func(c *ConfigSpec) { c.SyncBehaviour = "mirror" }),
			wantErr: `retl-connection:sf-to-amp: connection update: sync_behaviour is immutable ("upsert" -> "mirror")`,
		},
		{
			name:    "refuses an event change",
			desired: withConfig(func(c *ConfigSpec) { c.Event = &EventSpec{Type: "track", Name: "Synced"} }),
			wantErr: "retl-connection:sf-to-amp: connection update: event is immutable",
		},
		{
			name:    "allows a schedule change",
			desired: withConfig(func(c *ConfigSpec) { c.Schedule = ScheduleSpec{Type: "basic", EveryMinutes: ptr(60)} }),
		},
		{
			name:    "allows a mapping change",
			desired: withConfig(func(c *ConfigSpec) { c.Mappings = nil }),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckImmutableChanges(planDiff(t, graphData(t, jsonMapperConfig()), graphData(t, tt.desired)))

			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantErr)
			assert.ErrorContains(t, err, "delete and recreate")
		})
	}
}

func TestCheckImmutableChanges_AllowsReplacement(t *testing.T) {
	t.Parallel()

	// A moved endpoint recreates the connection, which carries the new config.
	stored := graphData(t, jsonMapperConfig())
	desired := graphData(t, func() ConfigSpec { c := jsonMapperConfig(); c.SyncBehaviour = "mirror"; return c }())
	desired[DestinationKey] = "dst-2"

	assert.NoError(t, CheckImmutableChanges(planDiff(t, stored, desired)))
}

func TestCheckImmutableChanges_IgnoresOtherKinds(t *testing.T) {
	t.Parallel()

	diff := planDiff(t,
		graphData(t, jsonMapperConfig()),
		graphData(t, func() ConfigSpec { c := jsonMapperConfig(); c.SyncBehaviour = "mirror"; return c }()),
	)
	diff.UpdatedResources["retl-source-table:t"] = diff.UpdatedResources[planURN]
	delete(diff.UpdatedResources, planURN)

	assert.NoError(t, CheckImmutableChanges(diff))
	assert.NoError(t, CheckImmutableChanges(&differ.Diff{}))
}
