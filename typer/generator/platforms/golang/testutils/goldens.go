package testutils

import (
	"github.com/rudderlabs/rudder-iac/typer/plan"
	plantestutils "github.com/rudderlabs/rudder-iac/typer/plan/testutils"
)

// Golden is a plan whose generated output is committed under
// testdata/validator, each golden in its own package directory.
type Golden struct {
	Plan        func() *plan.TrackingPlan
	PackageName string
	// Path is relative to the Go platform's package directory.
	Path string
}

// Goldens is the table both the golden test and generate_reference_plan.go
// (make typer-go-update-testdata) run.
var Goldens = []Golden{
	{
		Plan:        plantestutils.GetReferenceTrackingPlan,
		PackageName: "ruddertyper",
		Path:        "testdata/validator/ruddertyper/ruddertyper.go",
	},
	{
		Plan:        GetExamplesTrackingPlan,
		PackageName: "examples",
		Path:        "testdata/validator/examples/ruddertyper.go",
	},
}
