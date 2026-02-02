package datacatalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	propertyRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/property"
	pstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/validate"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var log = logger.New("datacatalogprovider")

const importDir = "data-catalog"

type Provider struct {
	provider.EmptyProvider
	concurrency   int
	client        catalog.DataCatalog
	dc            *localcatalog.DataCatalog
	providerStore map[string]entityProvider
}

func New(client catalog.DataCatalog) *Provider {
	return &Provider{
		concurrency: config.GetConfig().Concurrency.CatalogProvider,
		client:      client,
		dc:          localcatalog.New(),
		providerStore: map[string]entityProvider{
			types.PropertyResourceType:     NewPropertyProvider(client, importDir),
			types.EventResourceType:        NewEventProvider(client, importDir),
			types.CustomTypeResourceType:   NewCustomTypeProvider(client, importDir),
			types.TrackingPlanResourceType: NewTrackingPlanProvider(client, importDir),
			types.CategoryResourceType:     NewCategoryProvider(client, importDir),
		},
	}
}

func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	return p.dc.ParseSpec(path, s)
}

func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	return p.dc.LoadSpec(path, s)
}

func (p *Provider) LoadLegacySpec(path string, s *specs.Spec) error {
	return p.dc.LoadLegacySpec(path, s)
}

func (p *Provider) MigrateSpec(s *specs.Spec) (*specs.Spec, error) {
	return p.dc.MigrateSpec(s)
}

func (p *Provider) SupportedKinds() []string {
	return []string{
		localcatalog.KindProperties,
		localcatalog.KindEvents,
		localcatalog.KindTrackingPlans,
		localcatalog.KindCustomTypes,
		localcatalog.KindCategories,
	}
}

func (p *Provider) SupportedTypes() []string {
	return []string{
		types.PropertyResourceType,
		types.EventResourceType,
		types.TrackingPlanResourceType,
		types.CustomTypeResourceType,
		types.CategoryResourceType,
	}
}

func (p *Provider) GetLocalCatalog() *localcatalog.DataCatalog {
	return p.dc
}

// Validate validates the provider's data catalog.
// The method accepts a *resources.Graph but currently ignores it and validates directly from the catalog
// (same behavior as earlier); future implementations may validate against the graph.
func (p *Provider) Validate(_ *resources.Graph) error {
	err := validate.ValidateCatalog(p.dc)
	if err == nil {
		log.Info("successfully validated the catalog")
		return nil
	}

	return fmt.Errorf("catalog is invalid: %s", err.Error())
}

func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	if err := inflateRefs(p.dc); err != nil {
		return nil, fmt.Errorf("inflating refs: %w", err)
	}

	return createResourceGraph(p.dc)
}

