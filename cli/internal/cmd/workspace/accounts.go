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

	// Both browser-handoff commands are hidden until the experimental flag is
	// on, like the sources group: they cannot work against production yet.
	oauthAccountCmds = []*cobra.Command{newCmdCreateAccount(), newCmdReauthorizeAccount()}
	for _, c := range oauthAccountCmds {
		cmd.AddCommand(c)
	}

	return cmd
}

// oauthAccountCmds is the commands UnhideOAuthAccounts shows. Config is read
// after the tree is built, so the flag cannot be checked where they are made.
var oauthAccountCmds []*cobra.Command

// UnhideOAuthAccounts lists the browser-handoff commands in --help. Called from
// the root command once the experimental flag has been read.
func UnhideOAuthAccounts() {
	for _, cmd := range oauthAccountCmds {
		cmd.Hidden = false
	}
}

func newCmdListAccounts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List accounts in the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			category, _ := cmd.Flags().GetString("category")
			accountType, _ := cmd.Flags().GetString("type")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
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

			return l.List(cmd.Context(), workspace.AccountResourceType, filters)
		},
	}

	cmd.Flags().String("category", "", "Filter by account category")
	cmd.Flags().String("type", "", "Filter by account type")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
