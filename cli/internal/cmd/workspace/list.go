package workspace

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/workspace"
	"github.com/spf13/cobra"
)

func NewCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List resources in the workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListAccounts())

	return cmd
}

func newCmdListAccounts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "List accounts in the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")
			accountType, _ := cmd.Flags().GetString("type")

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			p := d.CompositeProvider()
			l := lister.New(p)

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

	return cmd
}
