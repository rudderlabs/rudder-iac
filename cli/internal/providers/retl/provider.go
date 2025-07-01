package retl

import (
	"context"
	"encoding/json"
	"fmt"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type Provider struct {
	sqlModelSpecs []*SQLModelSpec
}

func New(client retlClient.RETLStore) *Provider {
	return &Provider{
		sqlModelSpecs: []*SQLModelSpec{},
	}
}

func (p *Provider) GetSupportedKinds() []string {
	return []string{"retl-sql-model"}
}

func (p *Provider) GetSupportedTypes() []string {
	return []string{
		"sql-model",
	}
}

func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	switch s.Kind {
	case "retl-sql-model":
		spec := &SQLModelSpec{}

		jsonByt, err := json.Marshal(s.Spec)
		if err != nil {
			return fmt.Errorf("marshalling the spec: %w", err)
		}

		if err := json.Unmarshal(jsonByt, &spec); err != nil {
			return fmt.Errorf("extracting the property spec: %w", err)
		}

		p.sqlModelSpecs = append(p.sqlModelSpecs, spec)
	}

	return nil
}

func (p *Provider) Validate() error {
	for _, spec := range p.sqlModelSpecs {
		if err := ValidateSQLModelSpec(spec); err != nil {
			return fmt.Errorf("validating sql model spec: %w", err)
		}
	}
	return nil
}

func (p *Provider) GetResourceGraph() (*resources.Graph, error) {
	return nil, nil
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
	return nil, nil
}

func (p *Provider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	return nil, nil
}

func (p *Provider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	return nil
}
