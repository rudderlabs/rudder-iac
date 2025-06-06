package schema

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/converter"
	"github.com/spf13/cobra"
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
