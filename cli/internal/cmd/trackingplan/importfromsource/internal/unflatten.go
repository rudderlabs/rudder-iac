package internal

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/jsonpath"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/unflatten"
	"github.com/spf13/cobra"
)

// NewCmdUnflatten creates the unflatten command
func NewCmdUnflatten() *cobra.Command {
	var (
		dryRun     bool
		verbose    bool
		indent     int
		jsonPath   string
		skipFailed bool
	)

	cmd := &cobra.Command{
		Use:   "unflatten [input-file] [output-file]",
		Short: "Unflatten schema JSON files with optional JSONPath extraction",
		Long: `Unflatten converts flattened schema keys back to nested JSON structures.
Optionally extract specific parts of the schema using JSONPath expressions.

The command first unflattens all schemas (converting dot notation to nested structures),
then applies JSONPath extraction if specified.`,
		Example: `  rudder-cli schema unflatten input.json output.json
  rudder-cli schema unflatten input.json output.json --jsonpath "$.properties"
  rudder-cli schema unflatten input.json output.json --jsonpath "$.context.traits" --skip-failed=false
  rudder-cli schema unflatten input.json output.json --verbose --dry-run`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnflatten(args[0], args[1], dryRun, verbose, indent, jsonPath, skipFailed)
		},
	}

	// Add flags for the unflatten command
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without writing output file")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().IntVar(&indent, "indent", 2, "Number of spaces for JSON indentation")
	cmd.Flags().StringVar(&jsonPath, "jsonpath", "", "JSONPath expression to extract value from schema (relative to schema field)")
	cmd.Flags().BoolVar(&skipFailed, "skip-failed", true, "Skip schemas where JSONPath fails (false: leave schema intact)")

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

// ProcessingError tracks errors during schema processing
type ProcessingError struct {
	SchemaUID string
	Error     error
}

// runUnflatten handles the unflatten command execution
func runUnflatten(inputFile, outputFile string, dryRun, verbose bool, indent int, jsonPath string, skipFailed bool) error {
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
		fmt.Printf("Found %d schemas to process\n", len(schemasFile.Schemas))
		if jsonPath != "" {
			fmt.Printf("Using JSONPath: %s\n", jsonPath)
			fmt.Printf("Skip failed: %v\n", skipFailed)
		}
	}

	// Create JSONPath processor if needed
	var processor *jsonpath.Processor
	if jsonPath != "" {
		processor = jsonpath.NewProcessor(jsonPath, skipFailed)
	}

	// Process each schema
	var processedSchemas []models.Schema
	var processingErrors []ProcessingError
	processedCount := 0
	skippedCount := 0

	for i := range schemasFile.Schemas {
		originalSchema := &schemasFile.Schemas[i]

		// Step 1: Always unflatten first
		if len(originalSchema.Schema) > 0 {
			originalSchema.Schema = unflatten.UnflattenSchema(originalSchema.Schema)
		}

		// Step 2: Apply JSONPath extraction if specified
		if processor != nil && !processor.IsRootPath() {
			result := processor.ProcessSchema(originalSchema.Schema)
			if result.Error != nil {
				// Handle error based on skip-failed setting
				processingErrors = append(processingErrors, ProcessingError{
					SchemaUID: originalSchema.UID,
					Error:     result.Error,
				})

				if processor.ShouldSkipOnError() {
					// Skip this schema
					skippedCount++
					continue
				}
				// else: keep the unflattened schema and continue
			} else {
				// Replace schema with extracted value
				// The result might be any type (map, array, primitive), so we need to handle it properly
				switch v := result.Value.(type) {
				case map[string]interface{}:
					originalSchema.Schema = v
				case []interface{}:
					// For arrays, keep them as arrays but we need to convert to map[string]interface{}
					// Since schema must be map[string]interface{}, we wrap it
					originalSchema.Schema = map[string]interface{}{
						"items": v,
					}
				case string, float64, bool, nil:
					// For primitive types, wrap them in a map to maintain schema structure
					originalSchema.Schema = map[string]interface{}{
						"value": v,
					}
				default:
					// Fallback: convert to map
					originalSchema.Schema = map[string]interface{}{
						"value": v,
					}
				}
			}
		}

		processedSchemas = append(processedSchemas, *originalSchema)
		processedCount++
	}

	// Update the schemas file with processed schemas
	schemasFile.Schemas = processedSchemas

	// Report processing results
	if verbose || len(processingErrors) > 0 {
		fmt.Printf("✓ Successfully processed %d schemas\n", processedCount)
		if skippedCount > 0 {
			fmt.Printf("⚠ Skipped %d schemas due to JSONPath errors\n", skippedCount)
		}
		if len(processingErrors) > 0 {
			fmt.Printf("⚠ JSONPath processing errors:\n")
			for _, err := range processingErrors {
				fmt.Printf("  - Schema UID %s: %s\n", err.SchemaUID, err.Error.Error())
			}
		}
	}

	if dryRun {
		fmt.Printf("DRY RUN: Would write output to %s\n", outputFile)
		if verbose {
			fmt.Printf("DRY RUN: Final output preview:\n")
			if len(schemasFile.Schemas) > 0 {
				fmt.Printf("  Schemas count: %d\n", len(schemasFile.Schemas))
				fmt.Printf("  First schema event: %s\n", schemasFile.Schemas[0].EventIdentifier)
				fmt.Printf("  First schema keys count: %d\n", countKeys(schemasFile.Schemas[0].Schema))
			}
		}
		return nil
	}

	// Write the output file
	if err := writeSchemasFile(outputFile, schemasFile, indent); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✓ Successfully processed %d schemas\n", processedCount)
	if skippedCount > 0 {
		fmt.Printf("⚠ Skipped %d schemas due to JSONPath errors\n", skippedCount)
	}
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

