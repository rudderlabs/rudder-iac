package interfaces

import (
	"context"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
)

// ProcessOptions holds configuration for schema processing
type ProcessOptions struct {
	JSONPath   string
	SkipFailed bool
	Verbose    bool
	Indent     int
	DryRun     bool
}

// ProcessingStats holds statistics about the processing operation
type ProcessingStats struct {
	SchemasProcessed int
	SchemasSkipped   int
	ErrorCount       int
	Duration         time.Duration
	LastError        error
}

// SchemaProcessor defines the interface for processing and transforming schemas
type SchemaProcessor interface {
	// ProcessSchemas transforms schemas (unflatten, JSONPath extraction)
	ProcessSchemas(ctx context.Context, input *models.SchemasFile, opts ProcessOptions) (*models.SchemasFile, error)

	// ValidateInput checks if the input schemas are valid for processing
	ValidateInput(input *models.SchemasFile) error

	// GetProcessingStats returns statistics about recent processing operations
	GetProcessingStats() ProcessingStats
}
