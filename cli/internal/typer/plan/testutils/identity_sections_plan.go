package testutils

import "github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"

// GetIdentitySectionsPlan returns a small plan that pins the identity-section
// matrix, which the main reference plan only covers half of: it has an
// identify on `traits` and a group on `context.traits`, so the other two
// combinations are never generated and never exercised against a real SDK.
//
// The combination that matters most here is identify on `context.traits` —
// that is the shape RudderStack's own webapp generated, and the one that
// silently stopped user traits reaching any event after the identify.
//
// A track event is included so tests can assert what a FOLLOWING event carries.
// Trait persistence is invisible if you only inspect the call you just made.
func GetIdentitySectionsPlan() *plan.TrackingPlan {
	rules := []plan.EventRule{
		// Identify carrying traits through context.traits.
		{
			Event:   *ReferenceEvents["Identify"],
			Section: plan.IdentitySectionContextTraits,
			Schema: plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"email":  {Property: *ReferenceProperties["email"], Required: true},
					"active": {Property: *ReferenceProperties["active"]},
				},
			},
		},
		// Group carrying traits through the SDK traits parameter.
		{
			Event:   *ReferenceEvents["Group"],
			Section: plan.IdentitySectionTraits,
			Schema: plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"active": {Property: *ReferenceProperties["active"], Required: true},
					"status": {Property: *ReferenceProperties["status"]},
				},
			},
		},
		// A plain track event, used to observe what later events carry.
		{
			Event:   *ReferenceEvents["User Signed Up"],
			Section: plan.IdentitySectionProperties,
			Schema: plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"active": {Property: *ReferenceProperties["active"], Required: true},
				},
			},
		},
	}

	return &plan.TrackingPlan{
		Name:  "Identity Sections Plan",
		Rules: rules,
		Metadata: plan.PlanMetadata{
			TrackingPlanID:      "plan_identity_sections",
			TrackingPlanVersion: 1,
			URL:                 "https://app.rudderstack.com/trackingPlans/plan_identity_sections",
		},
	}
}
