package workspace

import (
	"github.com/MakeNowJust/heredoc/v2"
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
	cmd.AddCommand(newCmdViewAccount())

	return cmd
}

func newCmdViewAccount() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "view <external-id>",
		Short: "Show one account in full",
		Long: heredoc.Doc(`
			Shows an account's definition and its non-secret options.

			Secret values are never shown. The API does not return them, and
			printing a placeholder where a password would be only suggests the
			CLI could read it.
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("workspace accounts view", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			row, err := findByExternalID(cmd.Context(), d.Providers().Workspace, workspace.AccountResourceType, args[0], "account")
			if err != nil {
				return err
			}

			err = lister.New(oneResource(row), lister.WithFormat(viewFormat(jsonOutput))).
				List(cmd.Context(), workspace.AccountResourceType, nil)
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
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
