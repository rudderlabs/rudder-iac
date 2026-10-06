package testutils

import (
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	plantestutils "github.com/rudderlabs/rudder-iac/typer/plan/testutils"
)

// Golden is a plan whose generated output is committed in the validator
// module (testdata/validator), each golden in its own package.
type Golden struct {
	Plan    func() *plan.TrackingPlan
	Options golang.GoOptions
	// Path is relative to the Go platform's package directory.
	Path string
}

// Goldens is the table both the golden test and generate_reference_plan.go
// (make typer-go-update-testdata) run.
var Goldens = []Golden{
	{
		Plan:    plantestutils.GetReferenceTrackingPlan,
		Options: golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "ruddertyper.go"},
		Path:    "testdata/validator/ruddertyper/ruddertyper.go",
	},
	{
		Plan:    GetExamplesTrackingPlan,
		Options: golang.GoOptions{PackageName: "examples", OutputFileName: "ruddertyper.go"},
		Path:    "testdata/validator/examples/ruddertyper.go",
	},
}
