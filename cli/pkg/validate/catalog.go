package validate

import (
	"fmt"

	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate/entity"
)

var (
	log = logger.New("validate")
)

type ValidationError struct {
	error
	Reference string
}

type CatalogEntityValidator interface {
	Validate(*catalog.DataCatalog) []ValidationError
}

func NewCatalogValidator() *CatalogValidator {

	var validators []entity.CatalogEntityValidator

	validators = append(validators, entity.NewEventValidator([]entity.EventValidationRule{
		&entity.EventRequiredKeysRule{},
		&entity.EventDuplicateKeysRule{},
	}))

	validators = append(validators, entity.NewPropertyEntityValidator([]entity.PropertyValdationRules{
		&entity.PropertyRequiredKeysRule{},
		&entity.PropertyDuplicateKeysRule{},
	}))

	return &CatalogValidator{
		validators: validators,
	}
}

type ValidationErrors []entity.ValidationError

func (errs ValidationErrors) Error() string {
	var str string

	for idx, err := range errs {
		str += fmt.Sprintf("Error: %d\n", idx+1)
		str += fmt.Sprintf("    EntityType: %s\n", err.EntityType)
		str += fmt.Sprintf("    Reference: %s\n", err.Reference)
		str += fmt.Sprintf("    Err: %s\n", err.Err.Error())
		str += "\n"
	}

	return str
}

type CatalogValidator struct {
	validators []entity.CatalogEntityValidator
}

func (c *CatalogValidator) Validate(dc *catalog.DataCatalog) ValidationErrors {
	log.Info("validating the catalog")

	var errors []entity.ValidationError

	for _, validator := range c.validators {
		errors = append(errors, validator.Validate(dc)...)
	}

	return errors
}
