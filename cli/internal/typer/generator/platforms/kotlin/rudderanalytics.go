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

// shouldIncludePropertiesParameter checks if a properties/traits parameter should be included
// Returns false only for empty schemas without additionalProperties
func shouldIncludePropertiesParameter(rule *plan.EventRule) bool {
	isEmpty := len(rule.Schema.Properties) == 0
	return !isEmpty || rule.Schema.AdditionalProperties
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

	method.MethodArguments = []KotlinMethodArgument{}

	method.SDKCall = SDKCall{
		MethodName: "track",
		Arguments: []SDKCallArgument{
			// TODO: Handle proper escaping of event name
			{Name: "name", Value: fmt.Sprintf("\"%s\"", rule.Event.Name)},
		},
	}

	if shouldIncludePropertiesParameter(rule) {
		method.MethodArguments = append(method.MethodArguments,
			KotlinMethodArgument{Name: paramName, Type: className, Nullable: false})

		method.SDKCall.Arguments = append(method.SDKCall.Arguments,
			SDKCallArgument{Name: paramName, Value: paramName, ShouldSerialize: true})
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
	}
	method.SDKCall = SDKCall{
		MethodName: "identify",
		Arguments: []SDKCallArgument{
			{Name: "userId", Value: "userId"},
		},
	}

	if shouldIncludePropertiesParameter(rule) {
		method.MethodArguments = append(method.MethodArguments,
			KotlinMethodArgument{Name: paramName, Type: className})

		method.SDKCall.Arguments = append(method.SDKCall.Arguments,
			SDKCallArgument{Name: paramName, Value: paramName, ShouldSerialize: true})
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
	}
	method.SDKCall = SDKCall{
		MethodName: "group",
		Arguments: []SDKCallArgument{
			{Name: "groupId", Value: "groupId"},
		},
	}

	if shouldIncludePropertiesParameter(rule) {
		method.MethodArguments = append(method.MethodArguments,
			KotlinMethodArgument{Name: paramName, Type: className})

		method.SDKCall.Arguments = append(method.SDKCall.Arguments,
			SDKCallArgument{Name: paramName, Value: paramName, ShouldSerialize: true})
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
	}
	method.SDKCall = SDKCall{
		MethodName: "screen",
		Arguments: []SDKCallArgument{
			{Name: "screenName", Value: "screenName"},
			{Name: "category", Value: "category"},
		},
	}

	if shouldIncludePropertiesParameter(rule) {
		method.MethodArguments = append(method.MethodArguments,
			KotlinMethodArgument{Name: paramName, Type: className})

		method.SDKCall.Arguments = append(method.SDKCall.Arguments,
			SDKCallArgument{Name: paramName, Value: paramName, ShouldSerialize: true})
	}
	return nil
}
