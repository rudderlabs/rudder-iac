package validate

import (
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
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
	var catalogDir string

	validators := DefaultValidators()
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate locally defined catalog",
		Long:  "Validate locally defined catalog",
		Example: heredoc.Doc(`
			$ rudder-cli tp validate --loc <path-to-catalog-dir or file>
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, err := localcatalog.Read(catalogDir)
			if err != nil {
				return fmt.Errorf("reading catalog: %s", err.Error())
			}

			err = ValidateCatalog(validators, dc)
			if err == nil {
				log.Info("successfully validated the catalog")
				return nil
			}

			return fmt.Errorf("catalog is invalid: %s", err.Error())
		},
	}

	cmd.Flags().StringVarP(&catalogDir, "loc", "l", "", "Path to the directory containing the catalog files or catalog file itself")
	return cmd
}

func ValidateCatalog(validators []validate.CatalogValidator, dc *localcatalog.DataCatalog) (toReturn error) {
	log.Info("running validators on the catalog")

	combinedErrs := make([]validate.ValidationError, 0)
	for _, validator := range validators {
		errs := validator.Validate(dc)
		if len(errs) > 0 {
			combinedErrs = append(combinedErrs, errs...)
		}
	}

	errStr := ""
	for _, err := range combinedErrs {
		errStr += fmt.Sprintf("\nreference: %s, error: %s\n\n", err.Reference, err.Error())
	}

	if len(errStr) == 0 {
		return nil
	}

	return errors.New(errStr)
}

func DefaultValidators() []validate.CatalogValidator {
	return []validate.CatalogValidator{
		&validate.RequiredKeysValidator{},
		&validate.DuplicateNameIDKeysValidator{},
		&validate.RefValidator{},
	}
}
