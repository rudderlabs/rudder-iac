package typer

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/spf13/cobra"
)

var (
	log = logger.New("typer.cmd")
)

func NewCmdTyper() *cobra.Command {
	var (
		err            error
		trackingPlanID string
		platform       string
		outputPath     string
	)

	cmd := &cobra.Command{
		Use:   "typer",
		Short: "Generate platform-specific RudderAnalytics bindings from tracking plans",
		Long: heredoc.Doc(`
			RudderTyper generates strongly-typed RudderAnalytics bindings from your tracking plans.
			It reads tracking plan configurations and generates platform-specific code for type-safe event tracking.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli typer --tracking-plan-id tp_123 --platform kotlin --output ./generated
			$ rudder-cli typer --tracking-plan-id tp_123 --platform kotlin --output ./src/main/kotlin
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate required flags
			if trackingPlanID == "" {
				return fmt.Errorf("tracking plan ID is required (use --tracking-plan-id flag)")
			}
			if platform == "" {
				return fmt.Errorf("platform is required (use --platform flag)")
			}
			if outputPath == "" {
				return fmt.Errorf("output path is required (use --output flag)")
			}

			// Mock: initialize orchestrator
			log.Debug("mock: initialize orchestrator")
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Debug("typer generation",
				"trackingPlanID", trackingPlanID,
				"platform", platform,
				"outputPath", outputPath)

			defer func() {
				telemetry.TrackCommand("typer", err, []telemetry.KV{
					{K: "trackingPlanID", V: trackingPlanID},
					{K: "platform", V: platform},
					{K: "outputPath", V: outputPath},
				}...)
			}()

			// Provide user feedback
			fmt.Printf("🔧 Generating %s bindings for tracking plan: %s\n", platform, trackingPlanID)
			fmt.Println("📥 Fetching tracking plan data...")

			log.Debug("mock: would fetch tracking plan from provider", "id", trackingPlanID)

			fmt.Printf("⚡ Generating %s code...\n", platform)
			log.Debug("mock: would generate code for platform", "platform", platform)

			fmt.Printf("📁 Writing files to %s...\n", outputPath)
			log.Debug("mock: would write files to output directory", "path", outputPath)

			fmt.Printf("✅ Successfully generated %s bindings in %s\n", platform, outputPath)
			log.Debug("mock implementation complete - real orchestrator will be added in PR 2")

			return nil
		},
	}

	// Add required flags
	cmd.Flags().StringVarP(&trackingPlanID, "tracking-plan-id", "t", "", "Tracking plan ID to generate bindings for (required)")
	cmd.Flags().StringVarP(&platform, "platform", "p", "", "Target platform for code generation (required)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output directory path for generated files (required)")

	// Mark required flags
	cmd.MarkFlagRequired("tracking-plan-id")
	cmd.MarkFlagRequired("platform")
	cmd.MarkFlagRequired("output")

	return cmd
}
