package apply

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/validate"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
	pstate "github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/spf13/cobra"
)

var (
	log = logger.New("trackingplan.apply")
)

func NewCmdTPApply() *cobra.Command {
	var (
		localcatalog *localcatalog.DataCatalog
		s            *syncer.ProjectSyncer
		err          error
		catalogDir   string
		dryRun       bool
		confirm      bool
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply the changes to upstream catalog",
		Long: heredoc.Doc(`
			The tool reads the current state of local catalog defined by the customer. It identifies
			the changes based on the last recorded state. The diff is then applied to the upstream.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli tp apply --location </path/to/dir or file> --dry-run
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Here we might need to do validate
			localcatalog, err = readCatalog(catalogDir)
			if err != nil {
				return fmt.Errorf("reading catalog failed in pre-step: %w", err)
			}

			err = validate.ValidateCatalog(validate.DefaultValidators(), localcatalog)
			if err != nil {
				return fmt.Errorf("validating catalog: %w", err)
			}

			err = inflateRefs(localcatalog)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Debug("tp apply", "dryRun", dryRun, "confirm", confirm)
			log.Debug("identifying changes for the upstream catalog")

			defer func() {
				telemetry.TrackCommand("tp apply", err, []telemetry.KV{
					{K: "dryRun", V: dryRun},
					{K: "confirm", V: confirm},
				}...)
			}()

			s, err = common.NewSyncer()
			if err != nil {
				return err
			}

			err = s.Sync(
				context.Background(),
				createResourceGraph(localcatalog),
				syncer.SyncOptions{
					DryRun:  dryRun,
					Confirm: confirm,
				})

			if err != nil {
				return fmt.Errorf("syncing the state: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&catalogDir, "location", "l", "", "Path to the directory containing the catalog files  or catalog file itself")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the changes and not apply them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm the changes before applying")
	return cmd
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

func readCatalog(dirLoc string) (*localcatalog.DataCatalog, error) {
	catalog, err := localcatalog.Read(dirLoc)
	if err != nil {
		return nil, fmt.Errorf("reading catalog at location: %w", err)
	}
	return catalog, nil
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
