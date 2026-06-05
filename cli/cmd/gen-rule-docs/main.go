// Command gen-rule-docs regenerates the canonical validation rule documentation
// catalog (docs/generated/rules.yaml).
//
// It is intentionally a standalone dev/CI tool rather than a public rudder-cli
// subcommand: generating the catalog is a release-time concern, not something an
// end user runs. It enumerates the live validation rules, joins them with the
// authored *.docs.yaml fragments contributed by each provider, validates the
// result, and serializes it. No network calls are made, so unlike the regular
// CLI it does not require authentication.
//
// Invoke via `make gen-rule-docs` (or directly with `go run ./cli/cmd/gen-rule-docs`).
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
)

// version is overridden at build time; mirrors cli/cmd/rudder-cli.
var version = "0.0.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	outputDir := flag.String("output-dir", "./docs/generated/", "Directory to write the generated rules.yaml artifact")
	strict := flag.Bool("strict-verify", false, "Fail if the engine produces diagnostics beyond those documented (exact-match)")
	flag.Parse()

	app.Initialise(version)
	config.InitConfig(config.DefaultConfigFile())

	// All composition lives in app.GenerateRuleCatalog so the documented rule
	// set is built from the same providers and registry project validation
	// uses; this command only chooses the timestamp and writes the result.
	doc, verrs, err := app.GenerateRuleCatalog(time.Now().UTC().Format(time.RFC3339), *strict)
	if err != nil {
		return err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			fmt.Fprintln(os.Stderr, verr)
		}
		return fmt.Errorf("rule docs validation failed: %d error(s)", len(verrs))
	}

	if err := docs.Serialize(doc, *outputDir); err != nil {
		return fmt.Errorf("serializing rule docs: %w", err)
	}

	fmt.Printf("Rule documentation written to %s\n", *outputDir)
	return nil
}
