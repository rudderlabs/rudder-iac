package retl

import (
	"time"
)

type SourceType string
type SourceDefinition string

const (
	ModelSourceType SourceType = "model"

	SourceDefinitionPostgres   SourceDefinition = "postgres"
	SourceDefinitionRedshift   SourceDefinition = "redshift"
	SourceDefinitionSnowflake  SourceDefinition = "snowflake"
	SourceDefinitionBigQuery   SourceDefinition = "bigquery"
	SourceDefinitionMySQL      SourceDefinition = "mysql"
	SourceDefinitionDatabricks SourceDefinition = "databricks"
	SourceDefinitionTrino      SourceDefinition = "trino"
)

// validSourceDefinitions contains all valid source definition values
var ValidSourceDefinitions = []SourceDefinition{
	SourceDefinitionPostgres,
	SourceDefinitionRedshift,
	SourceDefinitionSnowflake,
	SourceDefinitionBigQuery,
	SourceDefinitionMySQL,
	SourceDefinitionDatabricks,
	SourceDefinitionTrino,
}

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
	SourceDefinitionName SourceDefinition   `json:"sourceDefinitionName"`
	AccountID            string             `json:"accountId"`
	CreatedAt            *time.Time         `json:"createdAt"`
	UpdatedAt            *time.Time         `json:"updatedAt"`
}

type RETLSourceCreateRequest struct {
	Name                 string             `json:"name"`
	Config               RETLSQLModelConfig `json:"config"`
	SourceType           SourceType         `json:"sourceType"`
	SourceDefinitionName SourceDefinition   `json:"sourceDefinitionName"`
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
