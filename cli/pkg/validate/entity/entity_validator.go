package entity

import (
	"errors"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

var (
	log = logger.New("entity.validate")
)

var (
	// Common Errors
	ErrDuplicateByID           = errors.New("duplicate entity by id")
	ErrDuplicateByName         = errors.New("duplicate entity by display_name")
	ErrDuplicateByNameType     = errors.New("duplicate entity by name and type")
	ErrMissingRequiredKeysID   = errors.New("missing required key id")
	ErrMissingRequiredKeysName = errors.New("missing required key display_name")

	// Event errors
	ErrInvalidRequiredKeysEventType = errors.New("invalid / missing required key event_type")
	ErrNotAllowedKeyName            = errors.New("display_name field is not allowed on non-track events")

	// Property errors
	ErrInvalidRequiredKeysPropertyType = errors.New("invalid / missing required key type")

	// TrackingPlan errors
	ErrMissingRequiredKeysRuleID        = errors.New("missing required key id in rule")
	ErrInvalidTrackingPlanEventRuleType = errors.New("invalid / missing required key type in rule")
	ErrMissingRequiredKeysRuleEvent     = errors.New("missing required key event in rule")
	ErrInvalidRefFormat                 = errors.New("invalid reference format")
	ErrMissingEntityFromRef             = errors.New("missing entity from reference") // TODO: Fix this ?
	ErrInvalidIdentityApplied           = errors.New("identity section only for non-track events")
	ErrDuplicateEntityRefs              = errors.New("duplicate entity refs")
)

const (
	Event        = "event"
	Property     = "property"
	TrackingPlan = "trackingPlan"
)

type ValidationError struct {
	EntityType string
	Err        error
	Reference  string
}

type CatalogEntity interface {
	*localcatalog.Event | *localcatalog.Property | *localcatalog.TrackingPlan
}

type CatalogEntityValidator interface {
	Validate(*localcatalog.DataCatalog) []ValidationError
}

type TypedCatalogEntityValidator[T CatalogEntity] interface {
	Validate(*localcatalog.DataCatalog) []ValidationError
	RegisterRule(ValidationRule[T])
}

type ValidationRule[T CatalogEntity] interface {
	Validate(string, T, *localcatalog.DataCatalog) []ValidationError
}
