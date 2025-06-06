package schema

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/config"
	"github.com/rudderlabs/rudder-iac/cli/pkg/schema/client"
	pkgModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/spf13/cobra"
)

// NewCmdFetch creates the fetch command
func NewCmdFetch() *cobra.Command {
	var (
		writeKey string
		dryRun   bool
		verbose  bool
		indent   int
	)

	cmd := &cobra.Command{
		Use:   "fetch [output-file]",
		Short: "Fetch event schemas from the API",
		Long: `Fetch event schemas from the Event Audit API and save them to a JSON file.
The output file will have the same structure as schemas_real.json.

Authentication and Configuration:
- Uses the main CLI's authentication system
- Access token: Set via 'rudder-cli auth login' or RUDDERSTACK_ACCESS_TOKEN environment variable
- API URL: Set via RUDDERSTACK_API_URL environment variable (defaults to RudderStack API)

Examples:
  rudder-cli schema fetch schemas.json
  rudder-cli schema fetch schemas.json --write-key=YOUR_WRITE_KEY
  rudder-cli schema fetch schemas.json --verbose --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(args[0], writeKey, dryRun, verbose, indent)
		},
	}

	// Add flags for the fetch command
	cmd.Flags().StringVar(&writeKey, "write-key", "", "Filter schemas by write key (source)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without writing output file")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().IntVar(&indent, "indent", 2, "Number of spaces for JSON indentation")

	return cmd
}

// runFetch handles the fetch command execution
func runFetch(outputFile, writeKey string, dryRun, verbose bool, indent int) error {
	if verbose {
		fmt.Printf("Fetching schemas from API...\n")
	}

	// Load configuration from environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if verbose {
		fmt.Printf("Using API URL: %s\n", cfg.APIURL)
		if writeKey != "" {
			fmt.Printf("Filtering by write key: %s\n", writeKey)
		}
	}

	// Create API client
	apiClient := client.NewSchemaClient(cfg.APIURL, cfg.APIToken)

	// Fetch schemas
	schemas, err := apiClient.FetchAllSchemas(writeKey)
	if err != nil {
		return fmt.Errorf("failed to fetch schemas: %w", err)
	}

	if verbose {
		fmt.Printf("Successfully fetched %d schemas\n", len(schemas))
	}

	// Create output structure
	output := pkgModels.SchemasFile{
		Schemas: schemas,
	}

	if dryRun {
		fmt.Printf("DRY RUN: Would write %d schemas to %s\n", len(schemas), outputFile)
		if verbose && len(schemas) > 0 {
			fmt.Printf("DRY RUN: First schema preview:\n")
			fmt.Printf("  UID: %s\n", schemas[0].UID)
			fmt.Printf("  Event: %s\n", schemas[0].EventIdentifier)
			fmt.Printf("  Write Key: %s\n", schemas[0].WriteKey)
			fmt.Printf("  Schema fields count: %d\n", len(schemas[0].Schema))
		}
		return nil
	}

	// Write output file
	if err := writeJSONFile(outputFile, output, indent); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✓ Successfully fetched %d schemas\n", len(schemas))
	fmt.Printf("✓ Output written to %s\n", outputFile)

	return nil
}

// writeJSONFile writes the schemas to a JSON file with proper indentation
func writeJSONFile(filename string, data interface{}, indent int) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if indent > 0 {
		encoder.SetIndent("", fmt.Sprintf("%*s", indent, ""))
	}

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}
