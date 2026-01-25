package model

import (
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// RelationshipResource represents a relationship resource (both entity and event)
type RelationshipResource struct {
	ID            string
	DisplayName   string
	Type          string                 // "entity" or "event"
	DataGraphRef  *resources.PropertyRef // Parent data graph
	FromModelRef  *resources.PropertyRef // Source model
	ToModelRef    *resources.PropertyRef // Target model
	SourceJoinKey string
	TargetJoinKey string
	Cardinality   string // only for entity relationships
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
