package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

const (
	legacyRegexPattern    = `^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`
	referenceRegexPattern = `^#(%s):([a-zA-Z0-9_-]+)$`

	customTypeLegacyReferenceTag     = "legacy_custom_type_ref"
	customTypeReferenceTag           = "custom_type_ref"
	customTypeLegacyReferenceMessage = "must be of pattern #/custom-types/<group>/<id>"
	customTypeReferenceMessage       = "must be of pattern #custom-types:<id>"

	propertyLegacyReferenceTag     = "legacy_property_ref"
	propertyReferenceTag           = "property_ref"
	propertyLegacyReferenceMessage = "must be of pattern #/properties/<group>/<id>"
	propertyReferenceMessage       = "must be of pattern #properties:<id>"

	eventLegacyReferenceTag     = "legacy_event_ref"
	eventReferenceTag           = "event_ref"
	eventLegacyReferenceMessage = "must be of pattern #/events/<group>/<id>"
	eventReferenceMessage       = "must be of pattern #events:<id>"

	categoryLegacyReferenceTag     = "legacy_category_ref"
	categoryReferenceTag           = "category_ref"
	categoryLegacyReferenceMessage = "must be of pattern #/categories/<group>/<id>"
	categoryReferenceMessage       = "must be of pattern #categories:<id>"

	trackingPlanLegacyReferenceTag     = "legacy_tracking_plan_ref"
	trackingPlanReferenceTag           = "tracking_plan_ref"
	trackingPlanLegacyReferenceMessage = "must be of pattern #/tp/<group>/<id>"
	trackingPlanReferenceMessage       = "must be of pattern #tracking-plan:<id>"

	// display_name is a shared pattern for human-readable names across resource types
	// (e.g., category name, tracking plan display_name)
	displayNameRegexPattern = `^[A-Z_a-z][ \w,.-]{2,64}$`
	displayNameRegexTag     = "display_name"
	displayNameErrorMessage = "must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"
)

var (
	CustomTypeLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindCustomTypes,
	)

	CustomTypeReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindCustomTypes,
	)

	PropertyLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindProperties,
	)

	PropertyReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindProperties,
	)

	EventLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindEvents,
	)

	EventReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindEvents,
	)

	CategoryLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindCategories,
	)

	CategoryReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindCategories,
	)

	TrackingPlanLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindTrackingPlans,
	)

	TrackingPlanReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindTrackingPlansV1,
	)

	ValidPrimitiveTypes = []string{
		"string", "number", "integer", "boolean", "null", "array", "object",
	}

	ValidFormatValues = []string{
		"date-time", "date", "time", "email", "uuid", "hostname", "ipv4", "ipv6",
	}

	ValidEventTypes = []string{
		"track", "screen", "identify", "group", "page",
	}
)

// In the init function of this package, we will be registering all the
func init() {
	// Register the reference patterns for all the resources in the catalog
	// for both legacy and new reference formats.

	// #/custom-types/<group>/<id>
	funcs.NewPattern(
		customTypeLegacyReferenceTag,
		CustomTypeLegacyReferenceRegex,
		customTypeLegacyReferenceMessage,
	)

	// #custom-types:<id>
	funcs.NewPattern(
		customTypeReferenceTag,
		CustomTypeReferenceRegex,
		customTypeReferenceMessage,
	)

	// #/properties/<group>/<id>
	funcs.NewPattern(
		propertyLegacyReferenceTag,
		PropertyLegacyReferenceRegex,
		propertyLegacyReferenceMessage,
	)

	// #properties:<id>
	funcs.NewPattern(
		propertyReferenceTag,
		PropertyReferenceRegex,
		propertyReferenceMessage,
	)

	// #/events/<group>/<id>
	funcs.NewPattern(
		eventLegacyReferenceTag,
		EventLegacyReferenceRegex,
		eventLegacyReferenceMessage,
	)

	// #events:<id>
	funcs.NewPattern(
		eventReferenceTag,
		EventReferenceRegex,
		eventReferenceMessage,
	)

	// #/categories/<group>/<id>
	funcs.NewPattern(
		categoryLegacyReferenceTag,
		CategoryLegacyReferenceRegex,
		categoryLegacyReferenceMessage,
	)

	// #categories:<id>
	funcs.NewPattern(
		categoryReferenceTag,
		CategoryReferenceRegex,
		categoryReferenceMessage,
	)

	// #/tp/<group>/<id>
	funcs.NewPattern(
		trackingPlanLegacyReferenceTag,
		TrackingPlanLegacyReferenceRegex,
		trackingPlanLegacyReferenceMessage,
	)

	// #tracking-plan:<id>
	funcs.NewPattern(
		trackingPlanReferenceTag,
		TrackingPlanReferenceRegex,
		trackingPlanReferenceMessage,
	)

	// Shared display name pattern for human-readable resource names
	funcs.NewPattern(
		displayNameRegexTag,
		displayNameRegexPattern,
		displayNameErrorMessage,
	)
}