// runUnflattenWithEventTypeConfig handles unflatten with event-type-specific configuration
func runUnflattenWithEventTypeConfig(inputFile, outputFile string, dryRun, verbose bool, indent int, eventTypeConfig *models.EventTypeConfig, skipFailed bool) error {
	if verbose {
		fmt.Printf("Processing %s with event-type-specific configuration...\n", inputFile)
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
		fmt.Printf("Found %d schemas to process\n", len(schemasFile.Schemas))
		if eventTypeConfig.HasCustomMappings() {
			fmt.Printf("Using event-type-specific configuration\n")
		}
	}

	// Process each schema
	var processedSchemas []models.Schema
	var processingErrors []ProcessingError
	processedCount := 0
	skippedCount := 0

	for i := range schemasFile.Schemas {
		originalSchema := &schemasFile.Schemas[i]

		// Step 1: Always unflatten first
		if len(originalSchema.Schema) > 0 {
			originalSchema.Schema = unflatten.UnflattenSchema(originalSchema.Schema)
		}

		// Step 2: Apply event-type-specific JSONPath extraction
		jsonPath := eventTypeConfig.GetJSONPathForEventType(originalSchema.EventType)

		if verbose {
			fmt.Printf("Schema %s (eventType: %s) using JSONPath: %s\n",
				originalSchema.UID, originalSchema.EventType, jsonPath)
		}

		// Create processor for this schema's event type
		processor := jsonpath.NewProcessor(jsonPath, skipFailed)

		if processor != nil && !processor.IsRootPath() {
			result := processor.ProcessSchema(originalSchema.Schema)
			if result.Error != nil {
				// Handle error based on skip-failed setting
				processingErrors = append(processingErrors, ProcessingError{
					SchemaUID: fmt.Sprintf("%s (eventType: %s)", originalSchema.UID, originalSchema.EventType),
					Error:     result.Error,
				})

				if processor.ShouldSkipOnError() {
					// Skip this schema
					skippedCount++
					continue
				}
				// else: keep the unflattened schema and continue
			} else {
				// Replace schema with extracted value
				switch v := result.Value.(type) {
				case map[string]interface{}:
					originalSchema.Schema = v
				case []interface{}:
					// For arrays, wrap them to maintain schema structure
					originalSchema.Schema = map[string]interface{}{
						"items": v,
					}
				case string, float64, bool, nil:
					// For primitive types, wrap them in a map to maintain schema structure
					originalSchema.Schema = map[string]interface{}{
						"value": v,
					}
				default:
					// Fallback: convert to map
					originalSchema.Schema = map[string]interface{}{
						"value": v,
					}
				}
			}
		}

		processedSchemas = append(processedSchemas, *originalSchema)
		processedCount++
	}

	// Update the schemas file with processed schemas
	schemasFile.Schemas = processedSchemas

	// Report processing results
	if verbose || len(processingErrors) > 0 {
		fmt.Printf("✓ Successfully processed %d schemas\n", processedCount)
		if skippedCount > 0 {
			fmt.Printf("⚠ Skipped %d schemas due to JSONPath errors\n", skippedCount)
		}
		if len(processingErrors) > 0 {
			fmt.Printf("⚠ JSONPath processing errors:\n")
			for _, err := range processingErrors {
				fmt.Printf("  - Schema %s: %s\n", err.SchemaUID, err.Error.Error())
			}
		}
	}

	if dryRun {
		fmt.Printf("DRY RUN: Would write output to %s\n", outputFile)
		if verbose {
			fmt.Printf("DRY RUN: Final output preview:\n")
			if len(schemasFile.Schemas) > 0 {
				fmt.Printf("  Schemas count: %d\n", len(schemasFile.Schemas))
				fmt.Printf("  First schema event: %s\n", schemasFile.Schemas[0].EventIdentifier)
				fmt.Printf("  First schema keys count: %d\n", countKeys(schemasFile.Schemas[0].Schema))
			}
		}
		return nil
	}

	// Write the output file
	if err := writeSchemasFile(outputFile, schemasFile, indent); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✓ Successfully processed %d schemas\n", processedCount)
	if skippedCount > 0 {
		fmt.Printf("⚠ Skipped %d schemas due to JSONPath errors\n", skippedCount)
	}
	fmt.Printf("✓ Output written to %s\n", outputFile)

	return nil
}

