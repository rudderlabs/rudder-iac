package reporters

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestPlanReporter(t *testing.T) {
	// Test default behavior (flat mode) with all resource types
	var buf bytes.Buffer

	r := &planReporter{}
	r.SetWriter(&buf)

	diff := &differ.Diff{
		ImportableResources: []string{"importable.resource1", "importable.resource2"},
		NewResources:        []string{"new.resource1"},
		UpdatedResources: map[string]differ.ResourceDiff{
			"updated.resource1": {
				URN: "updated.resource1",
				Diffs: map[string]differ.PropertyDiff{
					"name": {
						Property:    "name",
						SourceValue: "old-name",
						TargetValue: "new-name",
					},
					"size": {
						Property:    "size",
						SourceValue: 10,
						TargetValue: 20,
					},
					"complex": {
						Property:    "complex",
						SourceValue: map[string]any{"a": 1, "b": 2},
						TargetValue: map[string]any{"a": 1, "b": 3},
					},
				},
			},
			"updated.resource2": {
				URN: "updated.resource2",
				Diffs: map[string]differ.PropertyDiff{
					"items": {
						Property:    "items",
						SourceValue: []any{1, 2, 3},
						TargetValue: []any{1, 5, 3},
					},
				},
			},
		},
		RemovedResources: []string{"removed.resource1"},
	}

	r.ReportPlan(&planner.Plan{Diff: diff})

	expectedOutput := `Importable resources:
  - importable.resource1
  - importable.resource2

New resources:
  - new.resource1

Updated resources:
  - updated.resource1
    - complex: map[a:1 b:2] => map[a:1 b:3]
    - name: old-name => new-name
    - size: 10 => 20
  - updated.resource2
    - items: [1 2 3] => [1 5 3]

Removed resources:
  - removed.resource1

`

	assert.Equal(t, expectedOutput, buf.String())
}

func TestPlanReporter_NestedDiff(t *testing.T) {
	enableNestedDiffs(t)

	var buf bytes.Buffer

	r := &planReporter{}
	r.SetWriter(&buf)

	diff := &differ.Diff{
		UpdatedResources: map[string]differ.ResourceDiff{
			"resource.with_nested_maps": {
				URN: "resource.with_nested_maps",
				Diffs: map[string]differ.PropertyDiff{
					"name": {
						Property:    "name",
						SourceValue: "old-name",
						TargetValue: "new-name",
					},
					"size": {
						Property:    "size",
						SourceValue: 10,
						TargetValue: 20,
					},
					"complex": {
						Property:    "complex",
						SourceValue: map[string]any{"a": 1, "b": 2},
						TargetValue: map[string]any{"a": 1, "b": 3},
					},
				},
			},
			"resource.with_arrays": {
				URN: "resource.with_arrays",
				Diffs: map[string]differ.PropertyDiff{
					"items": {
						Property:    "items",
						SourceValue: []any{1, 2, 3},
						TargetValue: []any{1, 5, 3},
					},
				},
			},
			"resource.with_mixed_structures": {
				URN: "resource.with_mixed_structures",
				Diffs: map[string]differ.PropertyDiff{
					"config": {
						Property: "config",
						SourceValue: map[string]any{
							"servers": []any{
								map[string]any{"host": "a.com", "port": 80},
								map[string]any{"host": "b.com", "port": 443},
							},
						},
						TargetValue: map[string]any{
							"servers": []any{
								map[string]any{"host": "a.com", "port": 80},
								map[string]any{"host": "b.com", "port": 8443},
							},
						},
					},
				},
			},
			"resource.with_property_refs": {
				URN: "resource.with_property_refs",
				Diffs: map[string]differ.PropertyDiff{
					"properties": {
						Property: "properties",
						SourceValue: []any{
							map[string]any{
								"id":       resources.PropertyRef{URN: "property:slotPosition", Property: "id"},
								"localId":  "slotPosition",
								"required": false,
							},
							map[string]any{
								"id":       resources.PropertyRef{URN: "property:slotType", Property: "id"},
								"localId":  "slotType",
								"required": false,
							},
							map[string]any{
								"id":       resources.PropertyRef{URN: "property:totalSlotItems", Property: "id"},
								"localId":  "totalSlotItems",
								"required": true,
							},
						},
						TargetValue: []any{
							map[string]any{
								"id":       resources.PropertyRef{URN: "property:slotType", Property: "id"},
								"localId":  "slotType",
								"required": true,
							},
							map[string]any{
								"id":       resources.PropertyRef{URN: "property:totalSlotItems", Property: "id"},
								"localId":  "totalSlotItems",
								"required": true,
							},
						},
					},
				},
			},
		},
	}

	r.ReportPlan(&planner.Plan{Diff: diff})

	expectedOutput := `Updated resources:
  - resource.with_arrays
    - items.1: 2 => 5
  - resource.with_mixed_structures
    - config.servers.1.port: 443 => 8443
  - resource.with_nested_maps
    - complex.b: 2 => 3
    - name: old-name => new-name
    - size: 10 => 20
  - resource.with_property_refs
    - properties.0.id: property:slotPosition => property:slotType
    - properties.0.localId: slotPosition => slotType
    - properties.0.required: false => true
    - properties.1.id: property:slotType => property:totalSlotItems
    - properties.1.localId: slotType => totalSlotItems
    - properties.1.required: false => true
    - properties.2: map[id:{property:totalSlotItems id false  <nil>} localId:totalSlotItems required:true] => <nil>

`

	output := buf.String()

	// Verify PropertyRefs are displayed as URNs in the nested diffs
	assert.Contains(t, output, "property:slotPosition => property:slotType")
	assert.Contains(t, output, "property:slotType => property:totalSlotItems")

	// Verify all three array indices are reported (not just the last one)
	assert.Contains(t, output, "properties.0")
	assert.Contains(t, output, "properties.1")
	assert.Contains(t, output, "properties.2")

	assert.Equal(t, expectedOutput, output)
}

func enableNestedDiffs(t *testing.T) {
	t.Helper()

	viper.Set("experimental", true)
	viper.Set("flags.nestedDiffs", true)

	t.Cleanup(func() {
		viper.Set("experimental", false)
		viper.Set("flags.nestedDiffs", false)
	})
}
