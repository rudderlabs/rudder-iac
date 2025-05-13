package validate

import (
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

var log = logger.New("validate")

type ValidationError struct {
	error
	Reference string
}

type CatalogValidator interface {
	Validate(*localcatalog.DataCatalog) []ValidationError
}

func DefaultValidators() []CatalogValidator {
	return []CatalogValidator{
		&RequiredKeysValidator{},
		&DuplicateNameIDKeysValidator{},
		&RefValidator{},
	}
}

func ValidateCatalog(dc *localcatalog.DataCatalog) (toReturn error) {
	log.Info("running validators on the catalog")

	combinedErrs := make([]ValidationError, 0)
	validators := DefaultValidators()
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
