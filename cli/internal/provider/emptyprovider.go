package provider

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type EmptyProvider struct{}

func (p *EmptyProvider) Create(_ context.Context, _ string, _ string, _ resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) CreateRaw(_ context.Context, _ *resources.Resource) (any, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) Update(_ context.Context, _ string, _ string, _ resources.ResourceData, _ resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) UpdateRaw(_ context.Context, _ *resources.Resource, _ any, _ any) (any, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) Import(_ context.Context, _ string, _ string, _ resources.ResourceData, _ string, _ string) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) ImportRaw(_ context.Context, _ *resources.Resource, _ string) (any, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) Delete(_ context.Context, _ string, _ string, _ resources.ResourceData) error {
	return errNotImplemented
}

func (p *EmptyProvider) DeleteRaw(_ context.Context, _ string, _ string, _ any, _ any) error {
	return errNotImplemented
}
