package providers

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/workspace"
)

type DefaultResourceProvider struct {
}

func (p *DefaultResourceProvider) List(ctx context.Context, resourceType string) ([]*workspace.Resource, error) {
	return nil, NewErrUnsupporterResourceAction(resourceType, "list")
}

func (p *DefaultResourceProvider) Template(ctx context.Context, resource *workspace.Resource) ([]byte, error) {
	return nil, NewErrUnsupporterResourceAction(resource.Type, "template")
}

func (p *DefaultResourceProvider) ImportState(ctx context.Context, resource *workspace.Resource) (*state.ResourceState, error) {
	return nil, NewErrUnsupporterResourceAction(resource.Type, "import")
}
