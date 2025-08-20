package resources

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources/internal"
)

type Resource struct {
	r *internal.Resource
}

type ResourceData map[string]interface{}

func URN(ID string, resourceType string) string {
	return fmt.Sprintf("%s:%s", resourceType, ID)
}

type ImportOpts func(*internal.ResourceImportMetadata) *internal.ResourceImportMetadata

func WithResourceImportInfo(remoteId, workspaceId string) ImportOpts {
	return func(i *internal.ResourceImportMetadata) *internal.ResourceImportMetadata {
		if i == nil {
			i = &internal.ResourceImportMetadata{}
		}
		i.RemoteId = remoteId
		i.WorkspaceId = workspaceId
		return i
	}
}

func NewResource(id string, resourceType string, data ResourceData, dependencies []string, opts ...ImportOpts) *Resource {
	var importMetadata *internal.ResourceImportMetadata = nil
	for _, opt := range opts {
		importMetadata = opt(importMetadata)
	}

	return &Resource{
		r: &internal.Resource{
			URN:            URN(id, resourceType),
			ID:             id,
			Type:           resourceType,
			Data:           data,
			Dependencies:   dependencies,
			ImportMetadata: importMetadata,
		},
	}
}

func (r *Resource) ID() string {
	return r.r.ID
}

func (r *Resource) Type() string {
	return r.r.Type
}

func (r *Resource) Data() ResourceData {
	return r.r.Data
}

func (r *Resource) URN() string {
	return r.r.URN
}

func (r *Resource) Dependencies() []string {
	return r.r.Dependencies
}

func (r *Resource) ImportMetadata() *internal.ResourceImportMetadata {
	return r.r.ImportMetadata
}
