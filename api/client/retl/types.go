package retl

import (
	"time"
)

type SourceType string

type AsyncStatus string

const (
	ModelSourceType SourceType = "model"

	Pending   AsyncStatus = "pending"
	Failed    AsyncStatus = "failed"
	Completed AsyncStatus = "completed"
)

// State represents the complete RETL state
type State struct {
	Version   string                   `json:"version"`
	Resources map[string]ResourceState `json:"resources"`
}

// ResourceState represents the state of a single RETL resource
type ResourceState struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output"`
	Dependencies []string               `json:"dependencies"`
}

// PutStateRequest represents the request to update a resource state
type PutStateRequest struct {
	URN   string        `json:"urn"`
	State ResourceState `json:"state"`
}

// RETLSource represents a RETL source in the API
type RETLSource struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Config               RETLSQLModelConfig `json:"config"`
	IsEnabled            bool               `json:"enabled"`
	SourceType           SourceType         `json:"sourceType"`
	SourceDefinitionName string             `json:"sourceDefinitionName"`
	AccountID            string             `json:"accountId"`
	CreatedAt            *time.Time         `json:"createdAt"`
	UpdatedAt            *time.Time         `json:"updatedAt"`
	WorkspaceID          string             `json:"workspaceId"`
	ExternalID           string             `json:"externalId"`
}

type RETLSourceCreateRequest struct {
	Name                 string             `json:"name"`
	Config               RETLSQLModelConfig `json:"config"`
	SourceType           SourceType         `json:"sourceType"`
	SourceDefinitionName string             `json:"sourceDefinitionName"`
	AccountID            string             `json:"accountId"`
	Enabled              bool               `json:"enabled"`
	ExternalID           string             `json:"externalId"`
}

type RETLSourceUpdateRequest struct {
	Name      string             `json:"name"`
	Config    RETLSQLModelConfig `json:"config"`
	IsEnabled bool               `json:"enabled"`
	AccountID string             `json:"accountId"`
}

// RETLSourceConfig represents the config of a RETL SQL model source
type RETLSQLModelConfig struct {
	PrimaryKey  string `json:"primaryKey"`
	Sql         string `json:"sql"`
	Description string `json:"description,omitempty"`
}

// RETLSources represents a response of RETL sources
type RETLSources struct {
	Data []RETLSource `json:"data"`
}

// PreviewResultError represents an error in the preview result
type PreviewResultError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// PreviewSubmitRequest represents the request to submit a RETL source preview
type PreviewSubmitRequest struct {
	AccountID string `json:"accountId"`
	Limit     int    `json:"limit,omitempty"`
	SQL       string `json:"sql"`
}

// PreviewSubmitResponse represents the response from submitting a RETL source preview
type PreviewSubmitResponse struct {
	ID string `json:"id"`
}

// PreviewResultResponse represents the response containing preview results
type PreviewResultResponse struct {
	Status AsyncStatus      `json:"status"`
	Rows   []map[string]any `json:"rows,omitempty"`
	Error  string           `json:"error,omitempty"`
}
