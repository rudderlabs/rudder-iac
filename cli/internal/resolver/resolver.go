package resolver

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type ReferenceResolver interface {
	ResolveToReference(entityType string, remoteID string) (string, error)
}

type ImportRefResolver struct {
	Remote     *resources.ResourceCollection
	Graph      *resources.Graph
	Importable *resources.ResourceCollection
}

func (i *ImportRefResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	// If we find a resource in importable,
	// we should always find the reference in it
	resource, ok := i.Importable.GetByID(entityType, remoteID)
	if ok {
		return resource.Reference, nil
	}

	resource, ok = i.Remote.GetByID(entityType, remoteID)
	if !ok {
		return "", fmt.Errorf("resource not present in resources collection")
	}

	graphResource, ok := i.Graph.GetResource(
		resources.URN(
			resource.ExternalID,
			entityType,
		),
	)
	if !ok {
		return "", fmt.Errorf("resource not present in resources graph")
	}

	if graphResource.FileMetadata() == nil || graphResource.FileMetadata().MetadataRef == "" {
		return "", fmt.Errorf("file metadata on the graph resource is not present")
	}

	return graphResource.FileMetadata().MetadataRef, nil
}
