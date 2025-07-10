package workspace

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

const ProviderName = "workspace"

type Provider struct {
	client *client.Client
}

func New(client *client.Client) *Provider {
	return &Provider{
		client: client,
	}
}

func (p *Provider) GetName() string {
	return ProviderName
}

func (p *Provider) List(ctx context.Context, resourceType string, filters map[string]string) ([]resources.ResourceData, error) {
	switch resourceType {
	case AccountResourceType:
		return p.listAccounts(ctx, filters)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func (p *Provider) listAccounts(ctx context.Context, filters map[string]string) ([]resources.ResourceData, error) {
	var filteredAccounts []resources.ResourceData
	accounts, err := p.client.Accounts.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		if category, ok := filters["category"]; ok && account.Definition.Category != category {
			continue
		}
		if accountType, ok := filters["type"]; ok && account.Definition.Type != accountType {
			continue
		}
		acc := &Account{Account: &account}
		filteredAccounts = append(filteredAccounts, acc.ToResourceData())
	}

	return filteredAccounts, nil
}

func (p *Provider) GetSupportedKinds() []string {
	return []string{}
}

func (p *Provider) GetSupportedTypes() []string {
	return []string{AccountResourceType}
}

func (p *Provider) Validate() error {
	return nil
}

func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	return nil
}

func (p *Provider) GetResourceGraph() (*resources.Graph, error) {
	return resources.NewGraph(), nil
}

func (p *Provider) LoadState(ctx context.Context) (*state.State, error) {
	return state.EmptyState(), nil
}

func (p *Provider) PutResourceState(ctx context.Context, URN string, state *state.ResourceState) error {
	return nil
}

func (p *Provider) DeleteResourceState(ctx context.Context, state *state.ResourceState) error {
	return nil
}

func (p *Provider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	return nil, project.ErrNotImplemented
}

func (p *Provider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	return nil, project.ErrNotImplemented
}

func (p *Provider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	return project.ErrNotImplemented
}
