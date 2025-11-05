package workspace

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/workspace"
	"github.com/spf13/cobra"
)

func NewCmdAccounts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Manage accounts in the workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListAccounts())

	return cmd
}

func newCmdListAccounts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List accounts in the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")
			accountType, _ := cmd.Flags().GetString("type")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			var err error
			defer func() {
				telemetry.TrackCommand("workspace accounts list", err, []telemetry.KV{
					{K: "category", V: category},
					{K: "type", V: accountType},
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			p := d.Providers().Workspace
			format := lister.TableFormat
			if jsonOutput {
				format = lister.JSONFormat
			}
			l := lister.New(p, lister.WithFormat(format))

			filters := make(lister.Filters)
			if category != "" {
				filters["category"] = category
			}
			if accountType != "" {
				filters["type"] = accountType
			}

			err = l.List(cmd.Context(), workspace.AccountResourceType, filters)
			return err
		},
	}

	cmd.Flags().String("category", "", "Filter by account category")
	cmd.Flags().String("type", "", "Filter by account type")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
