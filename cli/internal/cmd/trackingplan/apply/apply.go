package apply

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/validate"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
	"github.com/spf13/cobra"
)

func NewCmdTPApply() *cobra.Command {
	var (
		localcatalog *localcatalog.DataCatalog
		err          error
		catalogDir   string
		skipPreview  bool
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply the changes to upstream catalog",
		Long: heredoc.Doc(`
			The tool reads the current state of local catalog defined by the customer. It identifies
			the changes based on the last recorded state. The diff is then applied to the upstream.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli tp apply --loc </path/to/dir or file> --dry-run
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Here we might need to do validate
			localcatalog, err = ReadCatalog(catalogDir)
			if err != nil {
				fmt.Println("reading catalog failed in pre-step: %w", err)
			}
			return validate.ValidateCatalog(validate.DefaultValidators(), localcatalog)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("identifying changes for the upstream catalog")
			// Always inflate the refs before registering the catalog
			//
			graph, err := createResourceGraph(localcatalog)
			if err != nil {
				return fmt.Errorf("creating resource graph: %w", err)
			}

			stateManager := &testutils.MemoryStateManager{}
			stateManager.Save(context.Background(), state.EmptyState())

			p, err := newProvider()
			if err != nil {
				return fmt.Errorf("creating provider: %w", err)
			}

			syncer := syncer.New(p, stateManager)
			if err := syncer.Sync(context.Background(), graph); err != nil {
				return fmt.Errorf("syncing the state: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&catalogDir, "loc", "l", "", "Path to the directory containing the catalog files  or catalog file itself")
	cmd.Flags().BoolVar(&skipPreview, "dry-run", false, "Only show the changes and not apply them")
	return cmd
}

func newProvider() (syncer.Provider, error) {
	rawClient, err := client.New(config.GetAccessToken(), client.WithBaseURL("https://api.staging.rudderlabs.com/v2"))
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return provider.NewPropertyProvider(client.NewRudderDataCatalog(rawClient)), nil
}

func createResourceGraph(catalog *localcatalog.DataCatalog) (*resources.Graph, error) {
	graph := resources.NewGraph()

	for _, props := range catalog.Properties {
		for _, prop := range props {
			fmt.Println("adding property to graph", prop.LocalID)

			graph.AddResource(resources.NewResource(prop.LocalID, "property", prop.GetData()))
		}
	}

	return graph, nil
}

func ReadCatalog(dirLoc string) (*localcatalog.DataCatalog, error) {
	catalog, err := localcatalog.Read(dirLoc)
	if err != nil {
		return nil, fmt.Errorf("reading catalog at location: %w", err)
	}

	return catalog, nil
}
