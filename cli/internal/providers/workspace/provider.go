package workspace

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// Accounts are read-only in the IaC model: the verb layer exposes get/describe,
// but they never participate in the apply/destroy cycle. MapRemoteToState returns
// an empty state on purpose so accounts can never end up in the source graph and
// thus can never be diffed for deletion. The lifecycle methods inherited from
// EmptyProvider all return "not implemented".
var _ provider.Provider = (*Provider)(nil)

type Provider struct {
	provider.EmptyProvider
	client *client.Client
}

func New(client *client.Client) *Provider {
	return &Provider{
		client: client,
	}
}

func (p *Provider) SupportedKinds() []string {
	return []string{}
}

func (p *Provider) SupportedTypes() []string {
	return []string{AccountResourceType}
}

// List backs the (deprecated) `workspace accounts list` command.
func (p *Provider) List(ctx context.Context, resourceType string, filters lister.Filters) ([]resources.ResourceData, error) {
	if resourceType != AccountResourceType {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	return p.listAccounts(ctx, filters)
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

// LoadResourcesFromRemote backs the `get`/`describe` verbs. Accounts have no
// external id, so they always surface as unmanaged.
func (p *Provider) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	accounts, err := p.client.Accounts.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}

	out := make(map[string]*resources.RemoteResource, len(accounts))
	for _, account := range accounts {
		acc := &Account{Account: &account}
		out[account.ID] = &resources.RemoteResource{
			ID:   account.ID,
			Data: map[string]any(acc.ToResourceData()),
		}
	}

	collection := resources.NewRemoteResources()
	collection.Set(AccountResourceType, out)
	return collection, nil
}

// FormatForExport renders accounts as a plain field dump for `get -o yaml/json`
// and `describe`. Accounts are not re-appliable, so this is a read view, not a spec.
func (p *Provider) FormatForExport(collection *resources.RemoteResources, _ namer.Namer, _ resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
	accounts := collection.GetAll(AccountResourceType)
	entities := make([]writer.FormattableEntity, 0, len(accounts))
	for id, r := range accounts {
		entities = append(entities, writer.FormattableEntity{
			Content:      r.Data,
			RelativePath: fmt.Sprintf("%s.yaml", id),
		})
	}
	return entities, nil
}

// LoadImportable: accounts are not importable into IaC.
func (p *Provider) LoadImportable(_ context.Context, _ namer.Namer) (*resources.RemoteResources, error) {
	return resources.NewRemoteResources(), nil
}

// MapRemoteToState returns an empty state so accounts never enter the apply/destroy
// source graph (see the package-level note).
func (p *Provider) MapRemoteToState(_ *resources.RemoteResources) (*state.State, error) {
	return state.EmptyState(), nil
}

// ResourceGraph: accounts are never loaded from local specs.
func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	return resources.NewGraph(), nil
}

func (p *Provider) LoadSpec(_ string, _ *specs.Spec) error {
	return fmt.Errorf("workspace provider does not load specs")
}

func (p *Provider) ParseSpec(_ string, _ *specs.Spec) (*specs.ParsedSpec, error) {
	return nil, fmt.Errorf("workspace provider does not parse specs")
}
