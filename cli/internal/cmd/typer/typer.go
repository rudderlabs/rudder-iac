package typer

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/providers"
	"github.com/spf13/cobra"
)

func NewCmdTyper() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "typer",
		Short:  "Generate type-safe tracking code",
		Long:   "Generate type-safe tracking code from RudderStack tracking plans",
		Hidden: true,
		Args:   cobra.NoArgs,
	}

	cmd.AddCommand(newCmdGenerate())

	return cmd
}

func newCmdGenerate() *cobra.Command {
	var trackingPlanID string
	var platform string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate type-safe code from tracking plan",
		Long:  "Generate type-safe code from a RudderStack tracking plan",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !config.GetConfig().ExperimentalFlags.RudderTyper {
				return fmt.Errorf("typer commands are disabled")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if trackingPlanID == "" {
				return fmt.Errorf("tracking-plan-id is required")
			}

			if platform != "kotlin" {
				return fmt.Errorf("unsupported platform: %s (supported platforms: kotlin)", platform)
			}

			defer func() {
				telemetry.TrackCommand("typer", nil, []telemetry.KV{
					{K: "platform", V: platform},
				}...)
			}()

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("failed to initialize dependencies: %w", err)
			}

			client := deps.Client()
			dataCatalogClient := catalog.NewRudderDataCatalog(client)

			planProvider := providers.NewJSONSchemaPlanProvider(trackingPlanID, dataCatalogClient)
			rudderTyper := typer.NewRudderTyper(planProvider)

			options := typer.GenerationOptions{
				Platform:   platform,
				OutputPath: outputDir,
			}

			ctx := context.Background()
			return rudderTyper.Generate(ctx, options)
		},
	}

	cmd.Flags().StringVar(&trackingPlanID, "tracking-plan-id", "", "Tracking plan ID to generate code from")
	cmd.MarkFlagRequired("tracking-plan-id")

	cmd.Flags().StringVar(&platform, "platform", "kotlin", "Platform to generate code for (kotlin)")
	cmd.MarkFlagRequired("platform")

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated files")

	return cmd
}
