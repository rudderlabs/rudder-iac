package datagraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/rudderlabs/rudder-iac/api/client"
)

const (
	dataGraphsBasePath = "/v2/data-graphs"
)

// DataGraphStore is the interface for Data Graph operations
type DataGraphStore interface {
	// ListDataGraphs lists all data graphs with optional pagination and filtering
	ListDataGraphs(ctx context.Context, page, pageSize int, hasExternalID *bool) (*ListDataGraphsResponse, error)

	// GetDataGraph retrieves a data graph by ID
	GetDataGraph(ctx context.Context, id string) (*DataGraph, error)

	// CreateDataGraph creates a new data graph
	CreateDataGraph(ctx context.Context, req *CreateDataGraphRequest) (*DataGraph, error)

	// DeleteDataGraph deletes a data graph by ID
	DeleteDataGraph(ctx context.Context, id string) error

	// SetExternalID sets the external ID for a data graph and returns the updated data graph
	SetExternalID(ctx context.Context, id string, externalID string) (*DataGraph, error)
}

// rudderDataGraphStore implements the DataGraphStore interface
type rudderDataGraphStore struct {
	client *client.Client
}

// NewRudderDataGraphStore creates a new DataGraphStore implementation
func NewRudderDataGraphStore(c *client.Client) DataGraphStore {
	return &rudderDataGraphStore{
		client: c,
	}
}

// ListDataGraphs lists all data graphs with optional pagination and filtering
func (s *rudderDataGraphStore) ListDataGraphs(ctx context.Context, page, pageSize int, hasExternalID *bool) (*ListDataGraphsResponse, error) {
	path := dataGraphsBasePath

	query := url.Values{}
	if page > 0 {
		query.Add("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		query.Add("pageSize", strconv.Itoa(pageSize))
	}
	if hasExternalID != nil {
		query.Add("hasExternalId", strconv.FormatBool(*hasExternalID))
	}

	if len(query) > 0 {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing data graphs: %w", err)
	}

	var result ListDataGraphsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// GetDataGraph retrieves a data graph by ID
func (s *rudderDataGraphStore) GetDataGraph(ctx context.Context, id string) (*DataGraph, error) {
	if id == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", dataGraphsBasePath, id)
	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting data graph: %w", err)
	}

	var result DataGraph
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// CreateDataGraph creates a new data graph
func (s *rudderDataGraphStore) CreateDataGraph(ctx context.Context, req *CreateDataGraphRequest) (*DataGraph, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "POST", dataGraphsBasePath, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating data graph: %w", err)
	}

	var result DataGraph
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// DeleteDataGraph deletes a data graph by ID
func (s *rudderDataGraphStore) DeleteDataGraph(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", dataGraphsBasePath, id)
	_, err := s.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting data graph: %w", err)
	}

	return nil
}

// SetExternalID sets the external ID for a data graph and returns the updated data graph
func (s *rudderDataGraphStore) SetExternalID(ctx context.Context, id string, externalID string) (*DataGraph, error) {
	if id == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/external-id", dataGraphsBasePath, id)
	data, err := json.Marshal(map[string]string{"externalId": externalID})
	if err != nil {
		return nil, fmt.Errorf("marshalling external ID: %w", err)
	}

	resp, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("setting external ID: %w", err)
	}

	var result DataGraph
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}
