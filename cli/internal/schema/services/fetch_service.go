package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/interfaces"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	schemaErrors "github.com/rudderlabs/rudder-iac/cli/pkg/schema/errors"
	pkgModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
)

// FetchService implements the SchemaFetcher interface
type FetchService struct {
	client *client.Client
	logger *logger.Logger
	stats  interfaces.FetchStats
}

// NewFetchService creates a new fetch service
func NewFetchService(apiClient *client.Client, log *logger.Logger) interfaces.SchemaFetcher {
	return &FetchService{
		client: apiClient,
		logger: log,
		stats:  interfaces.FetchStats{},
	}
}

// FetchSchemas retrieves schemas from the Event Audit API
func (fs *FetchService) FetchSchemas(ctx context.Context, opts interfaces.FetchOptions) (*models.SchemasFile, error) {
	startTime := time.Now()
	defer func() {
		fs.stats.Duration = time.Since(startTime)
	}()

	if opts.Verbose {
		fs.logger.Info("Starting schema fetch operation")
		if opts.WriteKey != "" {
			fs.logger.Info(fmt.Sprintf("Filtering by write key: %s", opts.WriteKey))
		}
	}

	// Set default page size if not specified
	pageSize := opts.PageSize
	if pageSize == 0 {
		pageSize = 100
	}

	// Set timeout if not specified
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if opts.DryRun {
		fs.logger.Info("Dry run mode: would fetch schemas from Event Audit API")
		return &models.SchemasFile{Schemas: []models.Schema{}}, nil
	}

	var allSchemas []models.Schema
	currentPage := 1
	hasNext := true

	for hasNext {
		if opts.Verbose {
			fs.logger.Info(fmt.Sprintf("Fetching page %d...", currentPage))
		}

		// Build path with query parameters
		path := "v2/schemas"
		queryParams := url.Values{}
		queryParams.Set("page", strconv.Itoa(currentPage))
		queryParams.Set("limit", strconv.Itoa(pageSize))
		if opts.WriteKey != "" {
			queryParams.Set("writeKey", opts.WriteKey)
		}

		if queryStr := queryParams.Encode(); queryStr != "" {
			path = path + "?" + queryStr
		}

		// Make API request
		response, err := fs.client.Do(ctx, "GET", path, nil)
		if err != nil {
			fs.stats.ErrorCount++
			schemaErr := schemaErrors.NewFetchError(
				schemaErrors.ErrorTypeFetchAPI,
				"Failed to fetch schemas from API",
				schemaErrors.WithCause(err),
				schemaErrors.WithOperation("fetch_schemas"),
				schemaErrors.WithWriteKey(opts.WriteKey),
				schemaErrors.WithMetadata("page", strconv.Itoa(currentPage)),
				schemaErrors.AsRetryable(),
			)
			fs.stats.LastError = schemaErr
			return nil, schemaErr
		}

		// Parse response
		var apiResponse pkgModels.SchemasResponse
		if err := json.Unmarshal(response, &apiResponse); err != nil {
			fs.stats.ErrorCount++
			fs.stats.LastError = err
			return nil, fmt.Errorf("failed to parse API response: %w", err)
		}

		// Convert to internal model
		for _, schema := range apiResponse.Results {
			allSchemas = append(allSchemas, models.Schema{
				UID:             schema.UID,
				WriteKey:        schema.WriteKey,
				EventType:       schema.EventType,
				EventIdentifier: schema.EventIdentifier,
				Schema:          schema.Schema,
				CreatedAt:       schema.CreatedAt,
				LastSeen:        schema.LastSeen,
				Count:           schema.Count,
			})
		}

		fs.stats.PagesProcessed++
		hasNext = apiResponse.HasNext
		currentPage++

		if opts.Verbose {
			fs.logger.Info(fmt.Sprintf("Fetched %d schemas from page %d", len(apiResponse.Results), currentPage-1))
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("fetch operation cancelled: %w", ctx.Err())
		default:
			// Continue
		}
	}

	fs.stats.TotalSchemas = len(allSchemas)

	if opts.Verbose {
		fs.logger.Info(fmt.Sprintf("Successfully fetched %d schemas across %d pages", len(allSchemas), fs.stats.PagesProcessed))
	}

	return &models.SchemasFile{Schemas: allSchemas}, nil
}

// ValidateConnection checks if the API connection is working
func (fs *FetchService) ValidateConnection(ctx context.Context) error {
	// Simple health check by making a minimal API request
	path := "v2/schemas?limit=1"
	_, err := fs.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return fmt.Errorf("API connection validation failed: %w", err)
	}
	return nil
}

// GetFetchStats returns statistics about recent fetch operations
func (fs *FetchService) GetFetchStats() interfaces.FetchStats {
	return fs.stats
}
