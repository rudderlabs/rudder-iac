package datagraph

import (
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// DataGraph represents a data graph entity from the API
type DataGraph struct {
	ID                 string     `json:"id,omitempty"`
	Name               string     `json:"name"`
	WorkspaceID        string     `json:"workspaceId,omitempty"`
	WarehouseAccountID string     `json:"warehouseAccountId,omitempty"`
	ExternalID         string     `json:"externalId,omitempty"`
	CreatedAt          *time.Time `json:"createdAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
}

// CreateDataGraphRequest is the request body for creating a data graph
type CreateDataGraphRequest struct {
	Name               string `json:"name"`
	WarehouseAccountID string `json:"warehouseAccountId"`
}

// UpdateDataGraphRequest is the request body for updating a data graph
type UpdateDataGraphRequest struct {
	Name string `json:"name"`
}

// ListDataGraphsResponse represents the paginated response from listing data graphs
type ListDataGraphsResponse struct {
	Data   []DataGraph   `json:"data"`
	Paging client.Paging `json:"paging"`
}
