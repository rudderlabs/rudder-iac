package rules

import (
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/spf13/cobra"
)

var rulesLog = logger.New("docs", logger.Attr{
	Key:   "cmd",
	Value: "rules",
})

func NewCmdRules() *cobra.Command {
	var (
		deps      app.Deps
		err       error
		outputDir string
		format    string
	)

	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Generate the validation rule documentation catalog",
		Long: heredoc.Doc(`
			Generates the validation rule documentation catalog by enumerating the
			rules registered across all providers, enriching them with authored
			documentation fragments, and writing the flat canonical rules.yaml and
			rules.json artifacts.

			The catalog is always a flat list of rules. Use --format to control
			which artifacts are written: 'yaml' (Hugo/humans), 'json' (LLMs/tooling),
			or 'both' (default). Presentation grouping for Hugo is handled by a
			downstream translation layer, not this command.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli docs rules --output-dir ./docs/generated/
			$ rudder-cli docs rules --format json
			$ rudder-cli docs rules --format yaml
		`),
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormat(format); err != nil {
				return err
			}

			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := deps.Registry()
			if err != nil {
				return fmt.Errorf("building rule registry: %w", err)
			}

			cp := deps.CompositeProvider()
			entries := cp.RuleDocEntries()

			doc, verrs := docs.Generate(
				reg.AllSyntacticRules(),
				reg.AllSemanticRules(),
				entries,
				app.GetVersion(),
				time.Now().UTC().Format(time.RFC3339),
			)

			if len(verrs) > 0 {
				for _, verr := range verrs {
					fmt.Fprintln(cmd.ErrOrStderr(), verr)
				}
				return fmt.Errorf("rule docs validation failed: %d error(s)", len(verrs))
			}

			if err := docs.Serialize(doc, outputDir, format); err != nil {
				return fmt.Errorf("serializing rule docs: %w", err)
			}

			rulesLog.Info("Rule documentation generated", "outputDir", outputDir, "format", format)
			fmt.Fprintf(cmd.OutOrStdout(), "Rule documentation written to %s\n", outputDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&outputDir, "output-dir", "./docs/generated/", "Directory to write the generated rules.yaml and rules.json artifacts")
	cmd.Flags().StringVar(&format, "format", docs.FormatBoth, "Artifacts to write: 'yaml', 'json', or 'both' (default)")

	return cmd
}

// validateFormat rejects unsupported --format values.
func validateFormat(format string) error {
	switch format {
	case docs.FormatYAML, docs.FormatJSON, docs.FormatBoth:
		return nil
	default:
		return fmt.Errorf("invalid --format %q: must be %q, %q, or %q", format, docs.FormatYAML, docs.FormatJSON, docs.FormatBoth)
	}
}
