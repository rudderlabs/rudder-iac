package docs

import (
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newCmdExportValidationRules() *cobra.Command {
	return &cobra.Command{
		Use:   "export-validation-rules",
		Short: "Print the validation rule catalog as YAML",
		Long: heredoc.Doc(`
			Prints the catalog of every validation rule rudder-cli enforces — rule ID,
			phase, severity, description, the kinds it applies to, and worked valid and
			invalid examples.

			This is the same artifact CI generates as docs/generated/rules.yaml, built
			from the live rule registry, so it can never drift from what validate
			actually enforces. It is the authoritative reference for the rule IDs
			reported by 'rudder-cli validate'.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli docs export-validation-rules > rules.yaml
		`),
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, verrs, err := app.GenerateRuleCatalog(time.Now().UTC().Format(time.RFC3339))
			if err != nil {
				return fmt.Errorf("generating rule catalog: %w", err)
			}

			// A rule missing its authored doc fragment is a build-time defect caught
			// by the gen-rule-docs CI gate. Reaching a released binary means the gate
			// was bypassed; report it rather than handing a consumer a silently
			// incomplete catalog.
			if len(verrs) > 0 {
				for _, verr := range verrs {
					fmt.Fprintln(cmd.ErrOrStderr(), verr)
				}
				return fmt.Errorf("rule catalog is incomplete: %d error(s)", len(verrs))
			}

			out, err := yaml.Marshal(doc)
			if err != nil {
				return fmt.Errorf("marshaling rule catalog: %w", err)
			}

			_, err = cmd.OutOrStdout().Write(out)
			return err
		},
	}
}
