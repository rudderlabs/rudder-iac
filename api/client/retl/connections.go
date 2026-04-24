package retl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const retlConnectionsBasePath = "/v2/retl-connections"

// CreateConnection creates a new RETL connection.
func (r *RudderRETLStore) CreateConnection(ctx context.Context, req *CreateRETLConnectionRequest) (*RETLConnection, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	// Schedule is required server-side; guard here so callers get a clear
	// error instead of a 400 complaining about an empty enum value.
	if req.Schedule.Type == "" {
		return nil, fmt.Errorf("schedule.type is required")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling connection: %w", err)
	}

	resp, err := r.client.Do(ctx, "POST", retlConnectionsBasePath, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating RETL connection: %w", err)
	}

	var result RETLConnection
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// UpdateConnection updates mutable fields of a RETL connection.
func (r *RudderRETLStore) UpdateConnection(ctx context.Context, id string, req *UpdateRETLConnectionRequest) (*RETLConnection, error) {
	if id == "" {
		return nil, fmt.Errorf("connection ID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	// Schedule is required server-side; guard here so callers get a clear
	// error instead of a 400 complaining about an empty enum value.
	if req.Schedule.Type == "" {
		return nil, fmt.Errorf("schedule.type is required")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling connection: %w", err)
	}

	path := fmt.Sprintf("%s/%s", retlConnectionsBasePath, id)
	resp, err := r.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating RETL connection: %w", err)
	}

	var result RETLConnection
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// DeleteConnection soft-deletes a RETL connection.
func (r *RudderRETLStore) DeleteConnection(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("connection ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", retlConnectionsBasePath, id)
	if _, err := r.client.Do(ctx, "DELETE", path, nil); err != nil {
		return fmt.Errorf("deleting RETL connection: %w", err)
	}

	return nil
}

// GetConnection retrieves a RETL connection by ID.
func (r *RudderRETLStore) GetConnection(ctx context.Context, id string) (*RETLConnection, error) {
	if id == "" {
		return nil, fmt.Errorf("connection ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", retlConnectionsBasePath, id)
	resp, err := r.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting RETL connection: %w", err)
	}

	var result RETLConnection
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// ListConnections returns a paginated list of RETL connections matching the
// provided filters.
func (r *RudderRETLStore) ListConnections(ctx context.Context, req *ListRETLConnectionsRequest) (*RETLConnectionsPage, error) {
	path := retlConnectionsBasePath
	query := url.Values{}
	if req != nil {
		if req.SourceID != "" {
			query.Add("sourceId", req.SourceID)
		}
		if req.DestinationID != "" {
			query.Add("destinationId", req.DestinationID)
		}
		if req.HasExternalID != nil {
			query.Add("hasExternalId", strconv.FormatBool(*req.HasExternalID))
		}
		if req.Page > 0 {
			query.Add("page", strconv.Itoa(req.Page))
		}
		if req.PageSize > 0 {
			query.Add("pageSize", strconv.Itoa(req.PageSize))
		}
	}

	if len(query) > 0 {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	resp, err := r.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing RETL connections: %w", err)
	}

	var result RETLConnectionsPage
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// SetConnectionExternalId sets the external ID for a RETL connection.
func (r *RudderRETLStore) SetConnectionExternalId(ctx context.Context, req *SetRETLConnectionExternalIDRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if req.ID == "" {
		return fmt.Errorf("connection ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/external-id", retlConnectionsBasePath, req.ID)
	data, err := json.Marshal(map[string]string{"externalId": req.ExternalID})
	if err != nil {
		return fmt.Errorf("marshalling external ID: %w", err)
	}

	if _, err := r.client.Do(ctx, "PUT", path, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("setting external ID: %w", err)
	}

	return nil
}
