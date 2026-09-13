package sqlmodel

import (
	"context"
	"fmt"
	"time"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

const (
	// DefaultTimeout is the default timeout for preview operations
	DefaultTimeout = 60 * time.Second
	// DefaultPollInterval is the default interval for polling preview results
	DefaultPollInterval = 1 * time.Second
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

	// Preview reads the project graph without remote state, so the remote id a
	// referenced account resolves to is not known here.
	if _, ok := data[AccountIDKey].(*resources.PropertyRef); ok {
		return nil, fmt.Errorf("preview does not support sql models that reference their account yet: set account_id on %s to preview it", ID)
	}
	accountID, ok := data[AccountIDKey].(string)
	if !ok {
		return nil, fmt.Errorf("account ID not found in resource data")
	}

	return PreviewQuery(ctx, h.client, accountID, sql, limit)
}

// PreviewQuery runs sql against the account's warehouse through the preview API
// and polls for the resulting rows. It is shared with table sources, whose
// preview is a query derived from the table. If limit is 0, the query is
// validated without returning data.
func PreviewQuery(ctx context.Context, client retlClient.RETLStore, accountID, sql string, limit int) ([]map[string]any, error) {
	previewReq := &retlClient.PreviewSubmitRequest{
		SQL:       sql,
		AccountID: accountID,
		Limit:     limit,
	}

	submitResp, err := client.SubmitSourcePreview(ctx, previewReq)
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
			resultResp, err := client.GetSourcePreviewResult(ctx, requestID)
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
