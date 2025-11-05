package typer

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

var (
	rudderTyperLog = logger.New("typer.ruddertyper")
)

// PlanProvider defines the interface for retrieving tracking plan data
// This interface is defined by RudderTyper to specify exactly what it needs
// from its dependency, following the dependency inversion principle
type PlanProvider interface {
	GetTrackingPlan(ctx context.Context) (*plan.TrackingPlan, error)
}

// RudderTyper coordinates the RudderTyper code generation process
type RudderTyper struct {
	planProvider PlanProvider
}

// NewRudderTyper creates a new RudderTyper instance
func NewRudderTyper(planProvider PlanProvider) *RudderTyper {
	return &RudderTyper{
		planProvider: planProvider,
	}
}

// Generate orchestrates the complete code generation process
func (rt *RudderTyper) Generate(ctx context.Context, options core.GenerateOptions) error {
	rudderTyperLog.Debug("starting code generation",
		"platform", options.Platform,
		"outputPath", options.OutputPath)

	generator, err := generator.GeneratorForPlatform(options.Platform)
	if err != nil {
		return err
	}

	// parse platform-specific options
	platformOptions := generator.DefaultOptions()

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &platformOptions,
		WeaklyTypedInput: true,
		ErrorUnused:      true, // Returns error for unknown options
	})
	if err != nil {
		return err
	}

	if err := decoder.Decode(options.PlatformOptions); err != nil {
		return formatOptionsError(generator, options)
	}

	// Step 1: Fetch tracking plan data
	fmt.Println("📥 Fetching tracking plan data...")
	trackingPlan, err := rt.fetchTrackingPlan(ctx)
	if err != nil {
		return fmt.Errorf("fetching tracking plan: %w", err)
	}

	// Step 2: Generate platform-specific code
	fmt.Printf("⚡ Generating %s code...\n", options.Platform)
	files, err := generator.Generate(trackingPlan, options, platformOptions)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}
	rudderTyperLog.Debug("generated files", "count", len(files))

	// Step 3: Write files to output directory (only if there are files to write)
	if len(files) > 0 {
		fmt.Printf("📁 Writing files to %s...\n", options.OutputPath)
		err = rt.writeGeneratedFiles(files, options.OutputPath)
		if err != nil {
			return fmt.Errorf("writing generated files: %w", err)
		}
	} else {
		rudderTyperLog.Debug("No files to write, skipping file operations")
	}

	fmt.Printf("✅ Successfully generated %s bindings in %s\n", options.Platform, options.OutputPath)

	rudderTyperLog.Info("successfully generated code", "platform", options.Platform, "outputPath", options.OutputPath)
	return nil
}

// fetchTrackingPlan retrieves tracking plan data from the provider
func (rt *RudderTyper) fetchTrackingPlan(ctx context.Context) (*plan.TrackingPlan, error) {
	rudderTyperLog.Debug("fetching tracking plan")

	// Use the injected PlanProvider to get tracking plan data
	plan, err := rt.planProvider.GetTrackingPlan(ctx)
	if err != nil {
		rudderTyperLog.Debug("failed to fetch tracking plan", "error", err)
		return nil, fmt.Errorf("failed to get tracking plan: %w", err)
	}

	rudderTyperLog.Debug("tracking plan fetched successfully", "name", plan.Name, "rulesCount", len(plan.Rules))
	return plan, nil
}

// writeGeneratedFiles writes the generated files to the output directory
func (rt *RudderTyper) writeGeneratedFiles(files []*core.File, outputPath string) error {
	rudderTyperLog.Debug("writing generated files", "filesCount", len(files), "outputPath", outputPath)

	if len(files) == 0 {
		rudderTyperLog.Debug("no files to write, returning early")
		return nil
	}

	rudderTyperLog.Debug("creating FileManager for atomic operations", "outputPath", outputPath)
	fileManager := core.NewFileManager(outputPath)

	rudderTyperLog.Debug("would call fileManager.WriteFiles() for atomic batch write")
	return fileManager.WriteFiles(files)
}

// formatOptionsError formats a user-friendly error message for invalid platform options
func formatOptionsError(gen core.Generator, options core.GenerateOptions) error {
	// Get available options from the generator
	defaults := gen.DefaultOptions()

	// Build a list of available options by reflecting on the defaults struct
	var availableOptions []string
	defaultsType := reflect.TypeOf(defaults)
	for i := 0; i < defaultsType.NumField(); i++ {
		field := defaultsType.Field(i)
		optionName := field.Tag.Get("mapstructure")
		if optionName != "" {
			availableOptions = append(availableOptions, optionName)
		}
	}

	for o := range options.PlatformOptions {
		if !slices.Contains(availableOptions, o) {
			return fmt.Errorf("unknown option '%s' for platform '%s'", o, options.Platform)
		}
	}

	return nil
}
