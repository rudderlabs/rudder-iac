package testutils

import "github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"

// GetEmptyIdentityPlan returns a plan whose identify and group rules carry no
// properties. That takes a third emit path through the generator — neither
// dispatcher branch, but the no-traits shape, which exposes no traits parameter
// at all.
//
// It needs its own plan because identify and group are singletons: a plan can
// only demonstrate one shape of each. The path matters because whatever it
// passes in the SDK's traits slot applies to traits the caller never supplied
// and cannot see — set by an untyped analytics.identify, another plan, or an
// earlier session.
func GetEmptyIdentityPlan() *plan.TrackingPlan {
	rules := []plan.EventRule{
		{
			Event:   *ReferenceEvents["Identify"],
			Section: plan.IdentitySectionContextTraits,
			Schema:  plan.ObjectSchema{},
		},
		{
			Event:   *ReferenceEvents["Group"],
			Section: plan.IdentitySectionTraits,
			Schema:  plan.ObjectSchema{},
		},
		// Present so tests can observe what a FOLLOWING event carries.
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
		Name:  "Empty Identity Plan",
		Rules: rules,
		Metadata: plan.PlanMetadata{
			TrackingPlanID:      "plan_empty_identity",
			TrackingPlanVersion: 1,
			URL:                 "https://app.rudderstack.com/trackingPlans/plan_empty_identity",
		},
	}
}
