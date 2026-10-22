package docs

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/specdocs"
)

func newCmdSource() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "List the event stream source types and their connection tokens",
		Long: heredoc.Doc(`
			Lists the values an event stream source spec's type accepts and the token each one
			connects to a destination as. Pass that type to docs destination --source to see
			what a destination accepts from such a source.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli docs source --json
		`),
		// RunE checks the arguments, so that an extra one fails with the JSON
		// document --json asks for, not with cobra's text.
		Args: cobra.ArbitraryArgs,
	}
	resolveFormat := formatFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		format, formatErr := resolveFormat()
		defer func() {
			telemetry.TrackCommand(telemetry.CommandName(cmd), err, telemetry.KV{K: "format", V: reportedFormat(format)})
		}()

		var (
			out        = cmd.OutOrStdout()
			cliVersion = cmd.Root().Version
		)
		switch {
		case formatErr != nil:
			return fail(out, format, cliVersion, formatErr)
		case len(args) > 0:
			return fail(out, format, cliVersion,
				invalidArguments(fmt.Sprintf("docs source takes no arguments, got %d", len(args))))
		}

		list := specdocs.ListSources(cliVersion)
		if format == formatJSON {
			return specdocs.WriteJSON(out, list)
		}
		return specdocs.RenderSourceListMarkdown(out, list)
	}
	return cmd
}
