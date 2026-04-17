package main

import (
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/swift"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func main() {
	// Keep generator warnings off stdout so the file redirect stays clean.
	ui.SetWriter(os.Stderr)

	trackingPlan := testutils.GetReferenceTrackingPlan()
	gen := &swift.Generator{}

	files, err := gen.Generate(trackingPlan, core.GenerateOptions{RudderCLIVersion: "1.0.0"}, swift.SwiftOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(files) > 0 {
		fmt.Print(files[0].Content)
	}
}
