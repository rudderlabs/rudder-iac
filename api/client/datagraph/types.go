package datagraph

import (
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// DataGraph represents a data graph entity from the API
type DataGraph struct {
	ID          string     `json:"id,omitempty"`
	WorkspaceID string     `json:"workspaceId,omitempty"`
	AccountID   string     `json:"accountId,omitempty"`
	ExternalID  string     `json:"externalId,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
}

// CreateDataGraphRequest is the request body for creating a data graph
type CreateDataGraphRequest struct {
	AccountID  string `json:"accountId"`
	ExternalID string `json:"externalId,omitempty"`
}

// ListDataGraphsResponse represents the paginated response from listing data graphs
type ListDataGraphsResponse struct {
	Data   []DataGraph   `json:"data"`
	Paging client.Paging `json:"paging"`
}
