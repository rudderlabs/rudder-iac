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

	var (
		oldRefURN = "data-graph:old-graph"
		newRefURN = "data-graph:new-graph"
	)

	diff := &differ.Diff{
		ImportableResources: []string{"resource_type:resource1", "resource_type:resource2"},
		NewResources:        []string{"resource_type:resource3"},
		UpdatedResources: map[string]differ.ResourceDiff{
			"resource_type:resource4": {
				URN: "resource_type:resource4",
				Diffs: map[string]differ.PropertyDiff{
					"name": {
						Property:    "name",
						SourceValue: "old-name",
						TargetValue: "new-name",
					},
					"ref_ptr_changed": {
						Property:    "ref_ptr_changed",
						SourceValue: &resources.PropertyRef{URN: oldRefURN},
						TargetValue: &resources.PropertyRef{URN: newRefURN},
					},
					"ref_ptr_nil_source": {
						Property:    "ref_ptr_nil_source",
						SourceValue: (*resources.PropertyRef)(nil),
						TargetValue: &resources.PropertyRef{URN: newRefURN},
					},
					"ref_ptr_nil_target": {
						Property:    "ref_ptr_nil_target",
						SourceValue: &resources.PropertyRef{URN: oldRefURN},
						TargetValue: (*resources.PropertyRef)(nil),
					},
					"ref_val_changed": {
						Property:    "ref_val_changed",
						SourceValue: resources.PropertyRef{URN: oldRefURN},
						TargetValue: resources.PropertyRef{URN: newRefURN},
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
			"resource_type:resource5": {
				URN: "resource_type:resource5",
				Diffs: map[string]differ.PropertyDiff{
					"items": {
						Property:    "items",
						SourceValue: []any{1, 2, 3},
						TargetValue: []any{1, 5, 3},
					},
				},
			},
		},
		RemovedResources: []string{"resource_type:resource6"},
	}

	r.ReportPlan(&planner.Plan{Diff: diff})

	expectedOutput := "Importable resources:\n" +
		"  - resource_type:resource1\n" +
		"  - resource_type:resource2\n" +
		"\n" +
		"New resources:\n" +
		"  - resource_type:resource3\n" +
		"\n" +
		"Updated resources:\n" +
		"  - resource_type:resource4\n" +
		"    - complex: map[a:1 b:2] => map[a:1 b:3]\n" +
		"    - name: old-name => new-name\n" +
		"    - ref_ptr_changed: " + oldRefURN + " => " + newRefURN + "\n" +
		"    - ref_ptr_nil_source: <nil> => " + newRefURN + "\n" +
		"    - ref_ptr_nil_target: " + oldRefURN + " => <nil>\n" +
		"    - ref_val_changed: " + oldRefURN + " => " + newRefURN + "\n" +
		"    - size: 10 => 20\n" +
		"  - resource_type:resource5\n" +
		"    - items: [1 2 3] => [1 5 3]\n" +
		"\n" +
		"Removed resources:\n" +
		"  - resource_type:resource6\n" +
		"\n"

	assert.Equal(t, expectedOutput, buf.String())
}

func TestPlanReporter_NestedDiff(t *testing.T) {
	enableNestedDiffs(t)

	var buf bytes.Buffer

	r := &planReporter{}
	r.SetWriter(&buf)

	diff := &differ.Diff{
		UpdatedResources: map[string]differ.ResourceDiff{
			"resource_type:with_nested_maps": {
				URN: "resource_type:with_nested_maps",
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
			"resource_type:with_arrays": {
				URN: "resource_type:with_arrays",
				Diffs: map[string]differ.PropertyDiff{
					"items": {
						Property:    "items",
						SourceValue: []any{1, 2, 3},
						TargetValue: []any{1, 5, 3},
					},
				},
			},
			"resource_type:with_mixed_structures": {
				URN: "resource_type:with_mixed_structures",
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
			"resource_type:with_property_refs": {
				URN: "resource_type:with_property_refs",
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
  - resource_type:with_arrays
    - items[1]: 2 => 5
  - resource_type:with_mixed_structures
    - config.servers[1].port: 443 => 8443
  - resource_type:with_nested_maps
    - complex.b: 2 => 3
    - name: old-name => new-name
    - size: 10 => 20
  - resource_type:with_property_refs
    - properties[0].id: property:slotPosition => property:slotType
    - properties[0].localId: slotPosition => slotType
    - properties[0].required: false => true
    - properties[1].id: property:slotType => property:totalSlotItems
    - properties[1].localId: slotType => totalSlotItems
    - properties[1].required: false => true
    - properties[2]: map[id:{property:totalSlotItems id false  <nil>} localId:totalSlotItems required:true] => <nil>

`

	output := buf.String()

	// Verify PropertyRefs are displayed as URNs in the nested diffs
	assert.Contains(t, output, "property:slotPosition => property:slotType")
	assert.Contains(t, output, "property:slotType => property:totalSlotItems")

	// Verify all three array indices are reported (not just the last one)
	assert.Contains(t, output, "properties[0]")
	assert.Contains(t, output, "properties[1]")
	assert.Contains(t, output, "properties[2]")

	assert.Equal(t, expectedOutput, output)
}

func TestPrintablePropertyRef(t *testing.T) {
	urn := "data-graph:my-graph"

	t.Run("pointer PropertyRef renders URN", func(t *testing.T) {
		ref := &resources.PropertyRef{URN: urn, Property: "id"}
		result := printable(ref)
		assert.Contains(t, result, urn)
		assert.NotContains(t, result, "{")
	})

	t.Run("value PropertyRef renders URN", func(t *testing.T) {
		ref := resources.PropertyRef{URN: urn, Property: "id"}
		result := printable(ref)
		assert.Contains(t, result, urn)
		assert.NotContains(t, result, "{")
	})
}

// TestPlanReporter_RendersWarnings guards the warning render path: plan-time
// advisories carried on Plan.Warnings are appended after the diff render and
// styled via ui.Warning so they share the existing yellow "Warning:" prefix.
func TestPlanReporter_RendersWarnings(t *testing.T) {
	var buf bytes.Buffer
	r := &planReporter{}
	r.SetWriter(&buf)

	plan := &planner.Plan{
		Diff: &differ.Diff{},
		Warnings: []string{
			"metadata for created_at will remain in the workspace; v1 has no clear/delete path",
			"metadata for email will remain in the workspace; v1 has no clear/delete path",
		},
	}

	r.ReportPlan(plan)

	out := buf.String()
	// Each warning is rendered on its own line, prefixed with "Warning:" so the
	// existing ui.Warning style is reused without coupling the reporter to a
	// specific colour code (lipgloss may emit ANSI escapes depending on the
	// terminal).
	assert.Contains(t, out, "Warning:")
	assert.Contains(t, out, "metadata for created_at will remain in the workspace; v1 has no clear/delete path")
	assert.Contains(t, out, "metadata for email will remain in the workspace; v1 has no clear/delete path")
}

// TestPlanReporter_NoWarningsLeavesOutputUnchanged keeps the existing no-warning
// output stable: plans without advisories must not introduce stray newlines or
// "Warning:" prefixes.
func TestPlanReporter_NoWarningsLeavesOutputUnchanged(t *testing.T) {
	var buf bytes.Buffer
	r := &planReporter{}
	r.SetWriter(&buf)

	r.ReportPlan(&planner.Plan{Diff: &differ.Diff{}})

	assert.NotContains(t, buf.String(), "Warning:")
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
