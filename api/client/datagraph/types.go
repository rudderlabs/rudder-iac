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

// ListModelsRequest is the request for listing models
type ListModelsRequest struct {
	DataGraphID   string
	Page          int
	PageSize      int
	ModelType     *string // nil (all models), "entity", or "event"
	IsRoot        *bool
	HasExternalID *bool
}

// GetModelRequest is the request for getting a model
type GetModelRequest struct {
	DataGraphID string
	ModelID     string
}

// CreateModelRequest is the unified request for creating models
// Type field determines whether this is an entity or event model
type CreateModelRequest struct {
	DataGraphID string `json:"-"` // Not sent in JSON body

	Type        string `json:"type"` // "entity" or "event" - REQUIRED
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TableRef    string `json:"tableRef"`
	ExternalID  string `json:"externalId,omitempty"`

	// Entity model fields (required when Type == "entity")
	PrimaryID string `json:"primaryId,omitempty"`
	Root      bool   `json:"root,omitempty"`

	// Event model fields (required when Type == "event")
	Timestamp string `json:"timestamp,omitempty"`
}

// UpdateModelRequest is the unified request for updating models
// Type field determines whether this is an entity or event model update
type UpdateModelRequest struct {
	DataGraphID string `json:"-"` // Not sent in JSON body
	ModelID     string `json:"-"` // Not sent in JSON body

	Type        string `json:"type"` // "entity" or "event" - REQUIRED
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TableRef    string `json:"tableRef"`

	// Entity model fields (required when Type == "entity")
	PrimaryID string `json:"primaryId,omitempty"`
	Root      bool   `json:"root,omitempty"`

	// Event model fields (required when Type == "event")
	Timestamp string `json:"timestamp,omitempty"`
}

// DeleteModelRequest is the request for deleting a model
type DeleteModelRequest struct {
	DataGraphID string
	ModelID     string
}

// SetModelExternalIDRequest is the request for setting a model's external ID
type SetModelExternalIDRequest struct {
	DataGraphID string
	ModelID     string
	ExternalID  string
}

// ListModelsResponse represents the paginated response from listing models
type ListModelsResponse struct {
	Data   []Model       `json:"data"`
	Paging client.Paging `json:"paging"`
}

// Relationship represents a relationship in the data graph
type Relationship struct {
	ID            string     `json:"id,omitempty"`
	Name          string     `json:"name"`
	Cardinality   string     `json:"cardinality"`
	SourceModelID string     `json:"sourceModelId"`
	TargetModelID string     `json:"targetModelId"`
	SourceJoinKey string     `json:"sourceJoinKey"`
	TargetJoinKey string     `json:"targetJoinKey"`
	DataGraphID   string     `json:"dataGraphId,omitempty"`
	WorkspaceID   string     `json:"workspaceId,omitempty"`
	ExternalID    string     `json:"externalId,omitempty"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

// ListRelationshipsRequest is the request for listing relationships
type ListRelationshipsRequest struct {
	DataGraphID   string
	Page          int
	PageSize      int
	SourceModelID *string
	HasExternalID *bool
}

// GetRelationshipRequest is the request for getting a relationship
type GetRelationshipRequest struct {
	DataGraphID    string
	RelationshipID string
}

// CreateRelationshipRequest is the request for creating a relationship
type CreateRelationshipRequest struct {
	DataGraphID   string `json:"-"` // Path parameter
	Name          string `json:"name"`
	Cardinality   string `json:"cardinality"`
	SourceModelID string `json:"sourceModelId"`
	TargetModelID string `json:"targetModelId"`
	SourceJoinKey string `json:"sourceJoinKey"`
	TargetJoinKey string `json:"targetJoinKey"`
	ExternalID    string `json:"externalId,omitempty"`
}

// UpdateRelationshipRequest is the request for updating a relationship
type UpdateRelationshipRequest struct {
	DataGraphID    string `json:"-"` // Path parameter
	RelationshipID string `json:"-"` // Path parameter
	Name           string `json:"name"`
	Cardinality    string `json:"cardinality"`
	SourceModelID  string `json:"sourceModelId"`
	TargetModelID  string `json:"targetModelId"`
	SourceJoinKey  string `json:"sourceJoinKey"`
	TargetJoinKey  string `json:"targetJoinKey"`
}

// DeleteRelationshipRequest is the request for deleting a relationship
type DeleteRelationshipRequest struct {
	DataGraphID    string
	RelationshipID string
}

// SetRelationshipExternalIDRequest is the request for setting a relationship's external ID
type SetRelationshipExternalIDRequest struct {
	DataGraphID    string
	RelationshipID string
	ExternalID     string
}

// ListRelationshipsResponse represents the paginated response from listing relationships
type ListRelationshipsResponse struct {
	Data   []Relationship `json:"data"`
	Paging client.Paging  `json:"paging"`
}
