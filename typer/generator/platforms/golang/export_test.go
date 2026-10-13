package golang

import "github.com/rudderlabs/rudder-iac/typer/plan"

// RuntimeSource is the runtime the generator emits, so the golden test checks
// the same text rather than cutting runtime.go again.
var RuntimeSource = runtimeSource

// TrackPlan shares trackPlan with the external tests.
var TrackPlan = trackPlan

// trackPlan is a plan with one track event, "Some Event", declaring properties.
func trackPlan(properties ...plan.Property) *plan.TrackingPlan {
	schema := plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}}
	for _, p := range properties {
		schema.Properties[p.Name] = plan.PropertySchema{Property: p}
	}
	return &plan.TrackingPlan{Rules: []plan.EventRule{{
		Event:   plan.Event{EventType: plan.EventTypeTrack, Name: "Some Event"},
		Section: plan.IdentitySectionProperties,
		Schema:  schema,
	}}}
}
