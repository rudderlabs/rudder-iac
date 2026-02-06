package model

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

const TypeEventRule = "event_rule"

type ImportableTrackingPlan struct {
	localcatalog.TrackingPlan
}

// ForExport loads the tracking plan from the upstream and returns it in a format
// that can be exported to a file.
func (tp *ImportableTrackingPlan) ForExport(
	externalID string,
	upstream *catalog.TrackingPlanWithIdentifiers,
	resolver resolver.ReferenceResolver,
	idNamer namer.Namer,
) (map[string]any, error) {
	if err := tp.fromUpstream(externalID, upstream, resolver, idNamer); err != nil {
		return nil, fmt.Errorf("loading tracking plan: %w", err)
	}

	toReturn := make(map[string]any)

	byt, err := json.Marshal(tp.TrackingPlan)
	if err != nil {
		return nil, fmt.Errorf("marshalling tracking plan: %w", err)
	}

	if err := json.Unmarshal(byt, &toReturn); err != nil {
		return nil, fmt.Errorf("unmarshalling tracking plan: %w", err)
	}

	return toReturn, nil
}

func (tp *ImportableTrackingPlan) fromUpstream(
	externalID string,
	upstream *catalog.TrackingPlanWithIdentifiers,
	resolver resolver.ReferenceResolver,
	idNamer namer.Namer,
) error {
	tp.TrackingPlan.LocalID = externalID
	tp.TrackingPlan.Name = upstream.Name
	if upstream.Description != nil {
		tp.TrackingPlan.Description = *upstream.Description
	}

	rules := make([]*localcatalog.TPRule, 0, len(upstream.Events))
	for _, event := range upstream.Events {
		eventRef, err := resolver.ResolveToReference(
			types.EventResourceType,
			event.ID,
		)
		if err != nil {
			return fmt.Errorf("resolving reference for event %s: %w", event.ID, err)
		}

		if eventRef == "" {
			return fmt.Errorf("resolved reference is empty for event %s", event.ID)
		}

		ruleProperties, err := buildRuleProperties(event.Properties, resolver)
		if err != nil {
			return fmt.Errorf("building properties for event %s: %w", event.ID, err)
		}

		// Since rules are not first class citizens in catalog,
		// we need to generate the localID for the rule based on the event name and type
		// Product Viewed -> Product Viewed Rule ->product-viewed-rule
		// Identify -> Identify Rule ->identify-rule
		// Alias -> Alias Rule ->alias-rule
		name := event.Name
		if name == "" {
			name = event.EventType
		}
		ruleLocalID, err := idNamer.Name(namer.ScopeName{
			Name:  fmt.Sprintf("%s %s", name, "Rule"),
			Scope: TypeEventRule,
		})
		if err != nil {
			return fmt.Errorf("generating externalID for rule on event %s: %w", event.ID, err)
		}

		var importableVariants ImportableVariants
		if err := importableVariants.fromUpstream(event.Variants, resolver); err != nil {
			return fmt.Errorf("processing variants on event %s: %w", event.ID, err)
		}

		rule := &localcatalog.TPRule{
			Type:    TypeEventRule,
			LocalID: ruleLocalID,
			Event: &localcatalog.TPRuleEvent{
				Ref:             eventRef,
				AllowUnplanned:  event.AdditionalProperties,
				IdentitySection: event.IdentitySection,
			},
			Properties: ruleProperties,
			Variants:   importableVariants.Variants,
		}

		rules = append(rules, rule)
	}

	tp.TrackingPlan.Rules = rules
	return nil
}

// buildRuleProperties recursively builds TPRuleProperty from upstream properties
func buildRuleProperties(
	upstreamProps []*catalog.TrackingPlanEventProperty,
	resolver resolver.ReferenceResolver,
) ([]*localcatalog.TPRuleProperty, error) {
	if len(upstreamProps) == 0 {
		return nil, nil
	}

	ruleProps := make([]*localcatalog.TPRuleProperty, 0, len(upstreamProps))
	for _, prop := range upstreamProps {
		// Resolve property reference
		propRef, err := resolver.ResolveToReference(
			types.PropertyResourceType,
			prop.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("property reference resolution for property %s: %w", prop.ID, err)
		}

		if propRef == "" {
			return nil, fmt.Errorf("resolved property reference is empty for property %s", prop.ID)
		}

		// Recursively build nested properties
		nestedProps, err := buildRuleProperties(prop.Properties, resolver)
		if err != nil {
			return nil, fmt.Errorf("building nested properties for property %s: %w", prop.ID, err)
		}

		ruleProp := &localcatalog.TPRuleProperty{
			Ref:        propRef,
			Required:   prop.Required,
			Properties: nestedProps,
		}
		// only set additionalProperties if there are nested properties
		if len(nestedProps) > 0 {
			// copy the additionalProperties value to ensure we're not setting the address of a loop variable in ruleProps
			additionalProperties := prop.AdditionalProperties
			ruleProp.AdditionalProperties = &additionalProperties
		}
		ruleProps = append(ruleProps, ruleProp)
	}

	return ruleProps, nil
}

