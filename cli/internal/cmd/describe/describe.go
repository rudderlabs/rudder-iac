package describe

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
	"gopkg.in/yaml.v3"
)

// describeRouter is the minimal seam the describe command needs from the composite
// provider: per-type routing plus the full list of registered types for validation.
type describeRouter interface {
	provider.TypeRouter
	SupportedTypes() []string
}

// NewCmdDescribe returns the top-level `describe` cobra command.
func NewCmdDescribe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <type> <id>",
		Short: "Describe a resource as a human-readable spec layout",
		Long: `Describe renders a single managed remote resource as a human-readable layout.

The output includes the re-appliable spec fields and a Managed line indicating
whether the resource is currently tracked by this tool.

Examples:
  # Describe a managed event-stream source
  rudder-cli describe event-stream-source my-source-id`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("describe", err, []telemetry.KV{
					{K: "type", V: args[0]},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			router, ok := d.CompositeProvider().(describeRouter)
			if !ok {
				return fmt.Errorf("internal error: composite provider does not support per-type routing")
			}

			if err = resourceops.ValidateType(router.SupportedTypes(), args[0]); err != nil {
				return err
			}

			err = RunDescribe(cmd.Context(), cmd.OutOrStdout(), router, args[0], args[1])
			return err
		},
	}
	return cmd
}

// RunDescribe is the testable core. It materializes a single resource spec,
// determines its managed status, and writes a human-readable layout to out.
func RunDescribe(ctx context.Context, out io.Writer, router provider.TypeRouter, resourceType, id string) error {
	res := resourceops.New(router)

	prov, err := res.ProviderFor(resourceType)
	if err != nil {
		return err
	}

	// Single remote load: SpecYAMLWithManaged returns both the YAML and managed
	// flag without an extra FindRemote round-trip.
	yamlStr, managed, err := resourceops.SpecYAMLWithManaged(ctx, prov, resourceType, id)
	if err != nil {
		return err
	}

	var specMap map[string]any
	if err := yaml.Unmarshal([]byte(yamlStr), &specMap); err != nil {
		return fmt.Errorf("decoding spec YAML: %w", err)
	}

	managedStr := "no"
	if managed {
		managedStr = "yes"
	}

	header := fmt.Sprintf("--- %s / %s\n", resourceType, id)
	managedLine := fmt.Sprintf("Managed: %s\n\n", managedStr)
	body := ui.FormattedMap(specMap)

	_, err = fmt.Fprint(out, header+managedLine+body)
	return err
}
