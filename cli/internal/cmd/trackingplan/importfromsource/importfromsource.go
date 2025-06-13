package importfromsource

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/schema"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdTPImportFromSource() *cobra.Command {
	var (
		writeKey   string
		configFile string
		dryRun     bool
		verbose    bool
	)

	cmd := &cobra.Command{
		Use:   "importFromSource [output-dir]",
		Short: "Import schemas from source and convert them to Data Catalog YAML files",
		Long: `Import schemas from source using an optimized in-memory workflow that chains fetch, 
unflatten, and convert operations.

This command performs the complete workflow:
1. Fetch schemas from the Event Audit API (with optional writeKey filter)
2. Unflatten schemas using event-type-specific JSONPath mappings from config  
3. Convert unflattened schemas to RudderStack Data Catalog YAML files

The workflow uses in-memory processing for optimal performance, eliminating 
temporary files and reducing I/O overhead.

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
- Without config: all event types use "$.properties" 
- Track events always use "$.properties"`,
		Example: heredoc.Doc(`
			# Import all schemas with default configuration
			$ rudder-cli tp importFromSource output/

			# Import with specific writeKey
			$ rudder-cli tp importFromSource output/ --write-key "your-write-key"

			# Import with custom event type mappings  
			$ rudder-cli tp importFromSource output/ --config mappings.yaml

			# Dry run with verbose output
			$ rudder-cli tp importFromSource output/ --config mappings.yaml --dry-run --verbose
		`),
		Args: cobra.ExactArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !viper.GetBool("experimental") {
				return fmt.Errorf("importFromSource command requires experimental mode. Set RUDDERSTACK_CLI_EXPERIMENTAL=true or add \"experimental\": true to your config file")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputDir := args[0]
			return runImportFromSource(outputDir, writeKey, configFile, dryRun, verbose)
		},
	}

	cmd.Flags().StringVar(&writeKey, "write-key", "", "Write key to filter schemas (optional)")
	cmd.Flags().StringVar(&configFile, "config", "", "Path to YAML config file with event type mappings (optional)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

	return cmd
}

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
		fmt.Printf("✅ Fetched %d schemas\n\n", len(schemas))
	}

	// Step 3: Unflatten schemas in memory (optimized - no temp files)
	if verbose {
		fmt.Printf("🔄 Step 2/3: Unflattening schemas with event-type configuration...\n")
	}

	unflattenedSchemas, err := schema.UnflattenSchemasWithEventTypeConfig(
		schemas,
		eventTypeConfig,
		false, // skipFailed
		verbose,
	)
	if err != nil {
		return fmt.Errorf("failed to unflatten schemas: %w", err)
	}

	if verbose {
		fmt.Printf("✅ Unflattened %d schemas\n\n", len(unflattenedSchemas))
	}

	// Step 4: Convert schemas to YAML files (optimized - direct conversion)
	if verbose {
		fmt.Printf("📝 Step 3/3: Converting schemas to YAML files...\n")
	}

	result, err := schema.ConvertSchemasToYAML(
		unflattenedSchemas,
		outputDir,
		dryRun,
		verbose,
		2, // indent
	)
	if err != nil {
		return fmt.Errorf("failed to convert schemas: %w", err)
	}

	// Step 5: Display results
	if dryRun {
		fmt.Printf("🔍 DRY RUN COMPLETE - No files were created\n\n")
	} else {
		fmt.Printf("✅ IMPORT COMPLETE\n\n")
	}

	fmt.Printf("📊 Summary:\n")
	fmt.Printf("  • Fetched: %d schemas\n", len(schemas))
	fmt.Printf("  • Processed: %d schemas\n", len(unflattenedSchemas))
	if result != nil {
		fmt.Printf("  • Generated: %d YAML files\n", len(result.GeneratedFiles))
		fmt.Printf("  • Events: %d\n", result.EventsCount)
		fmt.Printf("  • Properties: %d\n", result.PropertiesCount)
		fmt.Printf("  • Custom Types: %d\n", result.CustomTypesCount)
	}
	fmt.Printf("  • Output directory: %s\n", outputDir)

	if !dryRun {
		fmt.Printf("\n🎉 Your schemas are ready to use with RudderStack Data Catalog!\n")
	}

	return nil
}
