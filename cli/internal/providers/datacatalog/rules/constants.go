package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

const (
	customTypeLegacyReferenceTag     = "legacy_custom_type_ref"
	customTypeReferenceTag           = "custom_type_ref"
	customTypeLegacyReferenceMessage = "must be a valid custom type reference #/custom-types/<group>/<id>"
	customTypeReferenceMessage       = "must be a valid custom type reference #custom-types:<id>"

	propertyLegacyReferenceTag     = "legacy_property_ref"
	propertyReferenceTag           = "property_ref"
	propertyLegacyReferenceMessage = "must be a valid property reference #/properties/<group>/<id>"
	propertyReferenceMessage       = "must be a valid property reference #properties:<id>"

	eventLegacyReferenceTag     = "legacy_event_ref"
	eventReferenceTag           = "event_ref"
	eventLegacyReferenceMessage = "must be a valid event reference #/events/<group>/<id>"
	eventReferenceMessage       = "must be a valid event reference #events:<id>"

	categoryLegacyReferenceTag     = "legacy_category_ref"
	categoryReferenceTag           = "category_ref"
	categoryLegacyReferenceMessage = "must be a valid category reference #/categories/<group>/<id>"
	categoryReferenceMessage       = "must be a valid category reference #categories:<id>"

	trackingPlanLegacyReferenceTag     = "legacy_tracking_plan_ref"
	trackingPlanReferenceTag           = "tracking_plan_ref"
	trackingPlanLegacyReferenceMessage = "must be a valid tracking plan reference #/tp/<group>/<id>"
	trackingPlanReferenceMessage       = "must be a valid tracking plan reference #tracking-plan:<id>"
)

var (
	customTypeLegacyReferenceRegex = fmt.Sprintf(
		`^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`,
		localcatalog.KindCustomTypes,
	)

	customTypeReferenceRegex = fmt.Sprintf(
		`^#(%s):([a-zA-Z0-9_-]+)$`,
		localcatalog.KindCustomTypes,
	)

	propertyLegacyReferenceRegex = fmt.Sprintf(
		`^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`,
		localcatalog.KindProperties,
	)

	propertyReferenceRegex = fmt.Sprintf(
		`^#(%s):([a-zA-Z0-9_-]+)$`,
		localcatalog.KindProperties,
	)

	eventLegacyReferenceRegex = fmt.Sprintf(
		`^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`,
		localcatalog.KindEvents,
	)

	eventReferenceRegex = fmt.Sprintf(
		`^#(%s):([a-zA-Z0-9_-]+)$`,
		localcatalog.KindEvents,
	)

	categoryLegacyReferenceRegex = fmt.Sprintf(
		`^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`,
		localcatalog.KindCategories,
	)

	categoryReferenceRegex = fmt.Sprintf(
		`^#(%s):([a-zA-Z0-9_-]+)$`,
		localcatalog.KindCategories,
	)

	trackingPlanLegacyReferenceRegex = fmt.Sprintf(
		`^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`,
		localcatalog.KindTrackingPlans,
	)

	trackingPlanReferenceRegex = fmt.Sprintf(
		`^#(%s):([a-zA-Z0-9_-]+)$`,
		localcatalog.KindTrackingPlans,
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
