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

	s3SourceDefinition = "s3"

	Pending   AsyncStatus = "pending"
	Failed    AsyncStatus = "failed"
	Completed AsyncStatus = "completed"
)

// ConfigType is the sealed union of RETL source config shapes. Implementations
// are confined to this package via the unexported marker method, so the
// dispatch in RETLSource.UnmarshalJSON stays exhaustive.
type ConfigType interface {
	isRETLConfig()
}

// RETLSQLModelConfig is the config shape for SQL MODEL sources.
type RETLSQLModelConfig struct {
	PrimaryKey  string `json:"primaryKey"`
	Sql         string `json:"sql"`
	Description string `json:"description,omitempty"`
}

func (RETLSQLModelConfig) isRETLConfig() {}

// RETLTableConfig is the config shape for warehouse TABLE sources
// (sourceDefinitionName = snowflake/bigquery/postgres/etc.).
type RETLTableConfig struct {
	PrimaryKey string `json:"primaryKey"`
	Schema     string `json:"schema"`
	Table      string `json:"table"`
}

func (RETLTableConfig) isRETLConfig() {}

// RETLS3TableConfig is the config shape for S3 TABLE sources
// (sourceDefinitionName = s3). primaryKey is optional on S3.
type RETLS3TableConfig struct {
	BucketName   string `json:"bucketName"`
	ObjectPrefix string `json:"objectPrefix,omitempty"`
}

func (RETLS3TableConfig) isRETLConfig() {}

// RETLSource represents a RETL source in the API
type RETLSource struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Config               ConfigType `json:"config"`
	IsEnabled            bool       `json:"enabled"`
	SourceType           SourceType `json:"sourceType"`
	SourceDefinitionName string     `json:"sourceDefinitionName"`
	AccountID            string     `json:"accountId"`
	CreatedAt            *time.Time `json:"createdAt"`
	UpdatedAt            *time.Time `json:"updatedAt"`
	WorkspaceID          string     `json:"workspaceId"`
	ExternalID           string     `json:"externalId"`
}

// retlSourceWire mirrors RETLSource but keeps Config as raw JSON so we can
// pick the concrete config type from sourceType + sourceDefinitionName.
type retlSourceWire struct {
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

func (s *RETLSource) UnmarshalJSON(data []byte) error {
	var wire retlSourceWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*s = RETLSource{
		ID:                   wire.ID,
		Name:                 wire.Name,
		IsEnabled:            wire.IsEnabled,
		SourceType:           wire.SourceType,
		SourceDefinitionName: wire.SourceDefinitionName,
		AccountID:            wire.AccountID,
		CreatedAt:            wire.CreatedAt,
		UpdatedAt:            wire.UpdatedAt,
		WorkspaceID:          wire.WorkspaceID,
		ExternalID:           wire.ExternalID,
	}
	if len(wire.Config) == 0 || string(wire.Config) == "null" {
		return nil
	}
	cfg, err := decodeConfigFor(wire.SourceType, wire.SourceDefinitionName, wire.Config)
	if err != nil {
		return err
	}
	s.Config = cfg
	return nil
}

func decodeConfigFor(sourceType SourceType, sourceDefinitionName string, raw json.RawMessage) (ConfigType, error) {
	switch sourceType {
	case ModelSourceType:
		var cfg RETLSQLModelConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshalling RETL model config: %w", err)
		}
		return cfg, nil
	case TableSourceType:
		if sourceDefinitionName == s3SourceDefinition {
			var cfg RETLS3TableConfig
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return nil, fmt.Errorf("unmarshalling RETL s3 table config: %w", err)
			}
			return cfg, nil
		}
		var cfg RETLTableConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshalling RETL table config: %w", err)
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("unsupported RETL source type %q", sourceType)
	}
}

type RETLSourceCreateRequest struct {
	Name                 string     `json:"name"`
	Config               ConfigType `json:"config"`
	SourceType           SourceType `json:"sourceType"`
	SourceDefinitionName string     `json:"sourceDefinitionName"`
	AccountID            string     `json:"accountId"`
	Enabled              bool       `json:"enabled"`
	ExternalID           string     `json:"externalId"`
}

type RETLSourceUpdateRequest struct {
	Name      string     `json:"name"`
	Config    ConfigType `json:"config"`
	IsEnabled bool       `json:"enabled"`
	AccountID string     `json:"accountId"`
}

// DecodeConfig narrows a RETLSource.Config (held as the ConfigType interface)
// to a specific config type. Returns the zero value and no error when the
// interface is nil. Returns an error when the held concrete type doesn't match T.
func DecodeConfig[T ConfigType](raw ConfigType) (T, error) {
	var zero T
	if raw == nil {
		return zero, nil
	}
	typed, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("RETL config is %T, not %T", raw, zero)
	}
	return typed, nil
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
