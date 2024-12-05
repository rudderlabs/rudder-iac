package validate

import (
	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

type ValidationError struct {
	error
	Reference string
}

type CatalogValidator interface {
	Validate(*catalog.DataCatalog) []ValidationError
}
