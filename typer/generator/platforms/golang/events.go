package golang

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

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

// payloadNoun is what an event type's payload holds, which ends its type's
// name and names its conversion method: Traits for identify and group,
// Properties for the others.
func payloadNoun(t plan.EventType) string {
	if t == plan.EventTypeIdentify || t == plan.EventTypeGroup {
		return "Traits"
	}
	return "Properties"
}

// eventLabel names an event in doc comments: `the track event "Name"`, or
// "the identify event" for an event without a name.
func eventLabel(event plan.Event) string {
	label := fmt.Sprintf("the %s event", event.EventType)
	if event.Name != "" {
		label += " " + strconv.Quote(event.Name)
	}
	return label
}

// eventMethod returns the method name of rule r, which sends payload (nil
// when the rule's schema is empty and closed) through its event type's send
// helper. Only a track payload is required: the others may be nil, which
// sends none.
func eventMethod(packageName string, r eventRule, name string, payload *GoStruct) GoMethod {
	var (
		event = r.rule.Event
		track = event.EventType == plan.EventTypeTrack
		nouns = strings.ToLower(payloadNoun(event.EventType))
		m     = GoMethod{Name: name, Event: event.Name, Type: string(event.EventType), Payload: payload, Param: "props", Var: "p"}
		// args are the send helper's arguments before the payload.
		args  = []string{"identity"}
		notes []string
	)
	if nouns == "traits" {
		m.Param, m.Var = "traits", "t"
	}
	switch event.EventType {
	case plan.EventTypeTrack:
		args = append(args, strconv.Quote(event.Name))
	case plan.EventTypeGroup:
		args = append(args, "groupID")
		m.Params = append(m.Params, "groupID string")
		m.Example = append(m.Example, `"group-123"`)
	}
	send := func(value string) string {
		a := append(slices.Clone(args), value)
		if event.EventType == plan.EventTypeIdentify {
			a = append(a, strconv.FormatBool(r.rule.Section == plan.IdentitySectionContextTraits))
		}
		return fmt.Sprintf("r.%s(%s, opts)", event.EventType, strings.Join(a, ", "))
	}
	m.Send, m.OnNil = send(m.Var), send("nil")

	summary := fmt.Sprintf("%s sends %s", name, eventLabel(event))
	switch {
	case payload == nil:
		summary += ", which has no " + nouns
	case payload.MapAlias:
		notes = append(notes, m.Param+" may be nil; it is copied, so it can be reused after the call.")
	case payload.Variant != nil && track:
		m.OnNil = fmt.Sprintf(`fmt.Errorf("%%w: %s needs a %s case", ErrNilPayload)`, name, payload.Name)
		notes = append(notes, "props must hold one of the case structs; nil, including a nil pointer, returns ErrNilPayload.")
	case payload.Variant != nil:
		notes = append(notes, fmt.Sprintf("%s may be nil, including a nil pointer, which sends no %s.", m.Param, nouns))
	case !track:
		m.Pointer = true
		notes = append(notes, fmt.Sprintf("%s may be nil, which sends no %s.", m.Param, nouns))
	}
	if r.rule.Section == plan.IdentitySectionContextTraits && event.EventType == plan.EventTypeIdentify {
		notes = append(notes, "The traits are sent as context.traits, where they win over the traits WithAnalyticsContext sets.")
	}

	if payload != nil {
		var (
			typ     = payload.Name
			example = payload.Name
		)
		if payload.Variant != nil {
			example = payload.Variant.Implementations[0]
		}
		example = packageName + "." + example + "{ /* ... */ }"
		if m.Pointer {
			typ, example = "*"+typ, "&"+example
		}
		m.Params = append(m.Params, m.Param+" "+typ)
		m.Example = append(m.Example, example)
	}
	m.Doc = withDescription(strings.Join(append([]string{summary + "."}, notes...), " "), event.Description)
	return m
}
