// Command gen-cmd-docs regenerates the Markdown, YAML, and man page command reference.
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
	outputDir := flag.String("output-dir", "docs/generated", "Directory containing generated command references")
	manDir := flag.String("man-dir", "man", "Directory containing generated man pages")
	flag.Parse()

	root := cmd.NewRootCommand(cmd.ModeDocs)
	cmd.PrepareDocsTree(root)
	if err := cmddocs.Generate(root, *outputDir, *manDir); err != nil {
		return err
	}

	fmt.Printf("Command documentation written to %s and %s\n", *outputDir, *manDir)
	return nil
}
