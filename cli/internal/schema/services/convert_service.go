package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/interfaces"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"gopkg.in/yaml.v3"
)

// ConvertService implements the SchemaConverter interface
type ConvertService struct {
	logger logger.Logger
	stats  interfaces.ConversionStats
}

// NewConvertService creates a new convert service
func NewConvertService(log logger.Logger) interfaces.SchemaConverter {
	return &ConvertService{
		logger: log,
		stats:  interfaces.ConversionStats{},
	}
}

// ConvertToYAML converts schemas to RudderStack Data Catalog YAML files
func (cs *ConvertService) ConvertToYAML(ctx context.Context, input *models.SchemasFile, opts interfaces.ConversionOptions) (*interfaces.ConversionResult, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		cs.stats.TotalConversions++
		cs.updateAverageDuration(duration)
	}()

	if opts.Verbose {
		cs.logger.Info(fmt.Sprintf("Starting conversion of %d schemas to %s", len(input.Schemas), opts.OutputDir))
	}

	// Create analyzer and process schemas
	analyzer := converter.NewSchemaAnalyzer()
	err := analyzer.AnalyzeSchemas(input.Schemas)
	if err != nil {
		cs.stats.ErrorCount++
		cs.stats.LastError = err
		return nil, fmt.Errorf("failed to analyze schemas: %w", err)
	}

	if opts.Verbose {
		cs.logger.Info(fmt.Sprintf("Analysis complete: %d events, %d properties, %d custom types",
			len(analyzer.Events), len(analyzer.Properties), len(analyzer.CustomTypes)))
	}

	// Generate YAML structures
	eventsYAML := analyzer.GenerateEventsYAML()
	propertiesYAML := analyzer.GeneratePropertiesYAML()
	customTypesYAML := analyzer.GenerateCustomTypesYAML()
	trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(input.Schemas)

	result := &interfaces.ConversionResult{
		EventsCount:      len(eventsYAML.Spec.Events),
		PropertiesCount:  len(propertiesYAML.Spec.Properties),
		CustomTypesCount: len(customTypesYAML.Spec.Types),
		Duration:         time.Since(startTime),
	}

	// Collect tracking plan names
	for writeKey := range trackingPlansYAML {
		result.TrackingPlans = append(result.TrackingPlans, writeKey)
	}

	if opts.DryRun {
		return cs.performDryRun(eventsYAML, propertiesYAML, customTypesYAML, trackingPlansYAML, result, opts)
	}

	// Create output directory
	err = os.MkdirAll(opts.OutputDir, 0755)
	if err != nil {
		cs.stats.ErrorCount++
		cs.stats.LastError = err
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write YAML files
	err = cs.writeYAMLFiles(eventsYAML, propertiesYAML, customTypesYAML, trackingPlansYAML, result, opts)
	if err != nil {
		cs.stats.ErrorCount++
		cs.stats.LastError = err
		return nil, fmt.Errorf("failed to write YAML files: %w", err)
	}

	cs.stats.SuccessCount++

	if opts.Verbose {
		cs.logger.Info(fmt.Sprintf("Conversion completed successfully! Generated %d files in %s", len(result.GeneratedFiles), opts.OutputDir))
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("conversion operation cancelled: %w", ctx.Err())
	default:
		return result, nil
	}
}

// performDryRun handles dry run mode with previews
func (cs *ConvertService) performDryRun(eventsYAML *yamlModels.EventsYAML, propertiesYAML *yamlModels.PropertiesYAML, customTypesYAML *yamlModels.CustomTypesYAML, trackingPlansYAML map[string]*yamlModels.TrackingPlanYAML, result *interfaces.ConversionResult, opts interfaces.ConversionOptions) (*interfaces.ConversionResult, error) {
	cs.logger.Info("DRY RUN: Would generate the following files:")
	cs.logger.Info(fmt.Sprintf("  - events.yaml (%d events)", len(eventsYAML.Spec.Events)))
	cs.logger.Info(fmt.Sprintf("  - properties.yaml (%d properties)", len(propertiesYAML.Spec.Properties)))
	cs.logger.Info(fmt.Sprintf("  - custom-types.yaml (%d types)", len(customTypesYAML.Spec.Types)))
	cs.logger.Info(fmt.Sprintf("  - tracking-plans/ (%d files)", len(trackingPlansYAML)))

	// Show previews
	cs.printEventsPreview(eventsYAML, 3)
	cs.printPropertiesPreview(propertiesYAML, 3)
	cs.printCustomTypesPreview(customTypesYAML, 3)

	return result, nil
}

