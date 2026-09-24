package main

import (
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/typescript"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
)

func main() {
	gen := &typescript.Generator{}
	files, err := gen.Generate(
		testutils.GetEmptyIdentityPlan(),
		core.GenerateOptions{RudderCLIVersion: "1.0.0"},
		typescript.TypeScriptOptions{OutputFileName: "EmptyIdentity.ts"},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(files) > 0 {
		fmt.Print(files[0].Content)
	}
}
