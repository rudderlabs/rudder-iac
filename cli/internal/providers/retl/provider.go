package retl

import (
	"context"
	"fmt"
	"strconv"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"

	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// Provider implements the provider interface for RETL resources
type Provider struct {
	client     retlClient.RETLStore
	handlers   map[string]resourceHandler
	kindToType map[string]string
}

const importDir = "retl"

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
	p.handlers[sqlmodel.ResourceType] = sqlmodel.NewHandler(client, importDir)

	return p
}

func (p *Provider) SupportedKinds() []string {
	kinds := make([]string, 0, len(p.kindToType))
	for kind := range p.kindToType {
		kinds = append(kinds, kind)
	}
	return kinds
}

// SupportedTypes returns the list of supported resource types
func (p *Provider) SupportedTypes() []string {
	types := make([]string, 0, len(p.handlers))
	for resourceType := range p.handlers {
		types = append(types, resourceType)
	}
	return types
}

func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	resourceType, ok := p.kindToType[s.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported kind: %s", s.Kind)
	}
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.ParseSpec(path, s)
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
func (p *Provider) Validate(_ *resources.Graph) error {
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

	var hasExternalId *bool
	if hasEsternalIdStr, ok := filters["hasExternalId"]; ok {
		id, err := strconv.ParseBool(hasEsternalIdStr)
		if err != nil {
			return nil, fmt.Errorf("invalid hasExternalId filter: %w", err)
		}
		hasExternalId = &id
	}
	return handler.List(ctx, hasExternalId)
}

func (p *Provider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.Import(ctx, ID, data, remoteId)
}

// FetchImportData fetches a single remote RETL resource formatted for import
func (p *Provider) FetchImportData(ctx context.Context, resourceType string, importIDs specs.ImportIds) (writer.FormattableEntity, error) {
	// Only support SQL models for import in this phase
	if resourceType != sqlmodel.ResourceType {
		return writer.FormattableEntity{}, fmt.Errorf("import is only supported for SQL models, got: %s", resourceType)
	}

	handler, ok := p.handlers[resourceType]
	if !ok {
		return writer.FormattableEntity{}, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.FetchImportData(ctx, importIDs)
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
	collection := resources.NewResourceCollection()
	for _, handler := range p.handlers {
		resources, err := handler.LoadImportable(ctx, idNamer)
		if err != nil {
			return nil, fmt.Errorf("loading importable resources from handler %w", err)
		}
		collection, err = collection.Merge(resources)
		if err != nil {
			return nil, fmt.Errorf("merging importable resource collection for handler %w", err)
		}
	}
	return collection, nil
}

func (p *Provider) FormatForExport(ctx context.Context, collection *resources.ResourceCollection, idNamer namer.Namer, inputResolver resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
	allEntities := make([]writer.FormattableEntity, 0)
	for _, handler := range p.handlers {
		entities, err := handler.FormatForExport(ctx, collection, idNamer, inputResolver)
		if err != nil {
			return nil, fmt.Errorf("formatting for export for handler %w", err)
		}
		allEntities = append(allEntities, entities...)
	}
	return allEntities, nil
}
