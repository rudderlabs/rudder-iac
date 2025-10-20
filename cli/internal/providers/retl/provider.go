package retl

import (
	"context"
	"fmt"
	"strconv"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// Provider implements the provider interface for RETL resources
type Provider struct {
	client     retlClient.RETLStore
	handlers   map[string]resourceHandler
	kindToType map[string]string
}

// New creates a new RETL provider instance
func New(client retlClient.RETLStore) *Provider {
	p := &Provider{
		client:   client,
		handlers: make(map[string]resourceHandler),
		kindToType: map[string]string{
			"retl-source-sql-model": sqlmodel.ResourceType,
		},
	}

	// Register handlers
	p.handlers[sqlmodel.ResourceType] = sqlmodel.NewHandler(client)

	return p
}

func (p *Provider) GetName() string {
	return "retl"
}

func (p *Provider) GetSupportedKinds() []string {
	kinds := make([]string, 0, len(p.kindToType))
	for kind := range p.kindToType {
		kinds = append(kinds, kind)
	}
	return kinds
}

// GetSupportedTypes returns the list of supported resource types
func (p *Provider) GetSupportedTypes() []string {
	types := make([]string, 0, len(p.handlers))
	for resourceType := range p.handlers {
		types = append(types, resourceType)
	}
	return types
}

// LoadSpec loads a spec for the given kind
func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	resourceType, ok := p.kindToType[s.Kind]
	if !ok {
		return fmt.Errorf("unsupported kind: %s", s.Kind)
	}

	handler, ok := p.handlers[resourceType]
	if !ok {
		return fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.LoadSpec(path, s)
}

// Validate validates all loaded specs
func (p *Provider) Validate() error {
	for resourceType, handler := range p.handlers {
		if err := handler.Validate(); err != nil {
			return fmt.Errorf("validating %s: %w", resourceType, err)
		}
	}
	return nil
}

// GetResourceGraph returns a graph of all resources
func (p *Provider) GetResourceGraph() (*resources.Graph, error) {
	graph := resources.NewGraph()

	for resourceType, handler := range p.handlers {
		resources, err := handler.GetResources()
		if err != nil {
			return nil, fmt.Errorf("getting resources for %s: %w", resourceType, err)
		}

		for _, resource := range resources {
			graph.AddResource(resource)
		}
	}

	return graph, nil
}

// LoadState loads the current state of all resources
func (p *Provider) LoadState(ctx context.Context) (*state.State, error) {
	remoteState, err := p.client.ReadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading remote state: %w", err)
	}

	s := state.EmptyState()

	for _, rs := range remoteState.Resources {
		s.AddResource(&state.ResourceState{
			Type:         rs.Type,
			ID:           rs.ID,
			Input:        rs.Input,
			Output:       rs.Output,
			Dependencies: rs.Dependencies,
		})
	}

	return s, nil
}

// PutResourceState saves the state of a resource
func (p *Provider) PutResourceState(ctx context.Context, URN string, s *state.ResourceState) error {
	remoteID, ok := s.Output["id"].(string)
	if !ok {
		return fmt.Errorf("missing id in resource state")
	}

	req := retlClient.PutStateRequest{
		URN: URN,
		State: retlClient.ResourceState{
			ID:           s.ID,
			Type:         s.Type,
			Input:        s.Input,
			Output:       s.Output,
			Dependencies: s.Dependencies,
		},
	}

	return p.client.PutResourceState(ctx, remoteID, req)
}

// DeleteResourceState is deprecated as removing resource from the state
// will be handled from the delete retl source endpoint
func (p *Provider) DeleteResourceState(ctx context.Context, state *state.ResourceState) error {
	return nil
}

// Create creates a new resource
func (p *Provider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.Create(ctx, ID, data)
}

// Update updates an existing resource
func (p *Provider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.Update(ctx, ID, data, state)
}

// Delete deletes an existing resource
func (p *Provider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.Delete(ctx, ID, state)
}

// List lists RETL resources of the specified type with optional filters
func (p *Provider) List(ctx context.Context, resourceType string, filters lister.Filters) ([]resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	var hasExtranlId *bool
	if hasEsternalIdStr, ok := filters["hasExternalId"]; ok {
		hasExternalId, err := strconv.ParseBool(hasEsternalIdStr)
		if err != nil {
			return nil, fmt.Errorf("invalid hasExternalId filter: %w", err)
		}
		hasExtranlId = &hasExternalId
	}
	return handler.List(ctx, hasExtranlId)
}

func (p *Provider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.Import(ctx, ID, data, remoteId)
}

// Import imports a single remote RETL resource with local ID mapping
func (p *Provider) FetchImportData(ctx context.Context, resourceType string, args importremote.ImportArgs) ([]importremote.ImportData, error) {
	// Only support SQL models for import in this phase
	if resourceType != sqlmodel.ResourceType {
		return nil, fmt.Errorf("import is only supported for SQL models, got: %s", resourceType)
	}

	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.FetchImportData(ctx, args)
}

// LoadResourcesFromRemote loads all RETL resources from remote (no-op implementation)
func (p *Provider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()
	for resourceType, handler := range p.handlers {
		c, err := handler.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", resourceType, err)
		}
		collection, err = collection.Merge(c)
		if err != nil {
			return nil, fmt.Errorf("merging collection for %s: %w", resourceType, err)
		}
	}
	return collection, nil
}

// LoadStateFromResources reconstructs RETL state from loaded resources (no-op implementation)
func (p *Provider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	s := state.EmptyState()
	for sqlmodelResourceType, handler := range p.handlers {
		providerState, err := handler.LoadStateFromResources(ctx, collection)
		if err != nil {
			return nil, fmt.Errorf("loading state from provider handler %s: %w", sqlmodelResourceType, err)
		}
		s, err = s.Merge(providerState)
		if err != nil {
			return nil, fmt.Errorf("merging provider states: %w", err)
		}
	}
	return s, nil
}

// Preview returns the preview results for a resource
func (p *Provider) Preview(ctx context.Context, ID string, resourceType string, data resources.ResourceData, limit int) ([]map[string]any, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.Preview(ctx, ID, data, limit)
}

func (p *Provider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	return resources.NewResourceCollection(), nil
}

func (p *Provider) FormatForExport(ctx context.Context, collection *resources.ResourceCollection, idNamer namer.Namer, inputResolver resolver.ReferenceResolver) ([]importremote.FormattableEntity, error) {
	return nil, nil
}
