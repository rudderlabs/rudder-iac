//go:build ignore

// Writes every Go golden into the validator module (make
// typer-go-update-testdata). It shares its directory with the testutils
// package, hence the ignore constraint: run it with go run on this file.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testutils"
)

func main() {
	root := "typer/generator/platforms/golang"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	for _, g := range testutils.Goldens {
		path := filepath.Join(root, g.Path)
		files, err := (&golang.Generator{}).Generate(g.Plan(), core.GenerateOptions{RudderCLIVersion: "1.0.0"}, golang.GoOptions{PackageName: g.PackageName})
		if err != nil {
			fmt.Fprintf(os.Stderr, "generating %s: %v\n", path, err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "creating the directory of %s: %v\n", path, err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, []byte(files[0].Content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "writing %s: %v\n", path, err)
			os.Exit(1)
		}
	}
}
