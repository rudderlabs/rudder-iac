package workspace

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/spf13/cobra"
)

func NewCmdEventStreamSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event-stream-sources",
		Short: "Manage event stream sources in the workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListEventStreamSources())

	return cmd
}

func newCmdListEventStreamSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List event stream sources in the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")

			var err error
			defer func() {
				telemetry.TrackCommand("workspace event-stream-sources list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			format := lister.TableFormat
			if jsonOutput {
				format = lister.JSONFormat
			}

			l := lister.New(d.Providers().EventStream, lister.WithFormat(format))
			err = l.List(cmd.Context(), source.ResourceType, nil)
			return err
		},
	}

	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
