package resources

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources/internal"
)

type Resource struct {
	r *internal.Resource
}

type ResourceData map[string]interface{}

func URN(ID string, resourceType string) string {
	return fmt.Sprintf("%s:%s", resourceType, ID)
}

type ResourceOpts func(*internal.Resource)

func WithResourceFileMetadata(metadataRef string) ResourceOpts {
	return func(r *internal.Resource) {
		r.FileMetadata = &internal.ResourceFileMetadata{
			MetadataRef: metadataRef,
		}
	}
}

func WithResourceImportMetadata(remoteId, workspaceId string) ResourceOpts {
	return func(r *internal.Resource) {
		r.ImportMetadata = &internal.ResourceImportMetadata{
			RemoteId:    remoteId,
			WorkspaceId: workspaceId,
		}
	}
}

func WithRawData(rawData any) ResourceOpts {
	return func(r *internal.Resource) {
		r.RawData = rawData
	}
}

func NewResource(id string, resourceType string, data ResourceData, dependencies []string, opts ...ResourceOpts) *Resource {
	r := &internal.Resource{
		URN:          URN(id, resourceType),
		ID:           id,
		Type:         resourceType,
		Data:         data,
		Dependencies: dependencies,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(r)
	}
	return &Resource{r: r}
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

func (r *Resource) RawData() any {
	return r.r.RawData
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

func (r *Resource) FileMetadata() *internal.ResourceFileMetadata {
	return r.r.FileMetadata
}