func (p *Provider) List(ctx context.Context, resourceType string, filters lister.Filters) ([]resources.ResourceData, error) {
	switch resourceType {
	case types.TrackingPlanResourceType:
		return p.listTrackingPlans(ctx)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func (p *Provider) listTrackingPlans(ctx context.Context) ([]resources.ResourceData, error) {
	trackingPlans, err := p.client.GetTrackingPlansWithIdentifiers(ctx, catalog.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tracking plans: %w", err)
	}

	var result []resources.ResourceData
	for _, tp := range trackingPlans {
		// Handle nil description
		description := ""
		if tp.Description != nil {
			description = *tp.Description
		}

		// Include all relevant fields for Lister display
		// "id" and "name" are shown in columns, other fields in details panel
		resourceData := resources.ResourceData{
			"name":        tp.Name,
			"id":          tp.ID,
			"version":     tp.Version,
			"description": description,
			"createdAt":   tp.CreatedAt.String(),
			"updatedAt":   tp.UpdatedAt.String(),
		}
		result = append(result, resourceData)
	}

	return result, nil
}

func createResourceGraph(catalog *localcatalog.DataCatalog) (*resources.Graph, error) {
	graph := resources.NewGraph()

	// First, pre-calculate all URNs to use for references
	propIDToURN := make(map[string]string)
	for _, prop := range catalog.Properties {
		propIDToURN[prop.LocalID] = resources.URN(prop.LocalID, types.PropertyResourceType)
	}

	eventIDToURN := make(map[string]string)
	for _, event := range catalog.Events {
		eventIDToURN[event.LocalID] = resources.URN(event.LocalID, types.EventResourceType)
	}

	customTypeIDToURN := make(map[string]string)
	for _, customType := range catalog.CustomTypes {
		customTypeIDToURN[customType.LocalID] = resources.URN(customType.LocalID, types.CustomTypeResourceType)
	}

	categoryIDToURN := make(map[string]string)
	for _, category := range catalog.Categories {
		categoryIDToURN[category.LocalID] = resources.URN(category.LocalID, types.CategoryResourceType)
	}

	getResourceImportMetadata := func(kind, id string) resources.ResourceOpts {
		metadata, ok := catalog.ImportMetadata[resources.URN(kind, id)]
		if !ok {
			return nil
		}
		return resources.WithResourceImportMetadata(metadata.RemoteID, metadata.WorkspaceID)
	}

	// Helper function to get URN from reference
	getURNFromRef := func(ref string) string {
		return strings.TrimPrefix(ref, "#")
	}

	// Add properties to the graph
	for _, prop := range catalog.Properties {
		log.Debug("adding property to graph", "id", prop.LocalID)

		args := &pstate.PropertyArgs{}
		if err := args.FromCatalogPropertyType(prop, getURNFromRef); err != nil {
			return nil, fmt.Errorf("creating property args from catalog property: %s, err:%w", prop.LocalID, err)
		}

		resource := resources.NewResource(
			prop.LocalID,
			types.PropertyResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(localcatalog.KindProperties, prop.LocalID),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindProperties,
				prop.LocalID,
			)),
		)
		graph.AddResource(resource)

		propIDToURN[prop.LocalID] = resource.URN()
	}

	// Add events to the graph
	for _, event := range catalog.Events {
		log.Debug("adding event to graph", "event", event.LocalID)

		args := pstate.EventArgs{}
		args.FromCatalogEvent(&event, getURNFromRef)
		resource := resources.NewResource(
			event.LocalID,
			types.EventResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(localcatalog.KindEvents, event.LocalID),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindEvents,
				event.LocalID,
			)),
		)
		graph.AddResource(resource)

		eventIDToURN[event.LocalID] = resource.URN()
	}

	// Add custom types to the graph with dependencies on properties or other custom types
	for _, customType := range catalog.CustomTypes {
		log.Debug("adding custom type to graph", "id", customType.LocalID)

		// Add CustomTypeArgs
		args := pstate.CustomTypeArgs{}
		args.FromCatalogCustomType(&customType, getURNFromRef)
		resource := resources.NewResource(
			customType.LocalID,
			types.CustomTypeResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(localcatalog.KindCustomTypes, customType.LocalID),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindCustomTypes,
				customType.LocalID,
			)),
		)
		graph.AddResource(resource)
	}

	// Add categories to the graph
	for _, category := range catalog.Categories {
		log.Debug("adding category to graph", "id", category.LocalID)

		args := pstate.CategoryArgs{}
		args.FromCatalogCategory(&category)
		resource := resources.NewResource(
			category.LocalID,
			types.CategoryResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(localcatalog.KindCategories, category.LocalID),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindCategories,
				category.LocalID,
			)),
		)
		graph.AddResource(resource)
	}

	// Add tracking plans to the graph
	for _, tp := range catalog.TrackingPlans {
		log.Debug("adding tracking plan to graph", "tp", tp.LocalID)

		args := pstate.TrackingPlanArgs{}
		if err := args.FromCatalogTrackingPlan(tp, getURNFromRef); err != nil {
			return nil, fmt.Errorf("creating tracking plan args: %w", err)
		}

		resource := resources.NewResource(
			tp.LocalID,
			types.TrackingPlanResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(localcatalog.KindTrackingPlans, tp.LocalID),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindTrackingPlans,
				tp.LocalID,
			)),
		)
		graph.AddResource(resource)
		graph.AddDependencies(resource.URN(), getDependencies(tp, propIDToURN, eventIDToURN))
	}

	return graph, nil
}

// getDependencies simply fetch the dependencies on the trackingplan in form of the URN's
// of the properties and events that are used in the tracking plan
func getDependencies(tp *localcatalog.TrackingPlanV1, propIDToURN, eventIDToURN map[string]string) []string {
	dependencies := make([]string, 0)

	for _, event := range tp.EventProps {
		if urn, ok := eventIDToURN[event.LocalID]; ok {
			dependencies = append(dependencies, urn)
		}

		for _, prop := range event.Properties {
			if urn, ok := propIDToURN[prop.LocalID]; ok {
				dependencies = append(dependencies, urn)
			}
		}
	}

	return dependencies
}

func inflateRefs(catalog *localcatalog.DataCatalog) error {
	log.Debug("inflating all the references in the catalog")

	for _, tp := range catalog.TrackingPlans {
		if err := tp.ExpandRefs(catalog); err != nil {
			return fmt.Errorf("expanding refs on tp: %s err: %w", tp.LocalID, err)
		}
	}
	return nil
}

func (p *Provider) SyntacticRules() []rules.Rule {
	syntactic := []rules.Rule{
		propertyRules.NewPropertySpecSyntaxValidRule(),
	}

	return syntactic
}
