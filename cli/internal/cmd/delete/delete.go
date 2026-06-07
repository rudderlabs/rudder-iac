package delete

import (
	"context"
	"fmt"
	"io"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

// NewCmdDelete returns the top-level `delete` cobra command.
func NewCmdDelete() *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <type> <id>",
		Short: "Delete a managed resource from the remote workspace",
		Long: `Delete removes a managed (IaC-tracked) resource from the remote workspace by calling
the provider's LifecycleManager. An interactive confirmation prompt is shown unless
--confirm is passed.

Only resources with an external ID (i.e. managed via IaC) can be deleted through
this command. Unmanaged resources are rejected.

Examples:
  # Preview and confirm deletion interactively
  rudder-cli delete event-stream-source my-source

  # Delete without the interactive prompt
  rudder-cli delete event-stream-source my-source --confirm`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("delete", err, []telemetry.KV{
					{K: "type", V: args[0]},
					{K: "confirm", V: confirm},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			router, ok := d.CompositeProvider().(provider.TypeRouter)
			if !ok {
				return fmt.Errorf("internal error: composite provider does not support per-type routing")
			}

			res := resourceops.New(router)
			prov, err := res.ProviderFor(args[0])
			if err != nil {
				return err
			}

			err = RunDelete(cmd.Context(), cmd.OutOrStdout(), prov, args[0], args[1], confirm, ui.Confirm)
			return err
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "Skip the interactive confirmation prompt and proceed with deletion")

	return cmd
}

// RunDelete is the testable core. It previews the deletion, optionally prompts
// for confirmation, then delegates to resourceops.Delete.
func RunDelete(
	ctx context.Context,
	out io.Writer,
	prov provider.Provider,
	resourceType, id string,
	skipConfirm bool,
	confirmFn func(string) (bool, error),
) error {
	fmt.Fprintf(out, "Will delete %s %q from the remote workspace.\n", resourceType, id)

	if !skipConfirm {
		ok, err := confirmFn("Proceed with deletion?")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "aborted")
			return nil
		}
	}

	if err := resourceops.Delete(ctx, prov, resourceType, id); err != nil {
		return err
	}

	fmt.Fprintf(out, "Deleted %s %q successfully.\n", resourceType, id)
	return nil
}
