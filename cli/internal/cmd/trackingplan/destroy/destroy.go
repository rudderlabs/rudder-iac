package apply

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/spf13/cobra"
)

var log = logger.New("trackingplan", logger.Attr{
	Key:   "cmd",
	Value: "destroy",
})

func NewCmdTPDestroy() *cobra.Command {
	var (
		dryRun  bool
		confirm bool
	)

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Delete all resources from both the upstream catalog and state",
		Long: heredoc.Doc(`
			Delete all resources from both the upstream catalog and state
		`),
		Example: heredoc.Doc(`
			$ rudder-cli tp destroy --dry-run
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			log.Debug("tp destroy", "dryRun", dryRun, "confirm", confirm)

			defer func() {
				telemetry.TrackCommand("tp destroy", err, []telemetry.KV{
					{K: "dryRun", V: dryRun},
					{K: "confirm", V: confirm},
				}...)
			}()

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			s, err := syncer.New(deps.Providers().DataCatalog, &client.Workspace{},
				syncer.WithDryRun(dryRun),
				syncer.WithConfirmationPrompt(confirm),
			)
			if err != nil {
				return err
			}

			errors := s.Destroy(context.Background())
			if len(errors) > 0 {
				return fmt.Errorf("destroying resources: %w", errors[0])
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the changes and not apply them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm the changes before applying")

	return cmd
}
