package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"gopkg.in/yaml.v3"
)

// ConversionOptions holds configuration for the conversion process
type ConversionOptions struct {
	InputFile  string
	OutputDir  string
	DryRun     bool
	Verbose    bool
	YAMLIndent int
}

// ConversionResult holds the results of the conversion process
type ConversionResult struct {
	EventsCount      int
	PropertiesCount  int
	CustomTypesCount int
	TrackingPlans    []string
	GeneratedFiles   []string
}

// SchemaConverter handles the conversion of unflattened schemas to YAML files
type SchemaConverter struct {
	analyzer *SchemaAnalyzer
	options  ConversionOptions
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

// writeYAMLFile writes YAML data to a file with specified indentation
func writeYAMLFile(filename string, data interface{}, indent int) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(indent)
	defer encoder.Close()

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

// printEventsPreview shows a preview of events (simplified version)
func printEventsPreview(eventsYAML *yamlModels.EventsYAML, maxItems int) {
	fmt.Printf("Events preview (%d total):\n", len(eventsYAML.Spec.Events))
	for i, event := range eventsYAML.Spec.Events {
		if i >= maxItems {
			fmt.Printf("  ... and %d more\n", len(eventsYAML.Spec.Events)-maxItems)
			break
		}
		fmt.Printf("  - %s (%s)\n", event.Name, event.EventType)
	}
}

// printPropertiesPreview shows a preview of properties (simplified version)
func printPropertiesPreview(propertiesYAML *yamlModels.PropertiesYAML, maxItems int) {
	fmt.Printf("Properties preview (%d total):\n", len(propertiesYAML.Spec.Properties))
	for i, prop := range propertiesYAML.Spec.Properties {
		if i >= maxItems {
			fmt.Printf("  ... and %d more\n", len(propertiesYAML.Spec.Properties)-maxItems)
			break
		}
		fmt.Printf("  - %s (%s)\n", prop.Name, prop.Type)
	}
}

// printCustomTypesPreview shows a preview of custom types (simplified version)
func printCustomTypesPreview(customTypesYAML *yamlModels.CustomTypesYAML, maxItems int) {
	fmt.Printf("Custom types preview (%d total):\n", len(customTypesYAML.Spec.Types))
	for i, ctype := range customTypesYAML.Spec.Types {
		if i >= maxItems {
			fmt.Printf("  ... and %d more\n", len(customTypesYAML.Spec.Types)-maxItems)
			break
		}
		fmt.Printf("  - %s (%s)\n", ctype.Name, ctype.Type)
	}
}

// NewSchemaConverter creates a new schema converter
func NewSchemaConverter(options ConversionOptions) *SchemaConverter {
	// Set default YAML indent if not specified or invalid
	if options.YAMLIndent <= 0 {
		options.YAMLIndent = 2
	}

	return &SchemaConverter{
		analyzer: NewSchemaAnalyzer(),
		options:  options,
	}
}

// Convert performs the complete conversion process
func (sc *SchemaConverter) Convert() (*ConversionResult, error) {
	if sc.options.Verbose {
		fmt.Printf("Starting conversion of %s to %s\n", sc.options.InputFile, sc.options.OutputDir)
	}

	// Read and parse input file
	schemasFile, err := readSchemasFile(sc.options.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	if sc.options.Verbose {
		fmt.Printf("Found %d schemas to analyze\n", len(schemasFile.Schemas))
	}

	// Analyze schemas
	err = sc.analyzer.AnalyzeSchemas(schemasFile.Schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze schemas: %w", err)
	}

	if sc.options.Verbose {
		fmt.Printf("Analysis complete: %d events, %d properties, %d custom types\n",
			len(sc.analyzer.Events), len(sc.analyzer.Properties), len(sc.analyzer.CustomTypes))
	}

	// Generate YAML structures
	eventsYAML := sc.analyzer.GenerateEventsYAML()
	propertiesYAML := sc.analyzer.GeneratePropertiesYAML()
	customTypesYAML := sc.analyzer.GenerateCustomTypesYAML()
	trackingPlansYAML := sc.analyzer.GenerateTrackingPlansYAML(schemasFile.Schemas)

	result := &ConversionResult{
		EventsCount:      len(eventsYAML.Spec.Events),
		PropertiesCount:  len(propertiesYAML.Spec.Properties),
		CustomTypesCount: len(customTypesYAML.Spec.Types),
	}

	if sc.options.DryRun {
		return sc.performDryRun(eventsYAML, propertiesYAML, customTypesYAML, trackingPlansYAML, result)
	}

	// Create output directory
	err = os.MkdirAll(sc.options.OutputDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write YAML files
	err = sc.writeYAMLFiles(eventsYAML, propertiesYAML, customTypesYAML, trackingPlansYAML, result)
	if err != nil {
		return nil, fmt.Errorf("failed to write YAML files: %w", err)
	}

	if sc.options.Verbose {
		fmt.Printf("Conversion completed successfully!\n")
		fmt.Printf("Generated %d files in %s\n", len(result.GeneratedFiles), sc.options.OutputDir)
	}

	return result, nil
}

// performDryRun shows what would be generated without writing files
func (sc *SchemaConverter) performDryRun(eventsYAML *yamlModels.EventsYAML, propertiesYAML *yamlModels.PropertiesYAML,
	customTypesYAML *yamlModels.CustomTypesYAML, trackingPlansYAML map[string]*yamlModels.TrackingPlanYAML,
	result *ConversionResult) (*ConversionResult, error) {

	fmt.Printf("DRY RUN: Would generate the following files in %s:\n", sc.options.OutputDir)
	fmt.Printf("  ✓ events.yaml (%d events)\n", len(eventsYAML.Spec.Events))
	fmt.Printf("  ✓ properties.yaml (%d properties)\n", len(propertiesYAML.Spec.Properties))
	fmt.Printf("  ✓ custom-types.yaml (%d custom types)\n", len(customTypesYAML.Spec.Types))

	fmt.Printf("  ✓ tracking-plans/ directory\n")

	for writeKey := range trackingPlansYAML {
		planFile := fmt.Sprintf("writekey-%s.yaml", writeKey)
		fmt.Printf("    ✓ %s\n", planFile)
		result.TrackingPlans = append(result.TrackingPlans, planFile)
	}

	if sc.options.Verbose {
		fmt.Printf("\nDRY RUN: Preview of events.yaml:\n")
		printEventsPreview(eventsYAML, 3)

		fmt.Printf("\nDRY RUN: Preview of first few properties:\n")
		printPropertiesPreview(propertiesYAML, 3)

		if len(customTypesYAML.Spec.Types) > 0 {
			fmt.Printf("\nDRY RUN: Preview of first custom type:\n")
			printCustomTypesPreview(customTypesYAML, 1)
		}
	}

	return result, nil
}

// writeYAMLFiles writes all generated YAML files to disk
func (sc *SchemaConverter) writeYAMLFiles(eventsYAML *yamlModels.EventsYAML, propertiesYAML *yamlModels.PropertiesYAML,
	customTypesYAML *yamlModels.CustomTypesYAML, trackingPlansYAML map[string]*yamlModels.TrackingPlanYAML,
	result *ConversionResult) error {

	// Write events.yaml
	eventsFile := filepath.Join(sc.options.OutputDir, "events.yaml")
	err := sc.writeYAMLFile(eventsFile, eventsYAML)
	if err != nil {
		return fmt.Errorf("failed to write events.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, eventsFile)

	// Write properties.yaml
	propertiesFile := filepath.Join(sc.options.OutputDir, "properties.yaml")
	err = sc.writeYAMLFile(propertiesFile, propertiesYAML)
	if err != nil {
		return fmt.Errorf("failed to write properties.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, propertiesFile)

	// Write custom-types.yaml
	customTypesFile := filepath.Join(sc.options.OutputDir, "custom-types.yaml")
	err = sc.writeYAMLFile(customTypesFile, customTypesYAML)
	if err != nil {
		return fmt.Errorf("failed to write custom-types.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, customTypesFile)

	// Create tracking-plans directory
	trackingPlansDir := filepath.Join(sc.options.OutputDir, "tracking-plans")
	err = os.MkdirAll(trackingPlansDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create tracking-plans directory: %w", err)
	}

	// Write tracking plan files
	for writeKey, trackingPlan := range trackingPlansYAML {
		planFile := filepath.Join(trackingPlansDir, fmt.Sprintf("writekey-%s.yaml", writeKey))
		err = sc.writeYAMLFile(planFile, trackingPlan)
		if err != nil {
			return fmt.Errorf("failed to write tracking plan %s: %w", planFile, err)
		}
		result.GeneratedFiles = append(result.GeneratedFiles, planFile)
		result.TrackingPlans = append(result.TrackingPlans, fmt.Sprintf("writekey-%s.yaml", writeKey))
	}

	return nil
}

// writeYAMLFile writes a YAML structure to a file
func (sc *SchemaConverter) writeYAMLFile(filename string, data interface{}) error {
	return writeYAMLFile(filename, data, sc.options.YAMLIndent)
}