type ImportableTrackingPlanV1 struct {
	localcatalog.TrackingPlanV1
}

// ForExport loads the tracking plan from the upstream and returns it in a format
// that can be exported to a file.
func (tp *ImportableTrackingPlanV1) ForExport(
	externalID string,
	upstream *catalog.TrackingPlanWithIdentifiers,
	resolver resolver.ReferenceResolver,
	idNamer namer.Namer,
) (map[string]any, error) {
	if err := tp.fromUpstream(externalID, upstream, resolver, idNamer); err != nil {
		return nil, fmt.Errorf("loading tracking plan: %w", err)
	}

	toReturn := make(map[string]any)

	byt, err := json.Marshal(tp.TrackingPlanV1)
	if err != nil {
		return nil, fmt.Errorf("marshalling tracking plan: %w", err)
	}

	if err := json.Unmarshal(byt, &toReturn); err != nil {
		return nil, fmt.Errorf("unmarshalling tracking plan: %w", err)
	}

	return toReturn, nil
}

func (tp *ImportableTrackingPlanV1) fromUpstream(
	externalID string,
	upstream *catalog.TrackingPlanWithIdentifiers,
	resolver resolver.ReferenceResolver,
	idNamer namer.Namer,
) error {
	tp.TrackingPlanV1.LocalID = externalID
	tp.TrackingPlanV1.Name = upstream.Name
	if upstream.Description != nil {
		tp.TrackingPlanV1.Description = *upstream.Description
	}

	rules := make([]*localcatalog.TPRuleV1, 0, len(upstream.Events))
	for _, event := range upstream.Events {
		eventRef, err := resolver.ResolveToReference(
			types.EventResourceType,
			event.ID,
		)
		if err != nil {
			return fmt.Errorf("resolving reference for event %s: %w", event.ID, err)
		}

		if eventRef == "" {
			return fmt.Errorf("resolved reference is empty for event %s", event.ID)
		}

		ruleProperties, err := buildRulePropertiesV1(event.Properties, resolver)
		if err != nil {
			return fmt.Errorf("building properties for event %s: %w", event.ID, err)
		}

		// Since rules are not first class citizens in catalog,
		// we need to generate the localID for the rule based on the event name and type
		// Product Viewed -> Product Viewed Rule ->product-viewed-rule
		// Identify -> Identify Rule ->identify-rule
		// Alias -> Alias Rule ->alias-rule
		name := event.Name
		if name == "" {
			name = event.EventType
		}
		ruleLocalID, err := idNamer.Name(namer.ScopeName{
			Name:  fmt.Sprintf("%s %s", name, "Rule"),
			Scope: TypeEventRule,
		})
		if err != nil {
			return fmt.Errorf("generating externalID for rule on event %s: %w", event.ID, err)
		}

		var importableVariants ImportableVariantsV1
		if err := importableVariants.fromUpstream(event.Variants, resolver); err != nil {
			return fmt.Errorf("processing variants on event %s: %w", event.ID, err)
		}

		rule := &localcatalog.TPRuleV1{
			Type:                 TypeEventRule,
			LocalID:              ruleLocalID,
			Event:                eventRef,
			AdditionalProperties: event.AdditionalProperties,
			IdentitySection:      event.IdentitySection,
			Properties:           ruleProperties,
			Variants:             importableVariants.VariantsV1,
		}

		rules = append(rules, rule)
	}

	tp.TrackingPlanV1.Rules = rules
	return nil
}

// buildRulePropertiesV1 recursively builds TPRulePropertyV1 from upstream properties
func buildRulePropertiesV1(
	upstreamProps []*catalog.TrackingPlanEventProperty,
	resolver resolver.ReferenceResolver,
) ([]*localcatalog.TPRulePropertyV1, error) {
	if len(upstreamProps) == 0 {
		return nil, nil
	}

	ruleProps := make([]*localcatalog.TPRulePropertyV1, 0, len(upstreamProps))
	for _, prop := range upstreamProps {
		// Resolve property reference
		propRef, err := resolver.ResolveToReference(
			types.PropertyResourceType,
			prop.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("property reference resolution for property %s: %w", prop.ID, err)
		}

		if propRef == "" {
			return nil, fmt.Errorf("resolved property reference is empty for property %s", prop.ID)
		}

		// Recursively build nested properties
		nestedProps, err := buildRulePropertiesV1(prop.Properties, resolver)
		if err != nil {
			return nil, fmt.Errorf("building nested properties for property %s: %w", prop.ID, err)
		}

		ruleProp := &localcatalog.TPRulePropertyV1{
			Property:   propRef,
			Required:   prop.Required,
			Properties: nestedProps,
		}
		// only set additionalProperties if there are nested properties
		if len(nestedProps) > 0 {
			// copy the additionalProperties value to ensure we're not setting the address of a loop variable in ruleProps
			additionalProperties := prop.AdditionalProperties
			ruleProp.AdditionalProperties = &additionalProperties
		}
		ruleProps = append(ruleProps, ruleProp)
	}

	return ruleProps, nil
}
