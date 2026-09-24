package main

import (
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/typescript"
	"github.com/rudderlabs/rudder-iac/typer/plan/testutils"
)

func main() {
	gen := &typescript.Generator{}
	files, err := gen.Generate(
		testutils.GetIdentitySectionsPlan(),
		core.GenerateOptions{RudderCLIVersion: "1.0.0"},
		typescript.TypeScriptOptions{OutputFileName: "IdentitySections.ts"},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(files) > 0 {
		fmt.Print(files[0].Content)
	}
}
