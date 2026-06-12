package reporters

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
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

func TestPrintableSecret(t *testing.T) {
	const real = "sk_live_abcd1234"
	const masked = "****1234"

	t.Run("value secret is masked", func(t *testing.T) {
		result := printable(secret.New(real))
		assert.Contains(t, result, masked)
		assert.NotContains(t, result, real)
	})

	t.Run("pointer secret is masked", func(t *testing.T) {
		s := secret.New(real)
		result := printable(&s)
		assert.Contains(t, result, masked)
		assert.NotContains(t, result, real)
	})

	t.Run("unknown secret renders placeholder", func(t *testing.T) {
		assert.Contains(t, printable(secret.NewUnknown()), "(unknown)")
	})

	t.Run("nil pointer secret renders nil", func(t *testing.T) {
		assert.Contains(t, printable((*secret.String)(nil)), "<nil>")
	})
}

// TestRenderPropertyDiff_Secret proves a secret diff renders as "always re-applied"
// (no meaningful old => new) and never leaks the real value.
func TestRenderPropertyDiff_Secret(t *testing.T) {
	const real = "sk_live_old_1111"

	t.Run("flat secret diff renders the always-re-applied annotation", func(t *testing.T) {
		lines := renderPropertyDiff(differ.PropertyDiff{
			Property:    "token",
			SourceValue: secret.New(real),
			TargetValue: secret.NewUnknown(),
			SecretOnly:  true,
		})
		out := strings.Join(lines, "")
		assert.Contains(t, out, "token")
		assert.Contains(t, out, "(secret, always re-applied)")
		assert.NotContains(t, out, "=>")
		assert.NotContains(t, out, real)
	})

	// When a sibling field changes, the property diff is not secret-only, so the
	// nested renderer renders the real change normally and the secret leaf through
	// the generic path, where printable masks it. The "(secret, always re-applied)"
	// line is reserved for secret-only property diffs; a secret leaked here must
	// still never show its real value.
	t.Run("nested diff masks a secret leaf without leaking the value", func(t *testing.T) {
		enableNestedDiffs(t)
		localSecret, unknown := secret.New(real), secret.NewUnknown()
		lines := renderPropertyDiff(differ.PropertyDiff{
			Property:    "config",
			SourceValue: map[string]any{"token": &localSecret, "name": "a"},
			TargetValue: map[string]any{"token": &unknown, "name": "b"},
		})
		out := strings.Join(lines, "")
		assert.Contains(t, out, "config.name")
		assert.Contains(t, out, "config.token")
		assert.Contains(t, out, "(unknown)")
		assert.NotContains(t, out, real)
	})
}

// TestRenderDiff_SecretSections proves secret-only resources are listed in their own
// "Always re-applied" section, separate from real updates, with no value leak.
func TestRenderDiff_SecretSections(t *testing.T) {
	diff := &differ.Diff{
		UpdatedResources: map[string]differ.ResourceDiff{
			"type:real": {URN: "type:real", Diffs: map[string]differ.PropertyDiff{
				"name": {Property: "name", SourceValue: "a", TargetValue: "b"},
			}},
			"type:secret": {URN: "type:secret", SecretOnly: true, Diffs: map[string]differ.PropertyDiff{
				"token": {Property: "token", SourceValue: secret.New("sk_live_x"), TargetValue: secret.NewUnknown(), SecretOnly: true},
			}},
		},
	}

	out := renderDiff(diff)
	assert.Contains(t, out, "Updated resources")
	assert.Contains(t, out, "type:real")
	assert.Contains(t, out, "Always re-applied (secret values can't be read back)")
	assert.Contains(t, out, "type:secret")
	assert.Contains(t, out, "(secret, always re-applied)")
	assert.NotContains(t, out, "sk_live_x")
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
