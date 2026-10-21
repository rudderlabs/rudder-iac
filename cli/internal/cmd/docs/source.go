package docs

import (
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
		Args: cobra.NoArgs,
	}
	resolveFormat := formatFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, _ []string) (err error) {
		format, err := resolveFormat()
		defer func() {
			telemetry.TrackCommand(telemetry.CommandName(cmd), err, telemetry.KV{K: "format", V: reportedFormat(format)})
		}()
		if err != nil {
			return err
		}

		list := specdocs.ListSources(cmd.Root().Version)
		if format == formatJSON {
			return specdocs.WriteJSON(cmd.OutOrStdout(), list)
		}
		return specdocs.RenderSourceListMarkdown(cmd.OutOrStdout(), list)
	}
	return cmd
}
