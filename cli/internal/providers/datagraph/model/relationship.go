package model

import (
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// RelationshipResource represents a relationship resource
type RelationshipResource struct {
	ID             string                 `mapstructure:"id"`
	DisplayName    string                 `mapstructure:"display_name"`
	DataGraphRef   *resources.PropertyRef `mapstructure:"data_graph"`    // Parent data graph
	SourceModelRef *resources.PropertyRef `mapstructure:"source"` // Source model
	TargetModelRef *resources.PropertyRef `mapstructure:"target"` // Target model
	SourceJoinKey  string                 `mapstructure:"source_join_key"`
	TargetJoinKey  string                 `mapstructure:"target_join_key"`
	Cardinality    string                 `mapstructure:"cardinality"` // "one-to-one", "one-to-many", or "many-to-one"
}

// RelationshipState represents the remote state of a relationship
type RelationshipState struct {
	ID string // Remote relationship ID
}

// RemoteRelationship wraps dgClient.Relationship to implement RemoteResource interface
type RemoteRelationship struct {
	*dgClient.Relationship
}

// Metadata implements the RemoteResource interface
func (r RemoteRelationship) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: "", // Relationships don't have workspace ID directly
		Name:        r.Name,
	}
}
