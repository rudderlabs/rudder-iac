package datacatalog

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
	pstate "github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate"
)

var log = logger.New("datacatalogprovider")

type Provider struct {
	syncer.SyncProvider
	client catalog.DataCatalog
	dc     *localcatalog.DataCatalog
}

func New(client catalog.DataCatalog) *Provider {
	return &Provider{
		SyncProvider: provider.NewCatalogProvider(client),
		client:       client,
		dc:           localcatalog.New(),
	}
}

func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	return p.dc.LoadSpec(path, s)
}

func (p *Provider) GetSupportedKinds() []string {
	return []string{"properties", "events", "tp", "custom-types"}
}

func (p *Provider) GetSupportedTypes() []string {
	return []string{
		provider.PropertyResourceType,
		provider.EventResourceType,
		provider.TrackingPlanResourceType,
		provider.CustomTypeResourceType,
	}
}

func (p *Provider) GetLocalCatalog() *localcatalog.DataCatalog {
	return p.dc
}

func (p *Provider) Validate() error {
	err := validate.ValidateCatalog(p.dc)
	if err == nil {
		log.Info("successfully validated the catalog")
		return nil
	}

	return fmt.Errorf("catalog is invalid: %s", err.Error())
}

func (p *Provider) GetResourceGraph() (*resources.Graph, error) {
	if err := inflateRefs(p.dc); err != nil {
		return nil, fmt.Errorf("inflating refs: %w", err)
	}

	return createResourceGraph(p.dc)
}

func createResourceGraph(catalog *localcatalog.DataCatalog) (*resources.Graph, error) {
	graph := resources.NewGraph()

	// First, pre-calculate all URNs to use for references
	propIDToURN := make(map[string]string)
	for _, props := range catalog.Properties {
		for _, prop := range props {
			propIDToURN[prop.LocalID] = resources.URN(prop.LocalID, provider.PropertyResourceType)
		}
	}

	eventIDToURN := make(map[string]string)
	for _, events := range catalog.Events {
		for _, event := range events {
			eventIDToURN[event.LocalID] = resources.URN(event.LocalID, provider.EventResourceType)
		}
	}

	customTypeIDToURN := make(map[string]string)
	for _, customTypes := range catalog.CustomTypes {
		for _, customType := range customTypes {
			customTypeIDToURN[customType.LocalID] = resources.URN(customType.LocalID, provider.CustomTypeResourceType)
		}
	}

	// Helper function to get URN from reference
	getURNFromRef := func(ref string) string {
		// Format: #/entities/group/id
		parts := strings.Split(ref, "/")
		if len(parts) < 4 {
			return ""
		}

		var (
			entityType = parts[1]
			id         = parts[3]
		)

		switch entityType {
		case "properties":
			return propIDToURN[id]
		case "custom-types":
			return customTypeIDToURN[id]
		default:
			return ""
		}
	}

	// Add properties to the graph
	for group, props := range catalog.Properties {
		for _, prop := range props {
			log.Debug("adding property to graph", "id", prop.LocalID, "group", group)

			args := &pstate.PropertyArgs{}
			if err := args.FromCatalogPropertyType(prop, getURNFromRef); err != nil {
				return nil, fmt.Errorf("creating property args from catalog property: %s, err:%w", prop.LocalID, err)
			}

			resource := resources.NewResource(prop.LocalID, provider.PropertyResourceType, args.ToResourceData(), make([]string, 0))
			graph.AddResource(resource)

			propIDToURN[prop.LocalID] = resource.URN()
		}
	}

	// Add events to the graph
	for group, events := range catalog.Events {
		for _, event := range events {
			log.Debug("adding event to graph", "event", event.LocalID, "group", group)

			args := pstate.EventArgs{
				Name:        event.Name,
				Description: event.Description,
				EventType:   event.Type,
				CategoryID:  nil,
			}
			resource := resources.NewResource(event.LocalID, provider.EventResourceType, args.ToResourceData(), make([]string, 0))
			graph.AddResource(resource)

			eventIDToURN[event.LocalID] = resource.URN()
		}
	}

	// Add custom types to the graph with dependencies on properties or other custom types
	for group, customTypes := range catalog.CustomTypes {
		for _, customType := range customTypes {
			log.Debug("adding custom type to graph", "id", customType.LocalID, "group", group)

			// Add CustomTypeArgs
			args := pstate.CustomTypeArgs{}
			args.FromCatalogCustomType(&customType, getURNFromRef)
			resource := resources.NewResource(customType.LocalID, provider.CustomTypeResourceType, args.ToResourceData(), make([]string, 0))
			graph.AddResource(resource)
		}
	}

	// Add tracking plans to the graph
	for group, tp := range catalog.TrackingPlans {
		log.Debug("adding tracking plan to graph", "tp", tp.LocalID, "group", group)

		args := pstate.TrackingPlanArgs{}
		if err := args.FromCatalogTrackingPlan(tp, getURNFromRef); err != nil {
			return nil, fmt.Errorf("creating tracking plan args: %w", err)
		}

		resource := resources.NewResource(tp.LocalID, provider.TrackingPlanResourceType, args.ToResourceData(), make([]string, 0))
		graph.AddResource(resource)
		graph.AddDependencies(resource.URN(), getDependencies(tp, propIDToURN, eventIDToURN))
	}

	return graph, nil
}

// getDependencies simply fetch the dependencies on the trackingplan in form of the URN's
// of the properties and events that are used in the tracking plan
func getDependencies(tp *localcatalog.TrackingPlan, propIDToURN, eventIDToURN map[string]string) []string {
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
