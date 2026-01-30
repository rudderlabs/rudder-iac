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

// Model represents both entity and event models from the API
// Type field differentiates between "entity" and "event"
type Model struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name"`
	Type        string     `json:"type"` // "entity" or "event"
	Description string     `json:"description,omitempty"`
	TableRef    string     `json:"tableRef"`
	DataGraphID string     `json:"dataGraphId,omitempty"` // Parent data graph ID from API
	WorkspaceID string     `json:"workspaceId,omitempty"`
	ExternalID  string     `json:"externalId,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`

	// Entity model fields (only populated when Type == "entity")
	PrimaryID string `json:"primaryId,omitempty"`
	Root      bool   `json:"root,omitempty"`

	// Event model fields (only populated when Type == "event")
	Timestamp string `json:"timestamp,omitempty"`
}

// CreateEntityModelRequest is the request body for creating an entity model
type CreateEntityModelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TableRef    string `json:"tableRef"`
	ExternalID  string `json:"externalId,omitempty"`
	PrimaryID   string `json:"primaryId"`
	Root        bool   `json:"root"`
}

// CreateEventModelRequest is the request body for creating an event model
type CreateEventModelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TableRef    string `json:"tableRef"`
	ExternalID  string `json:"externalId,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// UpdateEntityModelRequest is the request body for updating an entity model
type UpdateEntityModelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TableRef    string `json:"tableRef"`
	PrimaryID   string `json:"primaryId"`
	Root        bool   `json:"root"`
}

// UpdateEventModelRequest is the request body for updating an event model
type UpdateEventModelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TableRef    string `json:"tableRef"`
	Timestamp   string `json:"timestamp"`
}

// ListModelsResponse represents the paginated response from listing models
type ListModelsResponse struct {
	Data   []Model       `json:"data"`
	Paging client.Paging `json:"paging"`
}
