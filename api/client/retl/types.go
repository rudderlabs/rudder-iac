package retl

import (
	"time"
)

type SourceType string

const (
	ModelSourceType SourceType = "model"
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
}

type RETLSourceCreateRequest struct {
	Name                 string             `json:"name"`
	Config               RETLSQLModelConfig `json:"config"`
	SourceType           SourceType         `json:"sourceType"`
	SourceDefinitionName string             `json:"sourceDefinitionName"`
	AccountID            string             `json:"accountId"`
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
	AccountID    string `json:"accountId"`
	FetchRows    bool   `json:"fetchRows"`
	FetchColumns bool   `json:"fetchColumns"`
	RowLimit     int    `json:"rowLimit"`
	SQL          string `json:"sql"`
}

// PreviewSubmitResponse represents the response from submitting a RETL source preview
type PreviewSubmitResponse struct {
	Data struct {
		RequestID string              `json:"requestId"`
		Error     *PreviewResultError `json:"error,omitempty"`
	} `json:"data"`
	Success bool `json:"success"`
}

// PreviewResultResponse represents the response containing preview results
type PreviewResultResponse struct {
	Data struct {
		State  string `json:"state"`
		Result struct {
			Success      bool                `json:"success"`
			ErrorDetails *PreviewResultError `json:"errorDetails,omitempty"`
			Data         *struct {
				Columns  []PreviewColumn  `json:"columns"`
				Rows     []map[string]any `json:"rows"`
				RowCount int              `json:"rowCount"`
			} `json:"data,omitempty"`
		} `json:"result"`
	} `json:"data"`
	Success bool `json:"success"`
}

// PreviewColumn represents a column in the preview results
type PreviewColumn struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	RawType string `json:"rawType"`
}
