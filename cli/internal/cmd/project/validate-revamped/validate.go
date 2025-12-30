package validaterevamped

import (
	"errors"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/engine"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/registry"
	"github.com/spf13/cobra"
)

var (
	validateLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "validate-revamped",
	})
)

// ErrValidationFailed is returned when validation finds errors
var ErrValidationFailed = errors.New("validation failed")

// NewCmdValidateRevamped creates the validate-revamped command
func NewCmdValidateRevamped() *cobra.Command {
	var (
		deps     app.Deps
		location string
		genDocs  bool
		output   string
		err      error
	)

	cmd := &cobra.Command{
		Use:   "validate-revamped",
		Short: "Validate project configuration with rich diagnostics",
		Long: heredoc.Doc(`
			Validates the project configuration files using the validation engine
			and displays detailed, Rust-style diagnostic output.

			This command provides:
			- Precise line and column information
			- Code fragments showing the problematic content
			- Squiggly underlines highlighting the issue
			- Summary of all issues found

			Additionally, you can generate markdown documentation for all
			validation rules using the --gen-docs flag.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli validate-revamped --location /path/to/project
			$ rudder-cli validate-revamped -l ./my-catalog
			$ rudder-cli validate-revamped --gen-docs --output ./docs/validation-rules.md
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			validateLog.Debug("validate-revamped", "location", location, "genDocs", genDocs)

			defer func() {
				telemetry.TrackCommand("validate-revamped", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "genDocs", V: genDocs},
				}...)
			}()

			// Create the default registry with all rules
			reg, regErr := registry.NewDefaultRegistry()
			if regErr != nil {
				err = fmt.Errorf("creating validation registry: %w", regErr)
				return err
			}

			// Handle documentation generation mode
			if genDocs {
				return runGenDocs(reg, output)
			}

			// Create validation engine
			eng, engErr := engine.NewEngine(
				location,
				reg,
				deps.CompositeProvider(),
			)
			if engErr != nil {
				err = fmt.Errorf("creating validation engine: %w", engErr)
				return err
			}

			// Run validation
			diagnostics := eng.Validate()

			// Render diagnostics
			renderer := ui.NewDiagnosticsRenderer(diagnostics)

			// Print all diagnostics
			rendered := renderer.Render()
			if rendered != "" {
				fmt.Println(rendered)
				fmt.Println() // Blank line before summary
			}

			// Print summary
			fmt.Println(renderer.Summary())

			// Return error if any errors were found (Cobra handles exit code)
			if renderer.HasErrors() {
				err = ErrValidationFailed
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".",
		"Path to the directory containing the project files")
	cmd.Flags().BoolVar(&genDocs, "gen-docs", false,
		"Generate markdown documentation for all validation rules")
	cmd.Flags().StringVarP(&output, "output", "o", "",
		"Output file path for generated documentation (default: stdout)")

	return cmd
}

// runGenDocs generates markdown documentation for all validation rules
func runGenDocs(reg *registry.RuleRegistry, outputPath string) error {
	generator := ui.NewRuleDocGenerator(reg)
	markdown := generator.Generate()

	if outputPath == "" {
		// Print to stdout
		fmt.Println(markdown)
		return nil
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("writing documentation to %s: %w", outputPath, err)
	}

	fmt.Printf("Documentation generated successfully at: %s\n", outputPath)
	return nil
}
