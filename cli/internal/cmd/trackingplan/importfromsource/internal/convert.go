package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/converter"
	pkgModels "github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewCmdConvert creates the convert command
func NewCmdConvert() *cobra.Command {
	var (
		dryRun  bool
		verbose bool
		indent  int
	)

	cmd := &cobra.Command{
		Use:   "convert [input-file] [output-dir]",
		Short: "Convert unflattened schemas to YAML files",
		Long: `Convert unflattened schemas to RudderStack Data Catalog YAML files.

This command takes an unflattened schemas JSON file as input and generates:
- events.yaml: All unique events extracted from eventIdentifier
- properties.yaml: All properties with custom type references
- custom-types.yaml: Custom object and array types
- tracking-plans/: Individual tracking plans grouped by writeKey

The generated YAML files follow the RudderStack Data Catalog specifications
and can be used with rudder-cli for tracking plan management.

Examples:
  rudder-cli schema convert schemas.json output/
  rudder-cli schema convert schemas.json output/ --verbose --dry-run`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(args[0], args[1], dryRun, verbose, indent)
		},
	}

	// Add flags for the convert command
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without writing files")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().IntVar(&indent, "indent", 2, "Number of spaces for YAML indentation")

	return cmd
}

// runConvert handles the convert command execution
func runConvert(inputFile, outputDir string, dryRun, verbose bool, indent int) error {
	if verbose {
		fmt.Printf("Converting schemas from %s to %s...\n", inputFile, outputDir)
	}

	// Create conversion options
	options := converter.ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     dryRun,
		Verbose:    verbose,
		YAMLIndent: indent,
	}

	// Create converter and run conversion
	schemaConverter := converter.NewSchemaConverter(options)
	result, err := schemaConverter.Convert()
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Display results
	if dryRun {
		fmt.Printf("\nDRY RUN SUMMARY:\n")
	} else {
		fmt.Printf("\nCONVERSION COMPLETED SUCCESSFULLY!\n")
	}

	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("  Events: %d\n", result.EventsCount)
	fmt.Printf("  Properties: %d\n", result.PropertiesCount)
	fmt.Printf("  Custom Types: %d\n", result.CustomTypesCount)
	fmt.Printf("  Tracking Plans: %d\n", len(result.TrackingPlans))

	if !dryRun {
		fmt.Printf("\n📁 Generated Files:\n")
		for _, file := range result.GeneratedFiles {
			fmt.Printf("  ✓ %s\n", file)
		}

		fmt.Printf("\n🎯 Next Steps:\n")
		fmt.Printf("  1. Review the generated YAML files\n")
		fmt.Printf("  2. Validate with: rudder-cli tp validate -l %s\n", outputDir)
		fmt.Printf("  3. Deploy with: rudder-cli tp apply -l %s\n", outputDir)
	}

	return nil
}

// RunConvert is a public wrapper for runConvert to be used by other commands
func RunConvert(inputFile, outputDir string, dryRun, verbose bool, indent int) error {
	return runConvert(inputFile, outputDir, dryRun, verbose, indent)
}

// ConvertSchemasToYAML converts schemas directly from memory to YAML files for optimized processing
func ConvertSchemasToYAML(schemas []pkgModels.Schema, outputDir string, dryRun, verbose bool, indent int) (*converter.ConversionResult, error) {
	if verbose {
		fmt.Printf("Converting %d schemas to YAML files in memory...\n", len(schemas))
	}

	// Create analyzer and analyze schemas directly
	analyzer := converter.NewSchemaAnalyzer()
	err := analyzer.AnalyzeSchemas(schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze schemas: %w", err)
	}

	if verbose {
		fmt.Printf("Analysis complete: %d events, %d properties, %d custom types\n",
			len(analyzer.Events), len(analyzer.Properties), len(analyzer.CustomTypes))
	}

	// Generate YAML structures
	eventsYAML := analyzer.GenerateEventsYAML()
	propertiesYAML := analyzer.GeneratePropertiesYAML()
	customTypesYAML := analyzer.GenerateCustomTypesYAML()
	trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(schemas)

	result := &converter.ConversionResult{
		EventsCount:      len(eventsYAML.Spec.Events),
		PropertiesCount:  len(propertiesYAML.Spec.Properties),
		CustomTypesCount: len(customTypesYAML.Spec.Types),
	}

	if dryRun {
		return performDryRunConversion(eventsYAML, propertiesYAML, customTypesYAML, trackingPlansYAML, result, outputDir, verbose)
	}

	// Create output directory
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write YAML files
	err = writeYAMLFiles(eventsYAML, propertiesYAML, customTypesYAML, trackingPlansYAML, result, outputDir, indent)
	if err != nil {
		return nil, fmt.Errorf("failed to write YAML files: %w", err)
	}

	if verbose {
		fmt.Printf("Conversion completed successfully!\n")
		fmt.Printf("Generated %d files in %s\n", len(result.GeneratedFiles), outputDir)
	}

	return result, nil
}

