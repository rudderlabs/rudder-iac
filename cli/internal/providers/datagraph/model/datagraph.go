package model

import (
	"github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
)

// DataGraphSpec represents the configuration for a data graph resource from YAML
// Maps to the "data-graph" kind YAML structure
type DataGraphSpec struct {
	ID        string      `json:"id" mapstructure:"id"`
	AccountID string      `json:"account_id" mapstructure:"account_id"`
	Models    []ModelSpec `json:"models,omitempty" mapstructure:"models"` // Inline models
}

// ModelSpec represents configuration for both entity and event models from YAML
// This is part of the DataGraphSpec, not a standalone spec kind
type ModelSpec struct {
	ID            string             `json:"id" mapstructure:"id"`
	DisplayName   string             `json:"display_name" mapstructure:"display_name"`
	Type          string             `json:"type" mapstructure:"type"` // "entity" or "event"
	Table         string             `json:"table" mapstructure:"table"`
	Description   string             `json:"description,omitempty" mapstructure:"description"`
	Relationships []RelationshipSpec `json:"relationships,omitempty" mapstructure:"relationships"` // Inline relationships

	// Entity model fields (only used when Type == "entity")
	PrimaryID string `json:"primary_id,omitempty" mapstructure:"primary_id"`
	Root      bool   `json:"root,omitempty" mapstructure:"root"`

	// Event model fields (only used when Type == "event")
	Timestamp string `json:"timestamp,omitempty" mapstructure:"timestamp"`
}

// RelationshipSpec represents configuration for relationships from YAML
// This is part of the ModelSpec, not a standalone spec kind
// Type is inferred from the source model's type
type RelationshipSpec struct {
	ID            string `json:"id" mapstructure:"id"`
	DisplayName   string `json:"display_name" mapstructure:"display_name"`
	Cardinality   string `json:"cardinality" mapstructure:"cardinality"` // Required: "one-to-one", "one-to-many", "many-to-one"
	Target        string `json:"target" mapstructure:"target"`           // Reference: '#data-graph-model:user'
	SourceJoinKey string `json:"source_join_key" mapstructure:"source_join_key"`
	TargetJoinKey string `json:"target_join_key" mapstructure:"target_join_key"`
}

// DataGraphResource represents the input data for a data graph
type DataGraphResource struct {
	ID        string
	AccountID string
}

// DataGraphState represents the output state of a data graph from the remote system
// Only contains computed fields (remote ID)
type DataGraphState struct {
	ID string // Remote ID
}

// RemoteDataGraph wraps datagraph.DataGraph to implement RemoteResource interface
type RemoteDataGraph struct {
	*datagraph.DataGraph
}

// Metadata implements the RemoteResource interface
func (r RemoteDataGraph) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: r.WorkspaceID,
	}
}
