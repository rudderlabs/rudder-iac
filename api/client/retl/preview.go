package retl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// SubmitSourcePreview submits a request to preview a RETL source
func (r *RudderRETLStore) SubmitSourcePreview(ctx context.Context, role string, request *PreviewSubmitRequest) (*PreviewSubmitResponse, error) {
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshalling preview request: %w", err)
	}

	path := fmt.Sprintf("/v2/retl-sources/preview/role/%s/submit", role)
	resp, err := r.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("submitting source preview: %w", err)
	}

	var result PreviewSubmitResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// GetSourcePreviewResult retrieves the results of a RETL source preview
func (r *RudderRETLStore) GetSourcePreviewResult(ctx context.Context, role string, resultID string) (*PreviewResultResponse, error) {
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}

	if resultID == "" {
		return nil, fmt.Errorf("result ID cannot be empty")
	}

	path := fmt.Sprintf("/v2/retl-sources/preview/role/%s/result/%s", role, resultID)
	resp, err := r.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting source preview result: %w", err)
	}

	var result PreviewResultResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}
