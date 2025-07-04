package retl

import (
	"encoding/json"
	"time"
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
	ID    string        `json:"id"`
	URN   string        `json:"urn"`
	State ResourceState `json:"state"`
}

// RETLSource represents a RETL source in the API
type RETLSource struct {
	ID                   string          `json:"id,omitempty"`
	Name                 string          `json:"name"`
	Config               json.RawMessage `json:"config"`
	IsEnabled            bool            `json:"enabled"`
	SourceType           string          `json:"sourceType"`
	SourceDefinitionName string          `json:"sourceDefinitionName"`
	AccountID            string          `json:"accountId"`
	PrimaryKey           string          `json:"primaryKey,omitempty"`
	CreatedAt            *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt            *time.Time      `json:"updatedAt,omitempty"`
}

// RETLSources represents a response of RETL sources
type RETLSources struct {
	Sources []RETLSource `json:"sources"`
}
