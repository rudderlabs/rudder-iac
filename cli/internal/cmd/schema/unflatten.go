package schema

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/unflatten"
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
  rudder-cli schema unflatten input.json output.json
  rudder-cli schema unflatten input.json output.json --verbose --dry-run`,
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

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// readSchemasFile reads a schemas JSON file and returns the parsed structure
func readSchemasFile(filePath string) (*models.SchemasFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var schemasFile models.SchemasFile
	if err := json.Unmarshal(data, &schemasFile); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", filePath, err)
	}

	return &schemasFile, nil
}

// writeSchemasFile writes the schemas structure to a JSON file with proper formatting
func writeSchemasFile(filePath string, schemasFile *models.SchemasFile, indent int) error {
	var data []byte
	var err error

	if indent > 0 {
		indentStr := ""
		for i := 0; i < indent; i++ {
			indentStr += " "
		}
		data, err = json.MarshalIndent(schemasFile, "", indentStr)
	} else {
		data, err = json.Marshal(schemasFile)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// runUnflatten handles the unflatten command execution
func runUnflatten(inputFile, outputFile string, dryRun, verbose bool, indent int) error {
	if verbose {
		fmt.Printf("Processing %s...\n", inputFile)
	}

	// Check if input file exists
	if !fileExists(inputFile) {
		return fmt.Errorf("input file %s does not exist", inputFile)
	}

	// Read the input file
	schemasFile, err := readSchemasFile(inputFile)
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
	if err := writeSchemasFile(outputFile, schemasFile, indent); err != nil {
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