// performDryRunConversion shows what would be generated without writing files
func performDryRunConversion(eventsYAML *yamlModels.EventsYAML, propertiesYAML *yamlModels.PropertiesYAML,
	customTypesYAML *yamlModels.CustomTypesYAML, trackingPlansYAML map[string]*yamlModels.TrackingPlanYAML,
	result *converter.ConversionResult, outputDir string, verbose bool) (*converter.ConversionResult, error) {

	fmt.Printf("DRY RUN: Would generate the following files in %s:\n", outputDir)
	fmt.Printf("  ✓ events.yaml (%d events)\n", len(eventsYAML.Spec.Events))
	fmt.Printf("  ✓ properties.yaml (%d properties)\n", len(propertiesYAML.Spec.Properties))
	fmt.Printf("  ✓ custom-types.yaml (%d custom types)\n", len(customTypesYAML.Spec.Types))

	fmt.Printf("  ✓ tracking-plans/ directory\n")

	for writeKey := range trackingPlansYAML {
		planFile := fmt.Sprintf("writekey-%s.yaml", writeKey)
		fmt.Printf("    ✓ %s\n", planFile)
		result.TrackingPlans = append(result.TrackingPlans, planFile)
	}

	if verbose {
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
func writeYAMLFiles(eventsYAML *yamlModels.EventsYAML, propertiesYAML *yamlModels.PropertiesYAML,
	customTypesYAML *yamlModels.CustomTypesYAML, trackingPlansYAML map[string]*yamlModels.TrackingPlanYAML,
	result *converter.ConversionResult, outputDir string, indent int) error {

	// Write events.yaml
	eventsFile := filepath.Join(outputDir, "events.yaml")
	if err := writeYAMLFile(eventsFile, eventsYAML, indent); err != nil {
		return fmt.Errorf("failed to write events.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, eventsFile)

	// Write properties.yaml
	propertiesFile := filepath.Join(outputDir, "properties.yaml")
	if err := writeYAMLFile(propertiesFile, propertiesYAML, indent); err != nil {
		return fmt.Errorf("failed to write properties.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, propertiesFile)

	// Write custom-types.yaml
	customTypesFile := filepath.Join(outputDir, "custom-types.yaml")
	if err := writeYAMLFile(customTypesFile, customTypesYAML, indent); err != nil {
		return fmt.Errorf("failed to write custom-types.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, customTypesFile)

	// Write tracking plans
	trackingPlansDir := filepath.Join(outputDir, "tracking-plans")
	if err := os.MkdirAll(trackingPlansDir, 0755); err != nil {
		return fmt.Errorf("failed to create tracking-plans directory: %w", err)
	}

	for writeKey, trackingPlan := range trackingPlansYAML {
		planFile := filepath.Join(trackingPlansDir, fmt.Sprintf("writekey-%s.yaml", writeKey))
		if err := writeYAMLFile(planFile, trackingPlan, indent); err != nil {
			return fmt.Errorf("failed to write tracking plan %s: %w", planFile, err)
		}
		result.GeneratedFiles = append(result.GeneratedFiles, planFile)
		result.TrackingPlans = append(result.TrackingPlans, fmt.Sprintf("writekey-%s.yaml", writeKey))
	}

	return nil
}

// Helper functions from the converter package
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
