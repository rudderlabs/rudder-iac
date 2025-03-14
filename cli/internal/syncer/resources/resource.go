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

func NewResource(id string, resourceType string, data ResourceData, dependencies []string) *Resource {
	return &Resource{
		r: &internal.Resource{
			URN:          URN(id, resourceType),
			ID:           id,
			Type:         resourceType,
			Data:         data,
			Dependencies: dependencies,
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
