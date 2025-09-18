package typer

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

var (
	orchestratorLog = logger.New("typer.orchestrator")
)

// Orchestrator coordinates the RudderTyper code generation process
type Orchestrator struct {
	deps app.Deps
}

// GenerationOptions contains configuration for code generation
type GenerationOptions struct {
	TrackingPlanID string
	Platform       string
	OutputPath     string
}

// NewOrchestrator creates a new RudderTyper orchestrator
func NewOrchestrator(deps app.Deps) *Orchestrator {
	return &Orchestrator{
		deps: deps,
	}
}

// Generate orchestrates the complete code generation process
func (o *Orchestrator) Generate(ctx context.Context, options GenerationOptions) error {
	orchestratorLog.Debug("starting code generation",
		"trackingPlanID", options.TrackingPlanID,
		"platform", options.Platform,
		"outputPath", options.OutputPath)

	// Step 1: Fetch tracking plan data
	fmt.Println("(mock) 📥 Fetching tracking plan data...")
	trackingPlan, err := o.fetchTrackingPlan(ctx, options.TrackingPlanID)
	if err != nil {
		return fmt.Errorf("fetching tracking plan: %w", err)
	}
	orchestratorLog.Debug("(mock) fetched tracking plan", "rules", len(trackingPlan.Rules))

	// Step 2: Generate platform-specific code
	fmt.Printf("⚡ Generating %s code...\n", options.Platform)
	files, err := o.generateCode(trackingPlan, options.Platform)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}
	orchestratorLog.Debug("generated files", "count", len(files))

	// Step 3: Write files to output directory (only if there are files to write)
	if len(files) > 0 {
		fmt.Printf("📁 Writing files to %s...\n", options.OutputPath)
		err = o.writeGeneratedFiles(files, options.OutputPath)
		if err != nil {
			return fmt.Errorf("writing generated files: %w", err)
		}
	} else {
		orchestratorLog.Debug("No files to write, skipping file operations")
	}

	fmt.Printf("✅ Successfully generated %s bindings in %s\n", options.Platform, options.OutputPath)

	orchestratorLog.Info("successfully generated code", "platform", options.Platform, "outputPath", options.OutputPath)
	return nil
}

// fetchTrackingPlan retrieves tracking plan data from the provider
func (o *Orchestrator) fetchTrackingPlan(ctx context.Context, trackingPlanID string) (*plan.TrackingPlan, error) {
	orchestratorLog.Debug("(mock) fetching tracking plan", "id", trackingPlanID)

	// Mock: fetch tracking plan from data catalog provider
	orchestratorLog.Debug("mock: would call o.deps.Providers().DataCatalog.GetTrackingPlan()", "id", trackingPlanID)

	// Mock: create sample tracking plan data
	mockPlan := &plan.TrackingPlan{
		Name: fmt.Sprintf("Tracking Plan %s", trackingPlanID),
		Rules: []plan.EventRule{
			{
				Event: plan.Event{
					EventType:   plan.EventTypeTrack,
					Name:        "UserSignupMockData",
					Description: "User signup event",
				},
				Section: plan.EventRuleSectionProperties,
				Schema: plan.ObjectSchema{
					Properties:           make(map[string]plan.PropertySchema),
					AdditionalProperties: false,
				},
			},
		},
	}

	orchestratorLog.Debug("mock: tracking plan fetched successfully", "name", mockPlan.Name)
	return mockPlan, nil
}

// generateCode generates platform-specific code from the tracking plan
func (o *Orchestrator) generateCode(trackingPlan *plan.TrackingPlan, platform string) ([]*core.File, error) {
	orchestratorLog.Debug("generating code for platform", "platform", platform, "rulesCount", len(trackingPlan.Rules))

	switch platform {
	case "kotlin":
		return kotlin.Generate(trackingPlan)
	default:
		return nil, fmt.Errorf("unsupported platform: %s (supported platforms: kotlin)", platform)
	}
}

// writeGeneratedFiles writes the generated files to the output directory
func (o *Orchestrator) writeGeneratedFiles(files []*core.File, outputPath string) error {
	orchestratorLog.Debug("writing generated files", "filesCount", len(files), "outputPath", outputPath)

	if len(files) == 0 {
		orchestratorLog.Debug("no files to write, returning early")
		return nil
	}

	// Convert absolute path to avoid issues
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	orchestratorLog.Debug("creating FileManager for atomic operations", "absPath", absOutputPath)
	fileManager := core.NewFileManager(absOutputPath)

	// Convert []*core.File to []core.File for WriteFiles
	coreFiles := make([]core.File, len(files))
	for i, file := range files {
		coreFiles[i] = core.File{
			Path:    file.Path,
			Content: file.Content,
		}
		orchestratorLog.Debug("prepared file for writing", "path", file.Path)
	}

	orchestratorLog.Debug("would call fileManager.WriteFiles() for atomic batch write")
	return fileManager.WriteFiles(coreFiles)
}
