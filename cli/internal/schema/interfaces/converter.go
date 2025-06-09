package interfaces

import (
	"context"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
)

// ConversionOptions holds configuration for schema conversion
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
	Duration         time.Duration
}

// ConversionStats holds statistics about conversion operations
type ConversionStats struct {
	TotalConversions int
	SuccessCount     int
	ErrorCount       int
	AverageDuration  time.Duration
	LastError        error
}

// SchemaConverter defines the interface for converting schemas to YAML
type SchemaConverter interface {
	// ConvertToYAML converts schemas to RudderStack Data Catalog YAML files
	ConvertToYAML(ctx context.Context, input *models.SchemasFile, opts ConversionOptions) (*ConversionResult, error)

	// ValidateOutput checks if the conversion result is valid
	ValidateOutput(result *ConversionResult) error

	// GetConversionStats returns statistics about recent conversion operations
	GetConversionStats() ConversionStats
}
