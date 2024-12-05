package validate

import (
	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

// RefValidator checks the references in tracking plan to other
// events and properties in data catalog and verifies if the refs are valid
type RefValidator struct {
}

// This is gonna be a tricky one, not sure how to implement this ?
func (rv *RefValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	// mainly gonna be for tracking plan to check if the refs are valid
	return nil
}
