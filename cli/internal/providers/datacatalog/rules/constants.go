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
)

var (
	customTypeLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindCustomTypes,
	)

	customTypeReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindCustomTypes,
	)

	propertyLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindProperties,
	)

	propertyReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindProperties,
	)

	eventLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindEvents,
	)

	eventReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindEvents,
	)

	categoryLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindCategories,
	)

	categoryReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindCategories,
	)

	trackingPlanLegacyReferenceRegex = fmt.Sprintf(
		legacyRegexPattern,
		localcatalog.KindTrackingPlans,
	)

	trackingPlanReferenceRegex = fmt.Sprintf(
		referenceRegexPattern,
		localcatalog.KindTrackingPlansV1,
	)
)

// In the init function of this package, we will be registering all the
func init() {
	// Register the reference patterns for all the resources in the catalog
	// for both legacy and new reference formats.

	// #/custom-types/<group>/<id>
	funcs.NewPattern(
		customTypeLegacyReferenceTag,
		customTypeLegacyReferenceRegex,
		customTypeLegacyReferenceMessage,
	)

	// #custom-types:<id>
	funcs.NewPattern(
		customTypeReferenceTag,
		customTypeReferenceRegex,
		customTypeReferenceMessage,
	)

	// #/properties/<group>/<id>
	funcs.NewPattern(
		propertyLegacyReferenceTag,
		propertyLegacyReferenceRegex,
		propertyLegacyReferenceMessage,
	)

	// #properties:<id>
	funcs.NewPattern(
		propertyReferenceTag,
		propertyReferenceRegex,
		propertyReferenceMessage,
	)

	// #/events/<group>/<id>
	funcs.NewPattern(
		eventLegacyReferenceTag,
		eventLegacyReferenceRegex,
		eventLegacyReferenceMessage,
	)

	// #events:<id>
	funcs.NewPattern(
		eventReferenceTag,
		eventReferenceRegex,
		eventReferenceMessage,
	)

	// #/categories/<group>/<id>
	funcs.NewPattern(
		categoryLegacyReferenceTag,
		categoryLegacyReferenceRegex,
		categoryLegacyReferenceMessage,
	)

	// #categories:<id>
	funcs.NewPattern(
		categoryReferenceTag,
		categoryReferenceRegex,
		categoryReferenceMessage,
	)

	// #/tp/<group>/<id>
	funcs.NewPattern(
		trackingPlanLegacyReferenceTag,
		trackingPlanLegacyReferenceRegex,
		trackingPlanLegacyReferenceMessage,
	)

	// #tracking-plan:<id>
	funcs.NewPattern(
		trackingPlanReferenceTag,
		trackingPlanReferenceRegex,
		trackingPlanReferenceMessage,
	)
}
