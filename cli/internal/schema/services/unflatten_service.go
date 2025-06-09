package services

import (
	"context"
	"fmt"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/interfaces"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/jsonpath"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/unflatten"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

// UnflattenService implements the SchemaProcessor interface
type UnflattenService struct {
	logger logger.Logger
	stats  interfaces.ProcessingStats
}

// NewUnflattenService creates a new unflatten service
func NewUnflattenService(log logger.Logger) interfaces.SchemaProcessor {
	return &UnflattenService{
		logger: log,
		stats:  interfaces.ProcessingStats{},
	}
}

// ProcessSchemas transforms schemas (unflatten, JSONPath extraction)
func (us *UnflattenService) ProcessSchemas(ctx context.Context, input *models.SchemasFile, opts interfaces.ProcessOptions) (*models.SchemasFile, error) {
	startTime := time.Now()
	defer func() {
		us.stats.Duration = time.Since(startTime)
	}()

	if opts.Verbose {
		us.logger.Info(fmt.Sprintf("Processing %d schemas", len(input.Schemas)))
		if opts.JSONPath != "" {
			us.logger.Info(fmt.Sprintf("Using JSONPath: %s", opts.JSONPath))
			us.logger.Info(fmt.Sprintf("Skip failed: %v", opts.SkipFailed))
		}
	}

	if opts.DryRun {
		us.logger.Info("Dry run mode: would process schemas with unflatten and JSONPath")
		return input, nil
	}

	// Create JSONPath processor if needed
	var processor *jsonpath.Processor
	if opts.JSONPath != "" {
		processor = jsonpath.NewProcessor(opts.JSONPath, opts.SkipFailed)
	}

	// Process each schema
	var processedSchemas []models.Schema
	processedCount := 0
	skippedCount := 0

	for i := range input.Schemas {
		originalSchema := &input.Schemas[i]

		// Step 1: Always unflatten first
		if len(originalSchema.Schema) > 0 {
			originalSchema.Schema = unflatten.UnflattenSchema(originalSchema.Schema)
		}

		// Step 2: Apply JSONPath extraction if specified
		if processor != nil && !processor.IsRootPath() {
			result := processor.ProcessSchema(originalSchema.Schema)
			if result.Error != nil {
				us.stats.ErrorCount++

				if opts.Verbose {
					us.logger.Info(fmt.Sprintf("JSONPath processing failed for schema %s: %v", originalSchema.UID, result.Error))
				}

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

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("processing operation cancelled: %w", ctx.Err())
		default:
			// Continue
		}
	}

	// Update statistics
	us.stats.SchemasProcessed = processedCount
	us.stats.SchemasSkipped = skippedCount

	if opts.Verbose {
		us.logger.Info(fmt.Sprintf("Processing completed: %d processed, %d skipped", processedCount, skippedCount))
	}

	// Return processed schemas
	return &models.SchemasFile{Schemas: processedSchemas}, nil
}

// ValidateInput checks if the input schemas are valid for processing
func (us *UnflattenService) ValidateInput(input *models.SchemasFile) error {
	if input == nil {
		return fmt.Errorf("input cannot be nil")
	}

	if len(input.Schemas) == 0 {
		return fmt.Errorf("no schemas to process")
	}

	// Validate each schema has required fields
	for i, schema := range input.Schemas {
		if schema.UID == "" {
			return fmt.Errorf("schema at index %d missing UID", i)
		}
		if schema.Schema == nil {
			return fmt.Errorf("schema at index %d has nil Schema field", i)
		}
	}

	return nil
}

// GetProcessingStats returns statistics about recent processing operations
func (us *UnflattenService) GetProcessingStats() interfaces.ProcessingStats {
	return us.stats
}
