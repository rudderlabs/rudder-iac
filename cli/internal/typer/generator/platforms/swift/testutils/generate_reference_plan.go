package main

import (
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/swift"
	plantestutils "github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
)

func main() {
	plan := plantestutils.GetReferenceTrackingPlan()
	gen := &swift.Generator{}

	files, err := gen.Generate(plan, core.GenerateOptions{RudderCLIVersion: "1.0.0"}, swift.SwiftOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(files) > 0 {
		fmt.Print(files[0].Content)
	}
}
