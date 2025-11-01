package provider

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type EmptyProvider struct{}

func (p *EmptyProvider) Create(_ context.Context, _ string, _ resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) CreateRaw(_ context.Context, _ *resources.Resource) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) Update(_ context.Context, _ string, _ string, _ resources.ResourceData, _ resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) UpdateRaw(_ context.Context, _ *resources.Resource, _ resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) Import(_ context.Context, _ string, _ resources.ResourceData, _ string) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *EmptyProvider) ImportRaw(_ context.Context, _ *resources.Resource, _ string) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}
