package main

import (
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func main() {
	// Keep generator warnings off stdout so the file redirect stays clean.
	ui.SetWriter(os.Stderr)

	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(files) > 0 {
		fmt.Print(files[0].Content)
	}
}
