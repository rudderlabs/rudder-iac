package apply

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
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
			log.Debug("tp destroy", "dryRun", dryRun, "confirm", confirm)
			syncer := syncer.New(app.Provider(), app.StateManager())
			syncer.DryRun = dryRun
			syncer.Confirm = confirm

			errors := syncer.Destroy(context.Background())
			if len(errors) > 0 {
				return fmt.Errorf("syncing the state: %w", errors[0])
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the changes and not apply them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm the changes before applying")
	return cmd
}
