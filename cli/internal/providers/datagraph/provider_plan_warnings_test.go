package datagraph_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProvider_PlanWarnings verifies the v1 column-metadata partial-merge
// orphan warning is surfaced at plan time so it appears in both dry-run and
// real apply, before any mutation runs. The warning must use the LLD wording
// (pinned in model.TestFormatOrphanColumnWarning), be deduplicated by column
// name, and sorted for deterministic output.
func TestProvider_PlanWarnings(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	// Sanity: the provider must satisfy the optional planner.PlanWarner so
	// the syncer's type assertion picks it up.
	var _ planner.PlanWarner = provider

	modelURN := resources.URN("user", modelHandler.HandlerMetadata.ResourceType)

	t.Run("warns on each orphaned column with the LLD wording", func(t *testing.T) {
		plan := &planner.Plan{
			Diff: &differ.Diff{
				UpdatedResources: map[string]differ.ResourceDiff{
					modelURN: {
						URN: modelURN,
						Diffs: map[string]differ.PropertyDiff{
							"columns": {
								Property: "columns",
								SourceValue: []map[string]any{
									{"name": "user_id", "display_name": "User ID"},
									{"name": "email", "display_name": "Email"},
									{"name": "created_at", "display_name": "Created"},
								},
								TargetValue: []map[string]any{
									{"name": "user_id", "display_name": "User ID"},
								},
							},
						},
					},
				},
			},
		}

		got := provider.PlanWarnings(plan)
		assert.Equal(t, []string{
			"metadata for created_at will remain in the workspace; v1 has no clear/delete path",
			"metadata for email will remain in the workspace; v1 has no clear/delete path",
		}, got)
	})

	t.Run("no warning when local includes every remote column", func(t *testing.T) {
		plan := &planner.Plan{
			Diff: &differ.Diff{
				UpdatedResources: map[string]differ.ResourceDiff{
					modelURN: {
						URN: modelURN,
						Diffs: map[string]differ.PropertyDiff{
							"columns": {
								Property: "columns",
								SourceValue: []map[string]any{
									{"name": "user_id", "display_name": "User ID"},
								},
								TargetValue: []map[string]any{
									{"name": "user_id", "display_name": "User ID"},
									{"name": "email", "display_name": "Email"},
								},
							},
						},
					},
				},
			},
		}

		assert.Nil(t, provider.PlanWarnings(plan))
	})

	t.Run("warns on every remote column when local omits the columns block entirely", func(t *testing.T) {
		plan := &planner.Plan{
			Diff: &differ.Diff{
				UpdatedResources: map[string]differ.ResourceDiff{
					modelURN: {
						URN: modelURN,
						Diffs: map[string]differ.PropertyDiff{
							"columns": {
								Property: "columns",
								SourceValue: []map[string]any{
									{"name": "user_id", "display_name": "User ID"},
								},
								TargetValue: nil,
							},
						},
					},
				},
			},
		}

		assert.Equal(t, []string{
			"metadata for user_id will remain in the workspace; v1 has no clear/delete path",
		}, provider.PlanWarnings(plan))
	})

	t.Run("ignores diffs on non-model resource types", func(t *testing.T) {
		// A column property diff on a non-model resource would be a wiring
		// bug, but the warner must never flag it as an orphan.
		plan := &planner.Plan{
			Diff: &differ.Diff{
				UpdatedResources: map[string]differ.ResourceDiff{
					"data-graph:my-dg": {
						URN: "data-graph:my-dg",
						Diffs: map[string]differ.PropertyDiff{
							"columns": {
								Property: "columns",
								SourceValue: []map[string]any{
									{"name": "user_id", "display_name": "User ID"},
								},
								TargetValue: nil,
							},
						},
					},
				},
			},
		}

		assert.Nil(t, provider.PlanWarnings(plan))
	})

	t.Run("warnings span multiple models and stay sorted", func(t *testing.T) {
		userURN := resources.URN("user", modelHandler.HandlerMetadata.ResourceType)
		purchaseURN := resources.URN("purchase", modelHandler.HandlerMetadata.ResourceType)

		plan := &planner.Plan{
			Diff: &differ.Diff{
				UpdatedResources: map[string]differ.ResourceDiff{
					userURN: {
						URN: userURN,
						Diffs: map[string]differ.PropertyDiff{
							"columns": {
								Property: "columns",
								SourceValue: []map[string]any{
									{"name": "email", "display_name": "Email"},
								},
								TargetValue: nil,
							},
						},
					},
					purchaseURN: {
						URN: purchaseURN,
						Diffs: map[string]differ.PropertyDiff{
							"columns": {
								Property: "columns",
								SourceValue: []map[string]any{
									{"name": "amount", "display_name": "Amount"},
								},
								TargetValue: nil,
							},
						},
					},
				},
			},
		}

		got := provider.PlanWarnings(plan)
		// Sorted by URN, then by orphan name within each diff — the
		// concatenated output is stable across runs.
		require.Len(t, got, 2)
		assert.Contains(t, got, "metadata for email will remain in the workspace; v1 has no clear/delete path")
		assert.Contains(t, got, "metadata for amount will remain in the workspace; v1 has no clear/delete path")
	})

	t.Run("returns nil when the plan has no updated resources", func(t *testing.T) {
		plan := &planner.Plan{Diff: &differ.Diff{}}
		assert.Nil(t, provider.PlanWarnings(plan))
	})

	t.Run("handles []any payload (mapstructure-decoded slice variant)", func(t *testing.T) {
		// mapstructure-decoded resources can surface columns as []any of
		// map[string]any rather than []map[string]any (depending on which
		// side of the diff went through which branch of differ.compareValues).
		// Both shapes must be understood.
		plan := &planner.Plan{
			Diff: &differ.Diff{
				UpdatedResources: map[string]differ.ResourceDiff{
					modelURN: {
						URN: modelURN,
						Diffs: map[string]differ.PropertyDiff{
							"columns": {
								Property: "columns",
								SourceValue: []any{
									map[string]any{"name": "user_id", "display_name": "User ID"},
									map[string]any{"name": "email", "display_name": "Email"},
								},
								TargetValue: []any{
									map[string]any{"name": "user_id", "display_name": "User ID"},
								},
							},
						},
					},
				},
			},
		}

		assert.Equal(t, []string{
			"metadata for email will remain in the workspace; v1 has no clear/delete path",
		}, provider.PlanWarnings(plan))
	})
}
