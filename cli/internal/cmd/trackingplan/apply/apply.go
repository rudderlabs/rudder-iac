package apply

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/spf13/cobra"
)

var (
	log = logger.New("trackingplan.apply")
)

func NewCmdTPApply() *cobra.Command {
	var (
		deps     app.Deps
		p        project.Project
		err      error
		location string
		dryRun   bool
		confirm  bool
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
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			p = project.New(location, deps.Providers().DataCatalog)
			if err := p.Load(); err != nil {
				return fmt.Errorf("loading project: %w", err)
			}

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

			workspace, err := deps.Client().Workspaces.GetByAuthToken(context.Background())
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			graph, err := p.GetResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}

			s, err := syncer.New(deps.Providers().DataCatalog, workspace,
				syncer.WithDryRun(dryRun),
				syncer.WithAskConfirmation(confirm),
				syncer.WithReporter(app.SyncReporter()),
			)
			if err != nil {
				return err
			}

			err = s.Sync(context.Background(), graph)
			if err != nil {
				return fmt.Errorf("syncing the state: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", "", "Path to the directory containing the catalog files  or catalog file itself")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the changes and not apply them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm the changes before applying")

	return cmd
}
