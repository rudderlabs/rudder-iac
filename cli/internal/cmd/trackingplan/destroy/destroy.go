package destroy

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/iac"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	log = logger.New("trackingplan", logger.Attr{
		Key:   "cmd",
		Value: "destroy",
	})
)

func NewCmdTPDestroy(store *iac.Store) *cobra.Command {

	var (
		skipCheck bool
	)

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Remove all the created resources upstream based on the state",
		Long: heredoc.Doc(`
			It reads and parses the state file and removes all the resources
			from upstream which are defined in the state file. Once executed the
			upstream resources will be unrecoverable
		`),
		Example: heredoc.Doc(`
			$ rudder-cli tp destroy --skip-check=true
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			log.Info("setting up the iac tool stack lazily")

			// Lazily setup the pulumi stack
			if err := store.Pulumi.Setup(context.Background()); err != nil {
				return fmt.Errorf("setting up iac stack lazily: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("destroying the upstream resources whose state is managed by the tool")

			ctx := context.Background()
			os.Setenv("PULUMI_CONFIG_PASSPHRASE", "")
			// TODO: Remove the hardcoded value
			// This is where we register our resources to the pulumi stack
			s, err := auto.UpsertStackInlineSource(
				ctx,
				store.Pulumi.Conf().GetQualifiedStack(),
				store.Pulumi.Conf().GetProject(),
				nil,
				auto.Pulumi(store.Pulumi.PulumiCommand()),
				auto.WorkDir(store.Pulumi.WorkDir()),
				auto.PulumiHome(store.Pulumi.HomeDir()),
			)

			if err != nil {
				return fmt.Errorf("creating instance of the stack: %w", err)
			}

			if _, err := s.Refresh(ctx); err != nil {
				return fmt.Errorf("refreshing the stack: %w", err)
			}

			var yes = true
			if !skipCheck {
				yes = utils.AreYouSure("Destroy will permanently remove upstream resources, continue ?")
			}

			if !yes {
				fmt.Println("Operation aborted")
				return nil
			}

			_, err = s.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))
			if err != nil {
				return fmt.Errorf("destroying the stack: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&skipCheck, "skip-check", false, "Skip the confirmation prompt")
	return cmd
}
