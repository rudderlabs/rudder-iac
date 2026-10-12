package golang

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/typer/plan"
)

// validSection reports whether the rule's section is one its event type
// takes, as on every platform: properties for track, page and screen, traits
// or context.traits for identify and group. Ported from the Kotlin
// generator's validateEventRuleSection.
func validSection(rule *plan.EventRule) bool {
	switch rule.Event.EventType {
	case plan.EventTypeTrack, plan.EventTypePage, plan.EventTypeScreen:
		return rule.Section == plan.IdentitySectionProperties
	case plan.EventTypeIdentify, plan.EventTypeGroup:
		return rule.Section == plan.IdentitySectionTraits || rule.Section == plan.IdentitySectionContextTraits
	}
	return false
}

// sectionWarning reports a rule skipped for its section, in the Kotlin
// generator's words.
func sectionWarning(rule *plan.EventRule) string {
	msg := fmt.Sprintf("invalid section %q for event type %q", rule.Section, rule.Event.EventType)
	if rule.Event.Name != "" {
		msg += fmt.Sprintf(" and event name %q", rule.Event.Name)
	}
	return msg
}
