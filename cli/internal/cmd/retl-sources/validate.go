package retlsource

import (
	"fmt"
	"io"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func newCmdValidate() *cobra.Command {
	var location string

	cmd := &cobra.Command{
		Use:   "validate <external-id>",
		Short: "Validate a RETL source's spec (SQL model or table)",
		Long: heredoc.Doc(`
			Validate a RETL source's spec.

			This checks the project's specs and that the source is defined in it. It
			does not run the source's query: reading from the warehouse is what
			` + "`rudder-cli retl-sources preview`" + ` is for, and it is opt-in because it
			executes a query against live data.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli retl-sources validate my-model
			$ rudder-cli retl-sources validate my-model --location ./project
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("retl-source external id is required")
			}
			externalID := args[0]
			var err error
			defer func() {
				telemetry.TrackCommand("retl-sources validate", err)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			// Load runs the project's syntactic and semantic rules over every spec,
			// so reaching the graph at all means the project validated.
			p := d.NewProject()
			if err := p.Load(location); err != nil {
				return fmt.Errorf("loading project: %w", err)
			}

			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}
			resource, err := findSource(graph, externalID)
			if err != nil {
				return err
			}

			reportValidation(cmd.OutOrStdout(), externalID, resource.Type())
			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the project directory")

	return cmd
}

// reportValidation names the source that validated and where the warehouse check
// went. Saying so matters more than it looks: this command used to run the query,
// so a bare success line would read as "your warehouse is reachable" to anyone
// who used the old behaviour.
func reportValidation(w io.Writer, externalID, resourceType string) {
	fmt.Fprintf(w, "✅ %s '%s' is valid\n", resourceType, externalID)
	fmt.Fprintf(w, "   To check that its query runs against the warehouse: rudder-cli retl-sources preview %s\n", externalID)
}
