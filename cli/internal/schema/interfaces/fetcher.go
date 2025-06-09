package interfaces

import (
	"context"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
)

// FetchOptions holds configuration for schema fetching
type FetchOptions struct {
	WriteKey string
	DryRun   bool
	Verbose  bool
	PageSize int
	Timeout  time.Duration
}

// FetchStats holds statistics about the fetch operation
type FetchStats struct {
	TotalSchemas   int
	PagesProcessed int
	Duration       time.Duration
	ErrorCount     int
	LastError      error
}

// SchemaFetcher defines the interface for fetching schemas from external APIs
type SchemaFetcher interface {
	// FetchSchemas retrieves schemas from the Event Audit API
	FetchSchemas(ctx context.Context, opts FetchOptions) (*models.SchemasFile, error)

	// ValidateConnection checks if the API connection is working
	ValidateConnection(ctx context.Context) error

	// GetFetchStats returns statistics about recent fetch operations
	GetFetchStats() FetchStats
}
