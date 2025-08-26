package sqlmodel

import (
	"context"
	"fmt"
	"time"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

const (
	// DefaultTimeout is the default timeout for preview operations
	DefaultTimeout = 60 * time.Second
	// DefaultPollInterval is the default interval for polling preview results
	DefaultPollInterval = 5 * time.Second
)

// Preview submits a preview request for an SQL model and polls for results
// Returns:
// - []string: column names
// - map[string]any: contains result data with keys: "errorMessage", "rows", "rowCount", and "columns" (array of column info)
// - error: any error that occurred
func (h *Handler) Preview(ctx context.Context, ID string, data resources.ResourceData, limit int) ([]string, []map[string]any, error) {
	// Extract SQL and other data from ResourceData
	sql, ok := data[SQLKey].(string)
	if !ok {
		return nil, nil, fmt.Errorf("SQL not found in resource data")
	}

	accountID, ok := data[AccountIDKey].(string)
	if !ok {
		return nil, nil, fmt.Errorf("account ID not found in resource data")
	}

	// Create preview request
	previewReq := &retlClient.PreviewSubmitRequest{
		SQL:          sql,
		AccountID:    accountID,
		FetchColumns: true,
		FetchRows:    true,
		RowLimit:     limit,
	}

	// Submit preview request
	sourceDefinition, ok := data[SourceDefinitionKey].(string)
	if !ok {
		return nil, nil, fmt.Errorf("source definition not found in resource data")
	}

	submitResp, err := h.client.SubmitSourcePreview(ctx, sourceDefinition, previewReq)
	if err != nil {
		return nil, nil, fmt.Errorf("submitting preview request: %w", err)
	}

	if !submitResp.Success {
		errMsg := "unknown error"
		if submitResp.Data.Error != nil {
			errMsg = submitResp.Data.Error.Message
		}
		return nil, nil, fmt.Errorf("preview request failed: %s", errMsg)
	}

	requestID := submitResp.Data.RequestID

	// Poll for results with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	ticker := time.NewTicker(DefaultPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, nil, fmt.Errorf("preview timed out after %s", DefaultTimeout)
		case <-ticker.C:
			resultResp, err := h.client.GetSourcePreviewResult(ctx, sourceDefinition, requestID)
			if err != nil {
				return nil, nil, fmt.Errorf("getting preview results: %w", err)
			}

			// Check if still in progress
			if resultResp.Data.State == "RUNNING" {
				continue
			}

			// Process the result
			success := resultResp.Success && resultResp.Data.Result.Success

			if !success {
				errMsg := "unknown error"
				if resultResp.Data.Result.ErrorDetails != nil {
					errMsg = resultResp.Data.Result.ErrorDetails.Message
				}
				return nil, nil, fmt.Errorf("preview request failed: %s", errMsg)
			}

			// Extract data if available
			var columnNames []string
			var rows []map[string]any

			if resultResp.Data.Result.Data != nil {
				// Convert columns
				if resultResp.Data.Result.Data.Columns != nil {
					columnNames = make([]string, len(resultResp.Data.Result.Data.Columns))
					for i, col := range resultResp.Data.Result.Data.Columns {
						columnNames[i] = col.Name
					}
				}
				rows = resultResp.Data.Result.Data.Rows
			}

			return columnNames, rows, nil
		}
	}
}
