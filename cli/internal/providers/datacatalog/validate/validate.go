package validate

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

var log = logger.New("validate")

// Match URN patterns: #custom-type:id, #category:id, #property:id, etc.
var urnRegex = regexp.MustCompile(`#[\w-]+:[\w-]+`)

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

// convertURNReferencesToPath transforms URN references in a string back to path-based references
func convertURNReferencesToPath(text string, refMap map[string]string) string {

	return urnRegex.ReplaceAllStringFunc(text, func(urn string) string {
		if pathRef, ok := refMap[urn]; ok {
			return pathRef
		}
		return urn
	})
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
		// Convert URN references back to path-based references for error display
		reference := convertURNReferencesToPath(err.Reference, dc.ReferenceMap)
		errorMsg := convertURNReferencesToPath(err.Error(), dc.ReferenceMap)
		errStr += fmt.Sprintf("\nreference: %s, error: %s\n\n", reference, errorMsg)
	}

	if len(errStr) == 0 {
		return nil
	}

	return errors.New(errStr)
}
