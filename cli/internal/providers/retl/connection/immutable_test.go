package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

func updated(urn string, props ...string) *differ.Diff {
	diffs := map[string]differ.PropertyDiff{}
	for _, p := range props {
		diffs[p] = differ.PropertyDiff{Property: p, SourceValue: "upsert", TargetValue: "mirror"}
	}
	return &differ.Diff{UpdatedResources: map[string]differ.ResourceDiff{urn: {URN: urn, Diffs: diffs}}}
}

// DEX-1020: the refusal used to come from Update, after unrelated resources in
// the same plan had been applied.
func TestCheckImmutableChanges_RefusesSyncBehaviourChange(t *testing.T) {
	err := CheckImmutableChanges(updated("retl-connection:sf-to-amp", "config.sync_behaviour"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "retl-connection:sf-to-amp")
	assert.Contains(t, err.Error(), "sync_behaviour is immutable (upsert -> mirror)")
	assert.Contains(t, err.Error(), "delete and recreate")
}

func TestCheckImmutableChanges_RefusesNestedEventChange(t *testing.T) {
	require.Error(t, CheckImmutableChanges(updated("retl-connection:c", "config.event.name")))
}

func TestCheckImmutableChanges_AllowsMutableChanges(t *testing.T) {
	assert.NoError(t, CheckImmutableChanges(updated("retl-connection:c", "config.schedule.every_minutes", "enabled")))
	// A key that merely starts with an immutable name is a different key.
	assert.NoError(t, CheckImmutableChanges(updated("retl-connection:c", "config.object_mappings")))
}

func TestCheckImmutableChanges_AllowsReplacement(t *testing.T) {
	// A moved endpoint recreates the connection, which carries the new config.
	assert.NoError(t, CheckImmutableChanges(updated("retl-connection:c", "config.sync_behaviour", "destination")))
	assert.NoError(t, CheckImmutableChanges(updated("retl-connection:c", "config.sync_behaviour", "source")))
}

func TestCheckImmutableChanges_IgnoresOtherKinds(t *testing.T) {
	assert.NoError(t, CheckImmutableChanges(updated("retl-source-table:t", "config.sync_behaviour")))
	assert.NoError(t, CheckImmutableChanges(&differ.Diff{}))
}
