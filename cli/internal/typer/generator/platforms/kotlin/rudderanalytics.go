package kotlin

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

// sectionToParamName converts an EventRuleSection to the appropriate parameter name
func sectionToParamName(section plan.IdentitySection) (string, error) {
	switch section {
	case plan.IdentitySectionProperties:
		return "properties", nil
	case plan.IdentitySectionTraits:
		return "traits", nil
	default:
		return "", fmt.Errorf("unknown event rule section: %s", section)
	}
}

// createRudderAnalyticsMethod creates a single RudderAnalyticsMethod from a plan.Event
func createRudderAnalyticsMethod(rule *plan.EventRule, nameRegistry *core.NameRegistry) (*RudderAnalyticsMethod, error) {
	method := &RudderAnalyticsMethod{
		Comment: rule.Event.Description,
	}

	var err error
	switch rule.Event.EventType {
	case plan.EventTypeTrack:
		err = buildTrackMethod(rule, method, nameRegistry)
	case plan.EventTypeIdentify:
		err = buildIdentifyMethod(rule, method, nameRegistry)
	case plan.EventTypeGroup:
		err = buildGroupMethod(rule, method, nameRegistry)
	case plan.EventTypeScreen:
		err = buildScreenMethod(rule, method, nameRegistry)
	default:
		return nil, nil // Skip page events
	}

	if err != nil {
		return nil, err
	}

	return method, nil
}

// buildTrackMethod configures a RudderAnalyticsMethod for a 'track' event
func buildTrackMethod(rule *plan.EventRule, method *RudderAnalyticsMethod, nameRegistry *core.NameRegistry) error {
	method.Name = FormatMethodName("track", rule.Event.Name)
	className, err := getOrRegisterEventDataClassName(rule, nameRegistry)
	if err != nil {
		return err
	}

	paramName, err := sectionToParamName(rule.Section)
	if err != nil {
		return err
	}

	method.MethodArguments = []KotlinMethodArgument{
		{Name: paramName, Type: className, Nullable: false},
	}
	method.SDKCall = SDKCall{
		MethodName: "track",
		Arguments: []SDKCallArgument{
			// TODO: Handle proper escaping of event name
			{Name: "name", Value: fmt.Sprintf("\"%s\"", rule.Event.Name)},
			{Name: paramName, Value: paramName, ShouldSerialize: true},
		},
	}
	return nil
}

// buildIdentifyMethod configures a RudderAnalyticsMethod for an 'identify' event
func buildIdentifyMethod(rule *plan.EventRule, method *RudderAnalyticsMethod, nameRegistry *core.NameRegistry) error {
	method.Name = "identify"
	className, err := getOrRegisterEventDataClassName(rule, nameRegistry)
	if err != nil {
		return err
	}

	paramName, err := sectionToParamName(rule.Section)
	if err != nil {
		return err
	}

	method.MethodArguments = []KotlinMethodArgument{
		{Name: "userId", Type: "String", Default: "\"\""},
		{Name: paramName, Type: className},
	}
	method.SDKCall = SDKCall{
		MethodName: "identify",
		Arguments: []SDKCallArgument{
			{Name: "userId", Value: "userId"},
			{Name: paramName, Value: paramName, ShouldSerialize: true},
		},
	}
	return nil
}

// buildGroupMethod configures a RudderAnalyticsMethod for a 'group' event
func buildGroupMethod(rule *plan.EventRule, method *RudderAnalyticsMethod, nameRegistry *core.NameRegistry) error {
	method.Name = "group"
	className, err := getOrRegisterEventDataClassName(rule, nameRegistry)
	if err != nil {
		return err
	}

	paramName, err := sectionToParamName(rule.Section)
	if err != nil {
		return err
	}

	method.MethodArguments = []KotlinMethodArgument{
		{Name: "groupId", Type: "String"},
		{Name: paramName, Type: className},
	}
	method.SDKCall = SDKCall{
		MethodName: "group",
		Arguments: []SDKCallArgument{
			{Name: "groupId", Value: "groupId"},
			{Name: paramName, Value: paramName, ShouldSerialize: true},
		},
	}
	return nil
}

// buildScreenMethod configures a RudderAnalyticsMethod for a 'screen' event
func buildScreenMethod(rule *plan.EventRule, method *RudderAnalyticsMethod, nameRegistry *core.NameRegistry) error {
	method.Name = "screen"
	className, err := getOrRegisterEventDataClassName(rule, nameRegistry)
	if err != nil {
		return err
	}

	paramName, err := sectionToParamName(rule.Section)
	if err != nil {
		return err
	}

	method.MethodArguments = []KotlinMethodArgument{
		{Name: "screenName", Type: "String", Nullable: false},
		{Name: "category", Type: "String", Default: "\"\""},
		{Name: paramName, Type: className, Nullable: false},
	}
	method.SDKCall = SDKCall{
		MethodName: "screen",
		Arguments: []SDKCallArgument{
			{Name: "screenName", Value: "screenName"},
			{Name: "category", Value: "category"},
			{Name: paramName, Value: paramName, ShouldSerialize: true},
		},
	}
	return nil
}
