package push

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	rdc "github.com/rudderlabs/rudder-data-catalog-provider/sdk/go/rudderdatacatalog/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/iac"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/validate"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/spf13/cobra"
)

func NewCmdTPPush(store *iac.Store) *cobra.Command {
	var (
		localcatalog *localcatalog.DataCatalog
		err          error
		catalogDir   string
		test         bool
	)

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Pushes local catalog to upstream",
		Long: heredoc.Doc(`
			The tool reads the current state of local catalog defined by the customer. It identifies
			the changes based on the last recorded state. The diff is then applied to the upstream.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli tp push --test=true --loc </path/to/dir or file>
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {

			// Lazily setup the pulumi stack
			if err := store.Pulumi.Setup(context.Background()); err != nil {
				return fmt.Errorf("setting up pulumi store lazily: %w", err)
			}

			// Here we might need to do validate
			localcatalog, err = ReadCatalog(catalogDir)
			if err != nil {
				fmt.Println("reading catalog failed in pre-step: %w", err)
			}
			return validate.ValidateCatalog(validate.DefaultValidators(), localcatalog)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Spinning the stack using the activation API", os.Getenv("RUDDER_ACCESS_TOKEN"))

			ctx := context.Background()
			os.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

			// TODO: Remove the hardcoded value
			// This is where we register our resources to the pulumi stack
			s, err := auto.UpsertStackInlineSource(
				ctx,
				store.Pulumi.Conf().GetQualifiedStack(),
				store.Pulumi.Conf().GetProject(),
				func() func(*pulumi.Context) error {
					return func(ctx *pulumi.Context) error {
						return RegisterCatalog(ctx, localcatalog)
					}
				}(),
				auto.Pulumi(store.Pulumi.PulumiCommand()),
				auto.PulumiHome(store.Pulumi.HomeDir()),
				auto.WorkDir(store.Pulumi.WorkDir()),
			)

			if err != nil {
				return fmt.Errorf("creating instance of the stack: %w", err)
			}

			if _, err := s.Refresh(ctx); err != nil {
				return fmt.Errorf("refreshing the stack: %w", err)
			}

			if test {
				_, err = s.Preview(ctx, optpreview.ProgressStreams(os.Stdout))
			} else {
				_, err = s.Up(ctx, optup.ProgressStreams(os.Stdout))
			}

			if err != nil {
				return fmt.Errorf("running the stack: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&catalogDir, "loc", "l", "", "Path to the directory containing the catalog files  or catalog file itself")
	cmd.Flags().BoolVarP(&test, "test", "t", true, "Run the command in test mode to preview changes")
	return cmd
}

func ReadCatalog(dirLoc string) (*localcatalog.DataCatalog, error) {
	catalog, err := localcatalog.Read(dirLoc)
	if err != nil {
		return nil, fmt.Errorf("reading catalog at location: %w", err)
	}

	return catalog, nil
}

func RegisterCatalog(ctx *pulumi.Context, dc *localcatalog.DataCatalog) error {
	// Lookup resource which is needed when registering tracking plan
	// to allow for depends upon relationship
	propResources := make(map[string]pulumi.Resource)
	if err := registerProperties(ctx, dc.Properties, propResources); err != nil {
		return fmt.Errorf("registering properties: %w", err)
	}

	// These lookup maps will help build dependsOn array
	eventResources := make(map[string]pulumi.Resource)
	if err := registerEvents(ctx, dc.Events, eventResources); err != nil {
		return fmt.Errorf("registering events: %w", err)
	}

	if err := registerTrackingPlans(ctx, dc.TrackingPlans, eventResources, propResources); err != nil {
		return fmt.Errorf("registering tracking plans: %w", err)
	}

	// Register the tracking plans with rules attached to them
	return nil
}

func registerProperties(
	ctx *pulumi.Context,
	toAdd map[localcatalog.EntityGroup][]localcatalog.Property,
	propResources map[string]pulumi.Resource) error {
	//add all properties within each logical entity group
	for _, propGroup := range toAdd {
		for _, prop := range propGroup {
			resource, err := rdc.NewProperty(
				ctx,
				prop.LocalID,
				&rdc.PropertyArgs{
					Name:        pulumi.String(prop.Name),
					Description: pulumi.String(prop.Description),
					Type:        pulumi.String(prop.Type),
					PropConfig:  pulumi.ToMap(prop.Config),
				})

			if err != nil {
				return fmt.Errorf("creating property in catalog: %s", err.Error())
			}
			propResources[prop.LocalID] = resource
		}
	}
	return nil
}

func registerEvents(
	ctx *pulumi.Context,
	toAdd map[localcatalog.EntityGroup][]localcatalog.Event,
	eventResources map[string]pulumi.Resource) error {
	// register events into the system
	for _, eventGroup := range toAdd {
		for _, event := range eventGroup {

			resource, err := rdc.NewEvent(
				ctx,
				event.LocalID,
				&rdc.EventArgs{
					Name:        pulumi.String(event.Name),
					Description: pulumi.String(event.Description),
					EventType:   pulumi.String(event.Type),
				},
			)
			if err != nil {
				return fmt.Errorf("creating event in catalog: %s", err.Error())
			}
			eventResources[event.LocalID] = resource
		}
	}
	return nil
}

func registerTrackingPlans(
	ctx *pulumi.Context,
	toAdd map[localcatalog.EntityGroup]localcatalog.TrackingPlan,
	eventResources map[string]pulumi.Resource,
	propResources map[string]pulumi.Resource) error {

	for _, tp := range toAdd {
		// collection of event and associated property resources
		// within the event aspect of trackingplan
		dependsOn := make(map[string]pulumi.Resource, 0) // to be used to create a dependency relationship
		eventargs := make(rdc.TPEventMap)
		for _, eventProps := range tp.EventProps {

			propargs := make(rdc.TPEventPropertyMap, 0)
			for _, prop := range eventProps.Properties {
				propargs[prop.LocalID] = rdc.TPEventPropertyArgs{
					Name:     pulumi.String(prop.Name),
					Required: pulumi.Bool(prop.Required),
					LocalId:  pulumi.String(prop.LocalID),
					Config:   pulumi.ToMap(prop.Config),
					Type:     pulumi.ToStringArray([]string{prop.Type}),
				}
				dependsOn[prop.LocalID] = propResources[prop.LocalID]
			}

			eventargs[eventProps.LocalID] = rdc.TPEventArgs{
				Name:           pulumi.String(eventProps.Name),
				Description:    pulumi.String(eventProps.Description),
				LocalId:        pulumi.String(eventProps.LocalID),
				EventType:      pulumi.String(eventProps.Type),
				AllowUnplanned: pulumi.Bool(eventProps.AllowUnplanned),
				Properties:     propargs,
			}

			dependsOn[eventProps.LocalID] = eventResources[eventProps.LocalID]
		}

		_, err := rdc.NewTrackingPlan(ctx, tp.LocalID, &rdc.TrackingPlanArgs{
			Name:        pulumi.String(tp.Name),
			Description: pulumi.String(tp.Description),
			Events:      eventargs,
		}, pulumi.DependsOn(getResources(dependsOn)))

		if err != nil {
			return fmt.Errorf("creating trackingplan in catalog: %s", err.Error())
		}
	}
	return nil
}

func getResources(ip map[string]pulumi.Resource) []pulumi.Resource {
	resources := make([]pulumi.Resource, 0)
	for _, val := range ip {
		resources = append(resources, val)
	}
	return resources
}
