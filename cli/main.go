package main

import (
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
)

func main() {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	files, err := kotlin.Generate(trackingPlan, core.GenerationOptions{
		RudderCLIVersion: "1.0.0",
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(files) > 0 {
		fmt.Println(files[0].Content)
	}
}
