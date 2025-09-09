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
// If limit is 0, the request will be validated without returning data
func (h *Handler) Preview(ctx context.Context, ID string, data resources.ResourceData, limit int) ([]map[string]any, error) {
	// Extract SQL and other data from ResourceData
	sql, ok := data[SQLKey].(string)
	if !ok {
		return nil, fmt.Errorf("SQL not found in resource data")
	}

	accountID, ok := data[AccountIDKey].(string)
	if !ok {
		return nil, fmt.Errorf("account ID not found in resource data")
	}

	sourceDefinition, ok := data[SourceDefinitionKey].(string)
	if !ok {
		return nil, fmt.Errorf("source definition not found in resource data")
	}

	// Create preview request
	previewReq := &retlClient.PreviewSubmitRequest{
		SQL:              sql,
		AccountID:        accountID,
		Limit:            limit,
		SourceDefinition: sourceDefinition,
	}

	submitResp, err := h.client.SubmitSourcePreview(ctx, previewReq)
	if err != nil {
		return nil, fmt.Errorf("submitting preview request: %w", err)
	}

	requestID := submitResp.ID

	// Poll for results with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	ticker := time.NewTicker(DefaultPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("preview timed out after %s", DefaultTimeout)
		case <-ticker.C:
			resultResp, err := h.client.GetSourcePreviewResult(ctx, requestID)
			if err != nil {
				return nil, fmt.Errorf("getting preview results: %w", err)
			}

			status := resultResp.Status
			switch status {
			case retlClient.Pending:
				continue
			case retlClient.Failed:
				return nil, fmt.Errorf("preview request failed: %s", resultResp.Error)
			case retlClient.Completed:
				return resultResp.Rows, nil
			}
		}
	}
}