// RunUnflattenWithEventTypeConfig is a public wrapper for runUnflattenWithEventTypeConfig to be used by other commands
func RunUnflattenWithEventTypeConfig(inputFile, outputFile string, dryRun, verbose bool, indent int, eventTypeConfig *models.EventTypeConfig, skipFailed bool) error {
	return runUnflattenWithEventTypeConfig(inputFile, outputFile, dryRun, verbose, indent, eventTypeConfig, skipFailed)
}

// UnflattenSchemasWithEventTypeConfig processes schemas in memory with event-type-specific configuration
func UnflattenSchemasWithEventTypeConfig(schemas []models.Schema, eventTypeConfig *models.EventTypeConfig, skipFailed, verbose bool) ([]models.Schema, error) {
	if verbose {
		fmt.Printf("Processing %d schemas with event-type-specific configuration in memory...\n", len(schemas))
		if eventTypeConfig.HasCustomMappings() {
			fmt.Printf("Using event-type-specific configuration\n")
		}
	}

	// Process each schema
	var processedSchemas []models.Schema
	var processingErrors []ProcessingError
	processedCount := 0
	skippedCount := 0

	for i := range schemas {
		originalSchema := schemas[i] // Create a copy to avoid modifying the original

		// Step 1: Always unflatten first
		if len(originalSchema.Schema) > 0 {
			originalSchema.Schema = unflatten.UnflattenSchema(originalSchema.Schema)
		}

		// Step 2: Apply event-type-specific JSONPath extraction
		jsonPath := eventTypeConfig.GetJSONPathForEventType(originalSchema.EventType)

		if verbose {
			fmt.Printf("Schema %s (eventType: %s) using JSONPath: %s\n",
				originalSchema.UID, originalSchema.EventType, jsonPath)
		}

		// Create processor for this schema's event type
		processor := jsonpath.NewProcessor(jsonPath, skipFailed)

		if processor != nil && !processor.IsRootPath() {
			result := processor.ProcessSchema(originalSchema.Schema)
			if result.Error != nil {
				// Handle error based on skip-failed setting
				processingErrors = append(processingErrors, ProcessingError{
					SchemaUID: fmt.Sprintf("%s (eventType: %s)", originalSchema.UID, originalSchema.EventType),
					Error:     result.Error,
				})

				if processor.ShouldSkipOnError() {
					// Skip this schema
					skippedCount++
					continue
				}
				// else: keep the unflattened schema and continue
			} else {
				// Replace schema with extracted value
				switch v := result.Value.(type) {
				case map[string]interface{}:
					originalSchema.Schema = v
				case []interface{}:
					// For arrays, wrap them to maintain schema structure
					originalSchema.Schema = map[string]interface{}{
						"items": v,
					}
				case string, float64, bool, nil:
					// For primitive types, wrap them in a map to maintain schema structure
					originalSchema.Schema = map[string]interface{}{
						"value": v,
					}
				default:
					// Fallback: convert to map
					originalSchema.Schema = map[string]interface{}{
						"value": v,
					}
				}
			}
		}

		processedSchemas = append(processedSchemas, originalSchema)
		processedCount++
	}

	// Report processing results
	if verbose || len(processingErrors) > 0 {
		fmt.Printf("✓ Successfully processed %d schemas\n", processedCount)
		if skippedCount > 0 {
			fmt.Printf("⚠ Skipped %d schemas due to JSONPath errors\n", skippedCount)
		}
		if len(processingErrors) > 0 {
			fmt.Printf("⚠ JSONPath processing errors:\n")
			for _, err := range processingErrors {
				fmt.Printf("  - Schema %s: %s\n", err.SchemaUID, err.Error.Error())
			}
		}
	}

	return processedSchemas, nil
}
