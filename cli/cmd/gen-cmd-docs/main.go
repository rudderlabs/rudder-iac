// Command gen-cmd-docs generates the public rudder-cli command reference.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmddocs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	outputDir := flag.String("output-dir", ".", "Directory in which to write commands/ and man/")
	flag.Parse()

	if err := cmddocs.Generate(cmd.NewDocumentationRootCmd(), *outputDir); err != nil {
		return fmt.Errorf("generating command documentation: %w", err)
	}

	fmt.Printf("Command documentation written below %s\n", *outputDir)
	return nil
}
