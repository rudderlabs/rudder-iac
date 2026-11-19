package typer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/providers"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/typer/generator/core"
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
	cmd.AddCommand(newCmdSnapshot())
	cmd.AddCommand(newCmdOptions())

	return cmd
}

func newCmdGenerate() *cobra.Command {
	var trackingPlanID string
	var trackingPlanVersion int
	var platform string
	var outputDir string
	var options []string
	var local bool
	var location string
	var snapshot string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate type-safe code from tracking plan",
		Long:  "Generate type-safe code from a RudderStack tracking plan",
		Example: heredoc.Doc(`
			$ rudder-cli typer generate --tracking-plan-id <id> --platform kotlin
			$ rudder-cli typer generate --local --location ./project --platform kotlin
			$ rudder-cli typer generate --tracking-plan-id <id> --tracking-plan-version 3 --platform kotlin
			$ rudder-cli typer generate --snapshot ./tracking-plan.snapshot.json --platform kotlin
		`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
					{K: "platform", V: platform},
					{K: "local", V: local},
					{K: "snapshot", V: snapshot != ""},
				}...)
			}()

			if local && snapshot != "" {
				return fmt.Errorf("--local and --snapshot cannot be used together")
			}
			// A snapshot is one plan, so an ID that names another plan would be
			// ignored and the wrong bindings generated without any warning
			if snapshot != "" && trackingPlanID != "" {
				return fmt.Errorf("--tracking-plan-id does not apply with --snapshot")
			}
			if err := validateTrackingPlanVersion(trackingPlanVersion, local || snapshot != ""); err != nil {
				return err
			}

			validPlatforms := map[string]bool{platformKotlin: true, platformSwift: true, platformTypeScript: true}
			if !validPlatforms[platform] {
				supported := make([]string, 0, len(validPlatforms))
				for p := range validPlatforms {
					supported = append(supported, p)
				}
				sort.Strings(supported)
				return fmt.Errorf("unsupported platform: %s (supported platforms: %s)", platform, strings.Join(supported, ", "))
			}

			var planProvider typer.PlanProvider
			switch {
			case snapshot != "":
				planProvider = providers.NewSnapshotPlanProvider(snapshot)
			case local:
				planProvider, err = providers.NewLocalCatalogPlanProviderForProject(location, trackingPlanID)
			default:
				planProvider, err = providers.NewRemoteCatalogPlanProvider(trackingPlanID, trackingPlanVersion)
			}
			if err != nil {
				return err
			}

			core.Warn = ui.PrintWarning
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

	cmd.Flags().IntVar(&trackingPlanVersion, "tracking-plan-version", 0, "Tracking plan version to generate code from (remote only). Defaults to the latest version")

	cmd.Flags().StringVar(&platform, "platform", platformKotlin, fmt.Sprintf("Platform to generate code for (%s, %s, %s)", platformKotlin, platformSwift, platformTypeScript))
	cmd.MarkFlagRequired("platform")

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated files")

	cmd.Flags().BoolVar(&local, "local", false, "Generate from local specs instead of the remote workspace (no workspace, apply, auth or network needed)")
	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the project directory or spec file (used with --local)")

	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Generate from a snapshot file written by 'typer snapshot'. Needs no access token and makes no API calls")

	cmd.Flags().StringArrayVar(&options, "option", []string{},
		"Platform-specific options in key=value format (use 'rudder-cli typer options <platform>' to see available options)")

	return cmd
}

// validateTrackingPlanVersion rejects a version that cannot apply. Local specs and
// snapshots are already a single fixed version, so a version there is a mistake
// worth surfacing instead of ignoring.
func validateTrackingPlanVersion(version int, fixedInput bool) error {
	if version < 0 {
		return fmt.Errorf("--tracking-plan-version must not be negative")
	}
	if version > 0 && fixedInput {
		return fmt.Errorf("--tracking-plan-version only applies when generating from the remote workspace")
	}
	return nil
}

func newCmdSnapshot() *cobra.Command {
	var (
		trackingPlanID      string
		trackingPlanVersion int
		outputFile          string
	)

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Save a remote tracking plan to a file for offline generation",
		Long: heredoc.Doc(`
			Fetches a tracking plan from the remote workspace and writes it to a file.
			Run this where credentials are available (for example in CI), then run
			'typer generate --snapshot' wherever the code is built, with no access token
			and no API calls.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli typer snapshot --tracking-plan-id <id> -o tracking-plan.snapshot.json
			$ rudder-cli typer snapshot --tracking-plan-id <id> --tracking-plan-version 3 -o tracking-plan.snapshot.json
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err)
			}()

			if trackingPlanID == "" {
				return fmt.Errorf("tracking-plan-id is required")
			}
			if err := validateTrackingPlanVersion(trackingPlanVersion, false); err != nil {
				return err
			}

			store, err := providers.NewRemoteCatalogStore()
			if err != nil {
				return err
			}

			tp, err := providers.WriteSnapshot(context.Background(), store, trackingPlanID, trackingPlanVersion, outputFile)
			if err != nil {
				return err
			}

			ui.Printf("Saved %q (version %d, %d events) to %s\n", tp.Name, tp.Version, len(tp.Events), outputFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&trackingPlanID, "tracking-plan-id", "", "Remote tracking plan ID to save")
	cmd.Flags().IntVar(&trackingPlanVersion, "tracking-plan-version", 0, "Tracking plan version to save. Defaults to the latest version")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "tracking-plan.snapshot.json", "File to write the snapshot to")

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