// writeYAMLFiles writes all YAML files to the output directory
func (cs *ConvertService) writeYAMLFiles(eventsYAML *yamlModels.EventsYAML, propertiesYAML *yamlModels.PropertiesYAML, customTypesYAML *yamlModels.CustomTypesYAML, trackingPlansYAML map[string]*yamlModels.TrackingPlanYAML, result *interfaces.ConversionResult, opts interfaces.ConversionOptions) error {
	// Set default indent
	indent := opts.YAMLIndent
	if indent <= 0 {
		indent = 2
	}

	// Write events.yaml
	eventsFile := filepath.Join(opts.OutputDir, "events.yaml")
	if err := cs.writeYAMLFile(eventsFile, eventsYAML, indent); err != nil {
		return fmt.Errorf("failed to write events.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, eventsFile)

	// Write properties.yaml
	propertiesFile := filepath.Join(opts.OutputDir, "properties.yaml")
	if err := cs.writeYAMLFile(propertiesFile, propertiesYAML, indent); err != nil {
		return fmt.Errorf("failed to write properties.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, propertiesFile)

	// Write custom-types.yaml
	customTypesFile := filepath.Join(opts.OutputDir, "custom-types.yaml")
	if err := cs.writeYAMLFile(customTypesFile, customTypesYAML, indent); err != nil {
		return fmt.Errorf("failed to write custom-types.yaml: %w", err)
	}
	result.GeneratedFiles = append(result.GeneratedFiles, customTypesFile)

	// Create tracking-plans directory
	trackingPlansDir := filepath.Join(opts.OutputDir, "tracking-plans")
	if err := os.MkdirAll(trackingPlansDir, 0755); err != nil {
		return fmt.Errorf("failed to create tracking-plans directory: %w", err)
	}

	// Write tracking plan files
	for writeKey, trackingPlan := range trackingPlansYAML {
		filename := fmt.Sprintf("%s.yaml", writeKey)
		trackingPlanFile := filepath.Join(trackingPlansDir, filename)
		if err := cs.writeYAMLFile(trackingPlanFile, trackingPlan, indent); err != nil {
			return fmt.Errorf("failed to write tracking plan %s: %w", filename, err)
		}
		result.GeneratedFiles = append(result.GeneratedFiles, trackingPlanFile)
	}

	return nil
}

// writeYAMLFile writes YAML data to a file with specified indentation
func (cs *ConvertService) writeYAMLFile(filename string, data interface{}, indent int) error {
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

// Helper methods for previews
func (cs *ConvertService) printEventsPreview(eventsYAML *yamlModels.EventsYAML, maxItems int) {
	cs.logger.Info(fmt.Sprintf("Events preview (%d total):", len(eventsYAML.Spec.Events)))
	for i, event := range eventsYAML.Spec.Events {
		if i >= maxItems {
			cs.logger.Info(fmt.Sprintf("  ... and %d more", len(eventsYAML.Spec.Events)-maxItems))
			break
		}
		cs.logger.Info(fmt.Sprintf("  - %s (%s)", event.Name, event.EventType))
	}
}

func (cs *ConvertService) printPropertiesPreview(propertiesYAML *yamlModels.PropertiesYAML, maxItems int) {
	cs.logger.Info(fmt.Sprintf("Properties preview (%d total):", len(propertiesYAML.Spec.Properties)))
	for i, prop := range propertiesYAML.Spec.Properties {
		if i >= maxItems {
			cs.logger.Info(fmt.Sprintf("  ... and %d more", len(propertiesYAML.Spec.Properties)-maxItems))
			break
		}
		cs.logger.Info(fmt.Sprintf("  - %s (%s)", prop.Name, prop.Type))
	}
}

func (cs *ConvertService) printCustomTypesPreview(customTypesYAML *yamlModels.CustomTypesYAML, maxItems int) {
	cs.logger.Info(fmt.Sprintf("Custom types preview (%d total):", len(customTypesYAML.Spec.Types)))
	for i, ctype := range customTypesYAML.Spec.Types {
		if i >= maxItems {
			cs.logger.Info(fmt.Sprintf("  ... and %d more", len(customTypesYAML.Spec.Types)-maxItems))
			break
		}
		cs.logger.Info(fmt.Sprintf("  - %s (%s)", ctype.Name, ctype.Type))
	}
}

// ValidateOutput checks if the conversion result is valid
func (cs *ConvertService) ValidateOutput(result *interfaces.ConversionResult) error {
	if result == nil {
		return fmt.Errorf("conversion result cannot be nil")
	}

	if result.EventsCount < 0 || result.PropertiesCount < 0 || result.CustomTypesCount < 0 {
		return fmt.Errorf("conversion counts cannot be negative")
	}

	if result.Duration < 0 {
		return fmt.Errorf("conversion duration cannot be negative")
	}

	return nil
}

// GetConversionStats returns statistics about recent conversion operations
func (cs *ConvertService) GetConversionStats() interfaces.ConversionStats {
	return cs.stats
}

// updateAverageDuration updates the running average duration
func (cs *ConvertService) updateAverageDuration(duration time.Duration) {
	if cs.stats.TotalConversions == 1 {
		cs.stats.AverageDuration = duration
	} else {
		// Update running average
		total := cs.stats.AverageDuration*time.Duration(cs.stats.TotalConversions-1) + duration
		cs.stats.AverageDuration = total / time.Duration(cs.stats.TotalConversions)
	}
}
