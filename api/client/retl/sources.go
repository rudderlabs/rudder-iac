package retl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// CreateRetlSource creates a new RETL source
func (r *RudderRETLStore) CreateRetlSource(ctx context.Context, source *RETLSource) (*RETLSource, error) {
	// Create a copy to avoid modifying the input and remove fields that should not be in request
	src := *source
	src.ID = ""
	src.CreatedAt = nil
	src.UpdatedAt = nil

	data, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("marshalling source: %w", err)
	}

	resp, err := r.client.Do(ctx, "POST", "/retl-sources", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating RETL source: %w", err)
	}

	var result struct {
		Source *RETLSource `json:"source"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return result.Source, nil
}

// UpdateRetlSource updates an existing RETL source
func (r *RudderRETLStore) UpdateRetlSource(ctx context.Context, source *RETLSource) (*RETLSource, error) {
	if source.ID == "" {
		return nil, fmt.Errorf("source ID cannot be empty")
	}

	// Create a copy to avoid modifying the input and remove fields that should not be in request
	src := *source
	src.CreatedAt = nil
	src.UpdatedAt = nil

	data, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("marshalling source: %w", err)
	}

	path := fmt.Sprintf("%s/%s", "/retl-sources", source.ID)
	resp, err := r.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating RETL source: %w", err)
	}

	var result struct {
		Source *RETLSource `json:"source"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return result.Source, nil
}

// DeleteRetlSource deletes a RETL source by ID
func (r *RudderRETLStore) DeleteRetlSource(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("source ID cannot be empty")
	}

	path := fmt.Sprintf("retl-sources/%s", id)
	_, err := r.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting RETL source: %w", err)
	}

	return nil
}

// GetRetlSource retrieves a RETL source by ID
func (r *RudderRETLStore) GetRetlSource(ctx context.Context, id string) (*RETLSource, error) {
	if id == "" {
		return nil, fmt.Errorf("source ID cannot be empty")
	}

	path := fmt.Sprintf("retl-sources/%s", id)
	resp, err := r.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting RETL source: %w", err)
	}

	var result struct {
		Source *RETLSource `json:"source"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return result.Source, nil
}

// ListRetlSources lists all RETL sources
func (r *RudderRETLStore) ListRetlSources(ctx context.Context) (*RETLSources, error) {
	resp, err := r.client.Do(ctx, "GET", "/retl-sources", nil)
	if err != nil {
		return nil, fmt.Errorf("listing RETL sources: %w", err)
	}

	var sources RETLSources
	if err := json.Unmarshal(resp, &sources); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &sources, nil
}
