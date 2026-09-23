package workspace

import (
	"context"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/spf13/cobra"
)

// sourceKinds are the rETL source resource types this command lists, in the
// order they are printed. A kind whose handler is not registered is skipped, so
// the listed set follows the experimental flags rather than a fixed type.
var sourceKinds = []string{sqlmodel.ResourceType, table.ResourceType}

// allKinds lists several resource types as one set, tagging every row with the
// kind it came from. Without the tag a merged table cannot say which rows are
// SQL models and which are tables.
type allKinds struct {
	provider lister.ListProvider
	kinds    []string
}

func (a allKinds) List(ctx context.Context, _ string, filters lister.Filters) ([]resources.ResourceData, error) {
	var all []resources.ResourceData
	for _, kind := range a.kinds {
		rs, err := a.provider.List(ctx, kind, filters)
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			r["kind"] = kind
			all = append(all, r)
		}
	}
	return all, nil
}

func NewCmdRetlSource() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retl-sources",
		Short: "Manage RETL sources in the workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListRetlSources())
	cmd.AddCommand(newCmdViewRetlSource())

	return cmd
}

func newCmdViewRetlSource() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "view <external-id>",
		Short: "Show one RETL source in full",
		Long:  "Shows what a source actually reads — its query or table, its primary key and the account it reads through — which is the part a list cannot show.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("workspace retl-sources view", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			retlProvider := d.Providers().RETL
			registered := retlProvider.SupportedTypes()
			kinds := make([]string, 0, len(sourceKinds))
			for _, kind := range sourceKinds {
				if slices.Contains(registered, kind) {
					kinds = append(kinds, kind)
				}
			}

			row, err := findByExternalID(cmd.Context(), allKinds{provider: retlProvider, kinds: kinds}, "retl-sources", args[0], "rETL source")
			if err != nil {
				return err
			}

			err = lister.New(oneResource(row), lister.WithFormat(viewFormat(jsonOutput))).
				List(cmd.Context(), "retl-source", nil)
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
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
				telemetry.TrackCommand("workspace retl-sources list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			retlProvider := d.Providers().RETL

			registered := retlProvider.SupportedTypes()
			kinds := make([]string, 0, len(sourceKinds))
			for _, kind := range sourceKinds {
				if slices.Contains(registered, kind) {
					kinds = append(kinds, kind)
				}
			}

			format := lister.TableFormat
			if jsonOutput {
				format = lister.JSONFormat
			}
			l := lister.New(allKinds{provider: retlProvider, kinds: kinds}, lister.WithFormat(format))

			err = l.List(cmd.Context(), "retl-sources", nil)
			return err
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
