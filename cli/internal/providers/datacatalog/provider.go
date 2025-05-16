package datacatalog

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
	pstate "github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate"
)

var log = logger.New("datacatalogprovider")

type Provider struct {
	*provider.CatalogProvider
	client catalog.DataCatalog
	dc     *localcatalog.DataCatalog
}

func New(client catalog.DataCatalog) *Provider {
	return &Provider{
		CatalogProvider: provider.NewCatalogProvider(client),
		client:          client,
		dc:              localcatalog.New(),
	}
}

func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	return p.dc.LoadSpec(path, s)
}

func (p *Provider) GetSupportedKinds() []string {
	return []string{"properties", "events", "tp"}
}

func (p *Provider) GetSupportedTypes() []string {
	return []string{
		"property",
		"event",
		"trackingplan",
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

	return createResourceGraph(p.dc), nil
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

func createResourceGraph(catalog *localcatalog.DataCatalog) *resources.Graph {
	graph := resources.NewGraph()

	propIDToURN := make(map[string]string)
	for group, props := range catalog.Properties {
		for _, prop := range props {
			log.Debug("adding property to graph", "id", prop.LocalID, "group", group)

			// fmt.Printf("property fromlocal: %+v\n", prop.Config == nil)
			args := pstate.PropertyArgs{
				Name:        prop.Name,
				Description: prop.Description,
				Type:        prop.Type,
				Config:      prop.Config,
			}
			// fmt.Printf("property inargs: %#v\n", args.Config == nil)
			// fmt.Printf("toresourcedata: %#v\n", args.ToResourceData()["config"] == nil)

			resource := resources.NewResource(prop.LocalID, provider.PropertyResourceType, args.ToResourceData(), make([]string, 0))
			graph.AddResource(resource)

			propIDToURN[prop.LocalID] = resource.URN()
		}
	}

	eventIDToURN := make(map[string]string)
	for group, events := range catalog.Events {
		for _, event := range events {
			log.Debug("adding event under group to graph", "event", event.LocalID, "group", group)

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

	for group, tp := range catalog.TrackingPlans {
		log.Debug("adding tracking plan to graph", "tp", tp.LocalID, "group", group)

		args := pstate.TrackingPlanArgs{}
		args.FromCatalogTrackingPlan(tp)

		resource := resources.NewResource(tp.LocalID, provider.TrackingPlanResourceType, args.ToResourceData(), make([]string, 0))
		graph.AddResource(resource)
		graph.AddDependencies(resource.URN(), getDependencies(tp, propIDToURN, eventIDToURN))
	}

	return graph
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
