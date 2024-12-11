package validate

import (
	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

var log = logger.New("validate")

type ValidationError struct {
	error
	Reference string
}

type CatalogValidator interface {
	Validate(*catalog.DataCatalog) []ValidationError
}
