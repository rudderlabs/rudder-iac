package source

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/stretchr/testify/assert"
)

func TestValidateRejectsSourceNameBackendCannotAccept(t *testing.T) {
	results := validateSourceSpec("", specs.SpecVersionV1, nil, esSource.SourceSpec{
		LocalID: "src", Name: "bad\nname", SourceDefinition: "javascript",
	})
	assert.NotEmpty(t, results)
}

func TestValidateRejectsEmptyGovernanceConfig(t *testing.T) {
	results := validateSourceSpec("", specs.SpecVersionV1, nil, esSource.SourceSpec{
		LocalID: "src", Name: "Valid Source", SourceDefinition: "javascript",
		Governance: &esSource.SourceGovernanceSpec{TrackingPlan: &esSource.TrackingPlanSpec{
			Ref: "#tracking-plan:tp", Config: &esSource.TrackingPlanConfigSpec{},
		}},
	})
	assert.NotEmpty(t, results)
}
