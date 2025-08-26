package workspace

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/previewer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/spf13/cobra"
)

func NewCmdRetlSource() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retl-sources",
		Short: "Manage RETL sources in the workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListRetlSources())
	cmd.AddCommand(newCmdPreviewRetlSource())

	return cmd
}

func newCmdListRetlSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List RETL sources in the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")

			var err error
			defer func() {
				telemetry.TrackCommand("workspace retl-source list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			// Cast the RETL provider to access the List method
			retlProvider, ok := d.Providers().RETL.(*retl.Provider)
			if !ok {
				return fmt.Errorf("failed to cast RETL provider")
			}

			format := lister.TableFormat
			if jsonOutput {
				format = lister.JSONFormat
			}
			l := lister.New(retlProvider, format)

			err = l.List(cmd.Context(), sqlmodel.ResourceType, nil)
			return err
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func newCmdPreviewRetlSource() *cobra.Command {
	var localID string
	var location string
	var limit int
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview a RETL source SQL model",
		Long:  "Preview a RETL source SQL model to see the data structure and sample rows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			localID, _ := cmd.Flags().GetString("local-id")
			location, _ := cmd.Flags().GetString("location")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			limit, _ := cmd.Flags().GetInt("limit")

			var err error
			defer func() {
				telemetry.TrackCommand("workspace retl-source preview", err, []telemetry.KV{
					{K: "local_id", V: localID},
					{K: "location", V: location},
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			p := project.New(location, d.CompositeProvider())
			if err := p.Load(); err != nil {
				return fmt.Errorf("loading project: %w", err)
			}

			graph, err := p.GetResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}
			resource, ok := graph.GetResource(sqlmodel.ResourceType + ":" + localID)
			if !ok {
				return fmt.Errorf("resource with local ID '%s' not found in project", localID)
			}
			resourceData := resource.Data()
			resourceType := resource.Type()

			// Get the RETL provider
			retlProvider, ok := d.Providers().RETL.(*retl.Provider)
			if !ok {
				return fmt.Errorf("failed to cast RETL provider")
			}
			opts := []previewer.PreviewerOpts{}
			if jsonOutput {
				opts = append(opts, previewer.WithJson(true))
			}
			if limit > 0 {
				opts = append(opts, previewer.WithLimit(limit))
			}
			previewer := previewer.New(retlProvider, opts...)

			return previewer.Preview(cmd.Context(), localID, resourceType, resourceData)
		},
	}

	cmd.Flags().StringVarP(&localID, "local-id", "i", "", "Local ID of the SQL model to preview")
	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the project directory")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output preview rows as JSON")
	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Number of rows to preview")
	cmd.MarkFlagRequired("local-id")

	return cmd
}
