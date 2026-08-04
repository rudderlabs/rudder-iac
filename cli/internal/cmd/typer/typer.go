package typer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/providers"
	"github.com/spf13/cobra"
)

const (
	platformKotlin     = "kotlin"
	platformSwift      = "swift"
	platformTypeScript = "typescript"
)

func NewCmdTyper() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "typer",
		Short: "Generate type-safe tracking code",
		Long:  "Generate type-safe tracking code from RudderStack tracking plans",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdGenerate())
	cmd.AddCommand(newCmdOptions())

	return cmd
}

func newCmdGenerate() *cobra.Command {
	var trackingPlanID string
	var platform string
	var outputDir string
	var options []string
	var local bool
	var location string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate type-safe code from tracking plan",
		Long:  "Generate type-safe code from a RudderStack tracking plan",
		Example: heredoc.Doc(`
			$ rudder-cli typer generate --tracking-plan-id <id> --platform kotlin
			$ rudder-cli typer generate --local --location ./project --platform kotlin
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			validPlatforms := map[string]bool{platformKotlin: true, platformSwift: true, platformTypeScript: true}
			if !validPlatforms[platform] {
				supported := make([]string, 0, len(validPlatforms))
				for p := range validPlatforms {
					supported = append(supported, p)
				}
				sort.Strings(supported)
				return fmt.Errorf("unsupported platform: %s (supported platforms: %s)", platform, strings.Join(supported, ", "))
			}

			if local && !config.GetConfig().ExperimentalFlags.LocalTyper {
				return fmt.Errorf("--local is experimental; enable it by setting both RUDDERSTACK_CLI_EXPERIMENTAL=true (the umbrella experimental flag) and RUDDERSTACK_X_LOCAL_TYPER=true (the 'localTyper' flag). The per-flag setting is ignored unless the umbrella flag is also set")
			}

			defer func() {
				telemetry.TrackCommand("typer", nil, []telemetry.KV{
					{K: "platform", V: platform},
					{K: "local", V: local},
				}...)
			}()

			var (
				planProvider typer.PlanProvider
				err          error
			)
			if local {
				planProvider, err = providers.NewLocalCatalogPlanProviderForProject(location, trackingPlanID)
			} else {
				planProvider, err = providers.NewRemoteCatalogPlanProvider(trackingPlanID)
			}
			if err != nil {
				return err
			}

			rudderTyper := typer.NewRudderTyper(planProvider)

			// Parse platform-specific options from key=value pairs
			platformOptions := parsePlatformOptions(options)

			genOptions := core.GenerateOptions{
				RudderCLIVersion: app.GetVersion(),
				Platform:         platform,
				OutputPath:       outputDir,
				PlatformOptions:  platformOptions,
			}

			return rudderTyper.Generate(context.Background(), genOptions)
		},
	}

	cmd.Flags().StringVar(&trackingPlanID, "tracking-plan-id", "", "Tracking plan ID to generate code from (remote), or local id of the plan in the specs (with --local)")

	cmd.Flags().StringVar(&platform, "platform", platformKotlin, fmt.Sprintf("Platform to generate code for (%s, %s, %s)", platformKotlin, platformSwift, platformTypeScript))
	cmd.MarkFlagRequired("platform")

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated files")

	cmd.Flags().BoolVar(&local, "local", false, "[experimental] Generate from local specs instead of the remote workspace (no apply or network needed); requires the 'localTyper' flag")
	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the project directory or spec file (used with --local)")

	cmd.Flags().StringArrayVar(&options, "option", []string{},
		"Platform-specific options in key=value format (use 'rudder-cli typer options <platform>' to see available options)")

	return cmd
}

// parsePlatformOptions converts a slice of "key=value" strings into a map[string]string
func parsePlatformOptions(optionStrs []string) map[string]string {
	options := make(map[string]string)
	for _, optStr := range optionStrs {
		parts := strings.SplitN(optStr, "=", 2)
		if len(parts) == 2 {
			options[parts[0]] = parts[1]
		} else {
			options[parts[0]] = ""
		}
	}
	return options
}
