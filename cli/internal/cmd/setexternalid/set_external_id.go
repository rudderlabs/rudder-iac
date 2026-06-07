package setexternalid

import (
	"context"
	"fmt"
	"io"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
	"github.com/spf13/cobra"
)

// setExternalIDRouter is the minimal seam the set-external-id command needs from
// the composite provider: per-type routing plus the full list of registered types
// for validation.
type setExternalIDRouter interface {
	provider.TypeRouter
	SupportedTypes() []string
}

// NewCmdSetExternalID returns the top-level `set-external-id` cobra command.
func NewCmdSetExternalID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-external-id <type> <remote-id> <external-id>",
		Short: "Associate an existing remote resource with a local external id",
		Long: `set-external-id associates an existing remote resource (identified by its remote ID)
with a local external ID, making it manageable by this tool.

Examples:
  # Claim an event-stream source
  rudder-cli set-external-id event-stream-source src_remote_1 my-source`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("set-external-id", err, []telemetry.KV{
					{K: "type", V: args[0]},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			router, ok := d.CompositeProvider().(setExternalIDRouter)
			if !ok {
				return fmt.Errorf("internal error: composite provider does not support per-type routing")
			}

			if err = resourceops.ValidateType(router.SupportedTypes(), args[0]); err != nil {
				return err
			}

			err = RunSetExternalID(cmd.Context(), cmd.OutOrStdout(), router, args[0], args[1], args[2])
			return err
		},
	}
	return cmd
}

// RunSetExternalID is the testable core. It asserts the ExternalIDSetter capability
// on the provider for resourceType, then sets the external ID on the remote resource.
func RunSetExternalID(ctx context.Context, out io.Writer, router provider.TypeRouter, resourceType, remoteID, externalID string) error {
	res := resourceops.New(router)

	setter, err := res.ExternalIDSetterFor(resourceType)
	if err != nil {
		return err
	}

	if err = setter.SetExternalID(ctx, resourceType, remoteID, externalID); err != nil {
		return fmt.Errorf("setting external id: %w", err)
	}

	fmt.Fprintf(out, "set external id %q for %s %q\n", externalID, resourceType, remoteID)
	return nil
}
