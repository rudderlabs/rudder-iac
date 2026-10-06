//go:build ignore

// Writes every Go golden into the validator module (make
// typer-go-update-testdata). It shares its directory with the testutils
// package, hence the ignore constraint: run it with go run on this file.
package main

import (
	"log"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testutils"
)

func main() {
	fm := core.NewFileManager("typer/generator/platforms/golang")
	for _, g := range testutils.Goldens {
		files, err := (&golang.Generator{}).Generate(g.Plan(), core.GenerateOptions{RudderCLIVersion: "1.0.0"}, g.Options)
		if err != nil {
			log.Fatalf("generating %s: %v", g.Path, err)
		}
		if err := fm.WriteFile(&core.File{Path: g.Path, Content: files[0].Content}); err != nil {
			log.Fatalf("writing %s: %v", g.Path, err)
		}
	}
}
