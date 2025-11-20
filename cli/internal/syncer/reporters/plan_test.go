package reporters

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestPlanReporter(t *testing.T) {
	r := &planReporter{}

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
				},
			},
		},
		RemovedResources: []string{"removed.resource1"},
	}

	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.ResetWriter()

	r.ReportPlan(&planner.Plan{Diff: diff})

	expectedOutput := `Importable resources:
  - importable.resource1
  - importable.resource2

New resources:
  - new.resource1

Updated resources:
  - updated.resource1
    - name: old-name => new-name
    - size: 10 => 20

Removed resources:
  - removed.resource1

`

	assert.Equal(t, expectedOutput, buf.String())
}
