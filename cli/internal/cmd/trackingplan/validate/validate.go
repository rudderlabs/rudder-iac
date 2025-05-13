package validate

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate"
	"github.com/spf13/cobra"
)

var (
	log = logger.New("trackingplan", logger.Attr{
		Key:   "cmd",
		Value: "validate",
	})
)

func NewCmdTPValidate() *cobra.Command {
	var (
		location string
		err      error
		dc       *localcatalog.DataCatalog
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate locally defined catalog",
		Long:  "Validate locally defined catalog",
		Example: heredoc.Doc(`
			$ rudder-cli tp validate --location <path-to-catalog-dir or file>
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("tp validate", err)
			}()

			loader := loader.New(location)
			specs, err := loader.Load()
			if err != nil {
				return fmt.Errorf("loading catalog: %s", err.Error())
			}

			dc, err = localcatalog.New(specs)
			if err != nil {
				return fmt.Errorf("reading catalog: %s", err.Error())
			}

			err = validate.ValidateCatalog(dc)
			if err == nil {
				log.Info("successfully validated the catalog")
				return nil
			}

			err = fmt.Errorf("catalog is invalid: %s", err.Error())
			return err
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", "", "Path to the directory containing the catalog files or catalog file itself")
	return cmd
}
