package model

import (
	"github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
)

// DataGraphSpec represents the configuration for a data graph resource from YAML
// Maps to the "data-graph" kind YAML structure
type DataGraphSpec struct {
	ID        string      `json:"id" mapstructure:"id" validate:"required"`
	AccountID string      `json:"account_id" mapstructure:"account_id" validate:"required"`
	Models    []ModelSpec `json:"models,omitempty" mapstructure:"models" validate:"omitempty,dive"`
}

// ModelSpec represents configuration for both entity and event models from YAML
// This is part of the DataGraphSpec, not a standalone spec kind
type ModelSpec struct {
	ID            string             `json:"id" yaml:"id" mapstructure:"id" validate:"required"`
	DisplayName   string             `json:"display_name" yaml:"display_name" mapstructure:"display_name" validate:"required"`
	Type          string             `json:"type" yaml:"type" mapstructure:"type" validate:"required,oneof=entity event"`
	Table         string             `json:"table" yaml:"table" mapstructure:"table" validate:"required"`
	Description   string             `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description"`
	Relationships []RelationshipSpec `json:"relationships,omitempty" yaml:"relationships,omitempty" mapstructure:"relationships" validate:"omitempty,dive"`

	// Entity model fields (only used when Type == "entity")
	// Conditional required: handled in rule function
	PrimaryID string `json:"primary_id,omitempty" yaml:"primary_id,omitempty" mapstructure:"primary_id"`
	Root      bool   `json:"root,omitempty" yaml:"root,omitempty" mapstructure:"root"`

	// Event model fields (only used when Type == "event")
	// Conditional required: handled in rule function
	Timestamp string `json:"timestamp,omitempty" yaml:"timestamp,omitempty" mapstructure:"timestamp"`
}

// RelationshipSpec represents configuration for relationships from YAML
// This is part of the ModelSpec, not a standalone spec kind
// Type is inferred from the source model's type
type RelationshipSpec struct {
	ID            string `json:"id" yaml:"id" mapstructure:"id" validate:"required"`
	DisplayName   string `json:"display_name" yaml:"display_name" mapstructure:"display_name" validate:"required"`
	Cardinality   string `json:"cardinality" yaml:"cardinality" mapstructure:"cardinality" validate:"required,oneof=one-to-one one-to-many many-to-one"`
	Target        string `json:"target" yaml:"target" mapstructure:"target" validate:"required"`
	SourceJoinKey string `json:"source_join_key" yaml:"source_join_key" mapstructure:"source_join_key" validate:"required"`
	TargetJoinKey string `json:"target_join_key" yaml:"target_join_key" mapstructure:"target_join_key" validate:"required"`
}

// DataGraphResource represents the input data for a data graph
type DataGraphResource struct {
	ID        string `mapstructure:"id"`
	AccountID string `mapstructure:"account_id"`
}

// DataGraphState represents the output state of a data graph from the remote system
// Only contains computed fields (remote ID)
type DataGraphState struct {
	ID string // Remote ID
}

// RemoteDataGraph wraps datagraph.DataGraph to implement RemoteResource interface
type RemoteDataGraph struct {
	*datagraph.DataGraph
	AccountName string // Resolved account name for human-readable naming
}

// Metadata implements the RemoteResource interface
func (r RemoteDataGraph) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: r.WorkspaceID,
		Name:        r.AccountName,
	}
}
