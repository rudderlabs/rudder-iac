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

// CatalogValidator is a validator for the entire catalog
// which includes setting up entity validators with their rules for better
// validation control
func NewCatalogValidator() *CatalogValidator {

	eventValidator := &entity.EventValidator{}
	eventValidator.RegisterRule(&entity.EventRequiredKeysRule{})
	eventValidator.RegisterRule(&entity.EventDuplicateKeysRule{})

	propValidator := &entity.PropertyEntityValidator{}
	propValidator.RegisterRule(&entity.PropertyRequiredKeysRule{})
	propValidator.RegisterRule(&entity.PropertyDuplicateKeysRule{})

	tpValidator := &entity.TrackingPlanEntityValidator{}
	tpValidator.RegisterRule(&entity.TrackingPlanRequiredKeysRule{})
	tpValidator.RegisterRule(&entity.TrackingPlanRefRule{})
	tpValidator.RegisterRule(&entity.TrackingPlanDuplicateKeysRule{})

	return &CatalogValidator{
		validators: []entity.CatalogEntityValidator{
			eventValidator,
			tpValidator,
			propValidator,
		},
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
