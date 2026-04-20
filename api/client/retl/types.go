package retl

import (
	"encoding/json"
	"fmt"
	"time"
)

type SourceType string

type AsyncStatus string

const (
	ModelSourceType SourceType = "model"
	TableSourceType SourceType = "table"

	Pending   AsyncStatus = "pending"
	Failed    AsyncStatus = "failed"
	Completed AsyncStatus = "completed"
)

// RETLSource represents a RETL source in the API
type RETLSource struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Config               json.RawMessage `json:"config"`
	IsEnabled            bool            `json:"enabled"`
	SourceType           SourceType      `json:"sourceType"`
	SourceDefinitionName string          `json:"sourceDefinitionName"`
	AccountID            string          `json:"accountId"`
	CreatedAt            *time.Time      `json:"createdAt"`
	UpdatedAt            *time.Time      `json:"updatedAt"`
	WorkspaceID          string          `json:"workspaceId"`
	ExternalID           string          `json:"externalId"`
}

type RETLSourceCreateRequest struct {
	Name                 string          `json:"name"`
	Config               json.RawMessage `json:"config"`
	SourceType           SourceType      `json:"sourceType"`
	SourceDefinitionName string          `json:"sourceDefinitionName"`
	AccountID            string          `json:"accountId"`
	Enabled              bool            `json:"enabled"`
	ExternalID           string          `json:"externalId"`
}

type RETLSourceUpdateRequest struct {
	Name      string          `json:"name"`
	Config    json.RawMessage `json:"config"`
	IsEnabled bool            `json:"enabled"`
	AccountID string          `json:"accountId"`
}

// RETLSQLModelConfig is the config shape for SQL MODEL sources.
type RETLSQLModelConfig struct {
	PrimaryKey  string `json:"primaryKey"`
	Sql         string `json:"sql"`
	Description string `json:"description,omitempty"`
}

// RETLTableConfig is the config shape for warehouse TABLE sources
// (sourceDefinitionName = snowflake/bigquery/postgres/etc.).
type RETLTableConfig struct {
	PrimaryKey string `json:"primaryKey"`
	Schema     string `json:"schema"`
	Table      string `json:"table"`
}

// RETLS3TableConfig is the config shape for S3 TABLE sources
// (sourceDefinitionName = s3). primaryKey is optional on S3.
type RETLS3TableConfig struct {
	BucketName   string `json:"bucketName"`
	ObjectPrefix string `json:"objectPrefix,omitempty"`
}

// ConfigType enumerates the RETL source config shapes supported by
// MarshalConfig / DecodeConfig. Add new config types here.
type ConfigType interface {
	RETLSQLModelConfig | RETLTableConfig | RETLS3TableConfig
}

// MarshalConfig encodes a typed RETL source config into the json.RawMessage
// shape expected by RETLSourceCreateRequest.Config / RETLSourceUpdateRequest.Config.
func MarshalConfig[T ConfigType](cfg T) (json.RawMessage, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshalling RETL config: %w", err)
	}
	return raw, nil
}

// DecodeConfig decodes a RETLSource.Config into the caller-supplied typed config.
// Empty input returns a zero-value T and no error.
func DecodeConfig[T ConfigType](raw json.RawMessage) (T, error) {
	var cfg T
	if len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("unmarshalling RETL config: %w", err)
	}
	return cfg, nil
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
