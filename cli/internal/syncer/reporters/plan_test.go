package reporters

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
)

func TestPlanReporter(t *testing.T) {
	var buf bytes.Buffer

	r := &planReporter{}
	r.SetWriter(&buf)

	var (
		oldRefURN = "data-graph:old-graph"
		newRefURN = "data-graph:new-graph"
	)

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
				},
			},
		},
		RemovedResources: []string{"removed.resource1"},
	}

	r.ReportPlan(&planner.Plan{Diff: diff})

	expectedOutput := "Importable resources:\n" +
		"  - importable.resource1\n" +
		"  - importable.resource2\n" +
		"\n" +
		"New resources:\n" +
		"  - new.resource1\n" +
		"\n" +
		"Updated resources:\n" +
		"  - updated.resource1\n" +
		"    - name: old-name => new-name\n" +
		"    - ref_ptr_changed: " + oldRefURN + " => " + newRefURN + "\n" +
		"    - ref_ptr_nil_source: <nil> => " + newRefURN + "\n" +
		"    - ref_ptr_nil_target: " + oldRefURN + " => <nil>\n" +
		"    - ref_val_changed: " + oldRefURN + " => " + newRefURN + "\n" +
		"    - size: 10 => 20\n" +
		"\n" +
		"Removed resources:\n" +
		"  - removed.resource1\n" +
		"\n"

	assert.Equal(t, expectedOutput, buf.String())
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
