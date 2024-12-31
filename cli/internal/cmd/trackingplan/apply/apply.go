package apply

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/validate"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/spf13/cobra"
)

var log = logger.New("trackingplan", logger.Attr{
	Key:   "cmd",
	Value: "apply",
})

func NewCmdTPApply() *cobra.Command {
	var (
		localcatalog *localcatalog.DataCatalog
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
			$ rudder-cli tp apply --loc </path/to/dir or file> --dry-run
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Here we might need to do validate
			localcatalog, err = readCatalog(catalogDir)
			if err != nil {
				return fmt.Errorf("reading catalog failed in pre-step: %w", err)
			}
			return validate.ValidateCatalog(validate.DefaultValidators(), localcatalog)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Debug("tp apply", "dryRun", dryRun, "confirm", confirm)
			log.Debug("identifying changes for the upstream catalog")
			// Always inflate the refs before registering the catalog
			graph, err := createResourceGraph(localcatalog)
			if err != nil {
				return fmt.Errorf("creating resource graph: %w", err)
			}

			syncer := syncer.New(app.Provider(), app.StateManager())
			syncer.DryRun = dryRun
			syncer.Confirm = confirm

			if err := syncer.Sync(context.Background(), graph); err != nil {
				return fmt.Errorf("syncing the state: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&catalogDir, "loc", "l", "", "Path to the directory containing the catalog files  or catalog file itself")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the changes and not apply them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm the changes before applying")
	return cmd
}

func createResourceGraph(catalog *localcatalog.DataCatalog) (*resources.Graph, error) {
	graph := resources.NewGraph()

	for _, props := range catalog.Properties {
		for _, prop := range props {
			log.Debug("adding property to graph", "id", prop.LocalID)

			graph.AddResource(resources.NewResource(prop.LocalID, "property", prop.GetData()))
		}
	}

	return graph, nil
}

func readCatalog(dirLoc string) (*localcatalog.DataCatalog, error) {
	catalog, err := localcatalog.Read(dirLoc)
	if err != nil {
		return nil, fmt.Errorf("reading catalog at location: %w", err)
	}

	return catalog, nil
}
