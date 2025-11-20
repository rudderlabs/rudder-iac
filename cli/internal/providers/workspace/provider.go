package workspace

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type Provider struct {
	client *client.Client
}

func New(client *client.Client) *Provider {
	return &Provider{
		client: client,
	}
}

func (p *Provider) List(ctx context.Context, resourceType string, filters lister.Filters) ([]resources.ResourceData, error) {
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
