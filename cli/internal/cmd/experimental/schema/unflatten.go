package schema

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/experimental/schema/unflatten"
	"github.com/rudderlabs/rudder-iac/cli/pkg/experimental/schema/utils"
	"github.com/spf13/cobra"
)

// NewCmdUnflatten creates the unflatten command
func NewCmdUnflatten() *cobra.Command {
	var (
		dryRun  bool
		verbose bool
		indent  int
	)

	cmd := &cobra.Command{
		Use:   "unflatten [input-file] [output-file]",
		Short: "Unflatten schema JSON files",
		Long: `Unflatten converts flattened schema keys back to nested JSON structures.
Input file should contain schemas with flattened keys using dot notation.
Output file will contain the same schemas with properly nested structures.

Examples:
  rudder-cli experimental schema unflatten input.json output.json
  rudder-cli experimental schema unflatten input.json output.json --verbose --dry-run`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnflatten(args[0], args[1], dryRun, verbose, indent)
		},
	}

	// Add flags for the unflatten command
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without writing output file")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().IntVar(&indent, "indent", 2, "Number of spaces for JSON indentation")

	return cmd
}

// runUnflatten handles the unflatten command execution
func runUnflatten(inputFile, outputFile string, dryRun, verbose bool, indent int) error {
	if verbose {
		fmt.Printf("Processing %s...\n", inputFile)
	}

	// Check if input file exists
	if !utils.FileExists(inputFile) {
		return fmt.Errorf("input file %s does not exist", inputFile)
	}

	// Read the input file
	schemasFile, err := utils.ReadSchemasFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d schemas to unflatten\n", len(schemasFile.Schemas))
	}

	// Process each schema
	processedCount := 0
	for i := range schemasFile.Schemas {
		if len(schemasFile.Schemas[i].Schema) > 0 {
			schemasFile.Schemas[i].Schema = unflatten.UnflattenSchema(schemasFile.Schemas[i].Schema)
			processedCount++
		}
	}

	if verbose {
		fmt.Printf("Successfully unflattened %d schemas\n", processedCount)
	}

	if dryRun {
		fmt.Printf("DRY RUN: Would write output to %s\n", outputFile)
		if verbose {
			fmt.Printf("DRY RUN: First schema preview:\n")
			if len(schemasFile.Schemas) > 0 {
				fmt.Printf("  Event: %s\n", schemasFile.Schemas[0].EventIdentifier)
				fmt.Printf("  Schema keys count: %d\n", countKeys(schemasFile.Schemas[0].Schema))
			}
		}
		return nil
	}

	// Write the output file
	if err := utils.WriteSchemasFile(outputFile, schemasFile, indent); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✓ Successfully unflattened %d schemas\n", processedCount)
	fmt.Printf("✓ Output written to %s\n", outputFile)

	return nil
}

// countKeys recursively counts the number of keys in a nested structure
func countKeys(obj interface{}) int {
	switch v := obj.(type) {
	case map[string]interface{}:
		count := 0
		for _, value := range v {
			count += 1 + countKeys(value)
		}
		return count
	case []interface{}:
		count := 0
		for _, value := range v {
			count += countKeys(value)
		}
		return count
	default:
		return 0
	}
}
