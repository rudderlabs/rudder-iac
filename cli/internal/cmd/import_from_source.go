package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/schema"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdImportFromSource() *cobra.Command {
	var (
		writeKey   string
		configFile string
		dryRun     bool
		verbose    bool
	)

	cmd := &cobra.Command{
		Use:   "importFromSource [output-dir]",
		Short: "Import schemas from source and convert them to Data Catalog YAML files",
		Long: `Import schemas from source using an optimized in-memory workflow for maximum performance and efficiency.

🚀 This command performs the complete workflow with high performance:
1. Fetch schemas from the Event Audit API into memory (with optional writeKey filter)
2. Process schemas using event-type-specific JSONPath mappings in memory
3. Convert processed schemas to RudderStack Data Catalog YAML files directly from memory

⚡ Performance Benefits:
- No temporary file I/O between processing stages
- No JSON marshalling/unmarshalling overhead between steps
- Reduced memory usage and faster execution
- All processing done in-memory with Go structs

Configuration File Format (YAML):
  event_mappings:
    identify: "$.traits"           # or "$.context.traits" or "$.properties"
    page: "$.context.traits"       # or "$.traits" or "$.properties"
    screen: "$.properties"         # or "$.traits" or "$.context.traits"
    group: "$.properties"          # or "$.traits" or "$.context.traits"
    alias: "$.traits"              # or "$.context.traits" or "$.properties"
  # Note: "track" always uses "$.properties" regardless of config

Default Behavior:
- Without writeKey: fetches all schemas  
- Without config: all event types use "$.properties" mapping
- track events always use "$.properties" even when config is provided`,
		Example: heredoc.Doc(`
			$ rudder-cli importFromSource output/
			$ rudder-cli importFromSource output/ --write-key=YOUR_WRITE_KEY
			$ rudder-cli importFromSource output/ --config=event-mappings.yaml
			$ rudder-cli importFromSource output/ --write-key=YOUR_WRITE_KEY --config=event-mappings.yaml --verbose
		`),
		Args: cobra.ExactArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !viper.GetBool("experimental") {
				return fmt.Errorf("importFromSource command requires experimental mode. Set RUDDERSTACK_CLI_EXPERIMENTAL=true or add \"experimental\": true to your config file")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImportFromSource(args[0], writeKey, configFile, dryRun, verbose)
		},
	}

	// Add flags
	cmd.Flags().StringVar(&writeKey, "write-key", "", "Filter schemas by write key (source)")
	cmd.Flags().StringVar(&configFile, "config", "", "YAML configuration file for event-type-specific JSONPath mappings")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without creating files")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	return cmd
}

// runImportFromSource handles the importFromSource command execution
func runImportFromSource(outputDir, writeKey, configFile string, dryRun, verbose bool) error {
	// Step 1: Load configuration
	eventTypeConfig, err := models.LoadEventTypeConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load event type configuration: %w", err)
	}

	if verbose {
		fmt.Printf("🎯 Event Type Configuration:\n")
		for eventType, path := range eventTypeConfig.EventMappings {
			fmt.Printf("  %s → %s\n", eventType, path)
		}
		fmt.Printf("  track → $.properties (always)\n")
		fmt.Printf("\n")
	}

	// Step 2: Fetch schemas in memory (optimized - no temp files)
	if verbose {
		fmt.Printf("📥 Step 1/3: Fetching schemas...\n")
	}

	schemas, err := schema.FetchSchemas(writeKey, verbose)
	if err != nil {
		return fmt.Errorf("failed to fetch schemas: %w", err)
	}

	if verbose {
		fmt.Printf("✓ Fetched %d schemas in memory\n\n", len(schemas))
	}

	// Step 3: Unflatten schemas with event-type configuration in memory (optimized)
	if verbose {
		fmt.Printf("🔧 Step 2/3: Processing schemas with event-type configuration...\n")
	}

	processedSchemas, err := schema.UnflattenSchemasWithEventTypeConfig(schemas, eventTypeConfig, false, verbose)
	if err != nil {
		return fmt.Errorf("failed to unflatten schemas: %w", err)
	}

	if verbose {
		fmt.Printf("✓ Processed %d schemas in memory\n\n", len(processedSchemas))
	}

	// Step 4: Convert to YAML files in memory (optimized)
	if verbose {
		fmt.Printf("📄 Step 3/3: Converting to YAML files...\n")
	}

	result, err := schema.ConvertSchemasToYAML(processedSchemas, outputDir, dryRun, verbose, 2)
	if err != nil {
		return fmt.Errorf("failed to convert schemas: %w", err)
	}

	// Display final results
	if dryRun {
		fmt.Printf("\n🎉 DRY RUN COMPLETED SUCCESSFULLY!\n")
		fmt.Printf("Would generate %d files using in-memory processing (no temp files)\n",
			result.EventsCount+result.PropertiesCount+result.CustomTypesCount+len(result.TrackingPlans))
	} else {
		fmt.Printf("\n🎉 IMPORT FROM SOURCE COMPLETED SUCCESSFULLY!\n")
		fmt.Printf("Generated %d files using optimized in-memory processing\n", len(result.GeneratedFiles))
	}

	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  Events: %d\n", result.EventsCount)
	fmt.Printf("  Properties: %d\n", result.PropertiesCount)
	fmt.Printf("  Custom Types: %d\n", result.CustomTypesCount)
	fmt.Printf("  Tracking Plans: %d\n", len(result.TrackingPlans))

	if verbose {
		fmt.Printf("\n⚡ Performance Benefits:\n")
		fmt.Printf("  • No temporary file I/O between stages\n")
		fmt.Printf("  • No JSON marshalling/unmarshalling overhead\n")
		fmt.Printf("  • Reduced memory usage and faster execution\n")
		fmt.Printf("  • All processing done in-memory with Go structs\n")
	}

	if !dryRun {
		fmt.Printf("\n📁 Output Directory: %s\n", outputDir)
		fmt.Printf("🎯 Next Steps:\n")
		fmt.Printf("  1. Review the generated YAML files\n")
		fmt.Printf("  2. Validate with: rudder-cli tp validate -l %s\n", outputDir)
		fmt.Printf("  3. Deploy with: rudder-cli tp apply -l %s\n", outputDir)
	}

	return nil
}
