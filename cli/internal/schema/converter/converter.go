package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/experimental/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/pkg/experimental/schema/utils"
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
	schemasFile, err := utils.ReadSchemasFile(sc.options.InputFile)
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
		sc.printYAMLPreview(eventsYAML, 3)

		fmt.Printf("\nDRY RUN: Preview of first few properties:\n")
		sc.printPropertiesPreview(propertiesYAML, 3)

		if len(customTypesYAML.Spec.Types) > 0 {
			fmt.Printf("\nDRY RUN: Preview of first custom type:\n")
			sc.printCustomTypesPreview(customTypesYAML, 1)
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
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(sc.options.YAMLIndent)
	defer encoder.Close()

	err = encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode YAML to %s: %w", filename, err)
	}

	if sc.options.Verbose {
		fmt.Printf("  ✓ Created %s\n", filename)
	}

	return nil
}

// printYAMLPreview prints a preview of YAML content
func (sc *SchemaConverter) printYAMLPreview(data interface{}, maxItems int) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		fmt.Printf("Error generating preview: %v\n", err)
		return
	}

	lines := strings.Split(string(yamlData), "\n")
	count := 0
	for _, line := range lines {
		if count >= 20 { // Limit preview length
			fmt.Printf("    ... (truncated)\n")
			break
		}
		fmt.Printf("    %s\n", line)
		count++
	}
}

// printPropertiesPreview prints a preview of properties
func (sc *SchemaConverter) printPropertiesPreview(propertiesYAML *yamlModels.PropertiesYAML, maxItems int) {
	fmt.Printf("    version: %s\n", propertiesYAML.Version)
	fmt.Printf("    kind: %s\n", propertiesYAML.Kind)
	fmt.Printf("    spec:\n")
	fmt.Printf("      properties:\n")

	count := 0
	for _, prop := range propertiesYAML.Spec.Properties {
		if count >= maxItems {
			fmt.Printf("        ... (showing %d of %d properties)\n", maxItems, len(propertiesYAML.Spec.Properties))
			break
		}
		fmt.Printf("        - id: %s\n", prop.ID)
		fmt.Printf("          name: %s\n", prop.Name)
		fmt.Printf("          type: %s\n", prop.Type)
		if prop.Description != "" {
			fmt.Printf("          description: %s\n", prop.Description)
		}
		count++
	}
}

// printCustomTypesPreview prints a preview of custom types
func (sc *SchemaConverter) printCustomTypesPreview(customTypesYAML *yamlModels.CustomTypesYAML, maxItems int) {
	fmt.Printf("    version: %s\n", customTypesYAML.Version)
	fmt.Printf("    kind: %s\n", customTypesYAML.Kind)
	fmt.Printf("    spec:\n")
	fmt.Printf("      types:\n")

	count := 0
	for _, customType := range customTypesYAML.Spec.Types {
		if count >= maxItems {
			fmt.Printf("        ... (showing %d of %d custom types)\n", maxItems, len(customTypesYAML.Spec.Types))
			break
		}
		fmt.Printf("        - id: %s\n", customType.ID)
		fmt.Printf("          name: %s\n", customType.Name)
		fmt.Printf("          type: %s\n", customType.Type)
		if customType.Description != "" {
			fmt.Printf("          description: %s\n", customType.Description)
		}
		count++
	}
}
