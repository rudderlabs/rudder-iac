package model

import (
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// RelationshipResource represents a relationship resource
type RelationshipResource struct {
	ID             string
	DisplayName    string
	DataGraphRef   *resources.PropertyRef // Parent data graph
	SourceModelRef *resources.PropertyRef // Source model
	TargetModelRef *resources.PropertyRef // Target model
	SourceJoinKey  string
	TargetJoinKey  string
	Cardinality    string // "one-to-one", "one-to-many", or "many-to-one"
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
