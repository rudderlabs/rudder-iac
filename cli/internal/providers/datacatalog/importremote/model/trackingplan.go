package model

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
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
		return nil, fmt.Errorf("loading tracking plan from upstream: %w", err)
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
			state.EventResourceType,
			event.ID,
		)
		if err != nil {
			return fmt.Errorf("event reference resolution for tracking plan %s: %w", tp.TrackingPlan.LocalID, err)
		}

		if eventRef == "" {
			return fmt.Errorf("resolved event reference is empty for tracking plan %s", tp.TrackingPlan.LocalID)
		}

		ruleProperties, err := buildRuleProperties(event.Properties, resolver)
		if err != nil {
			return fmt.Errorf("building properties for event %s in tracking plan %s: %w", event.ID, tp.TrackingPlan.LocalID, err)
		}

		// Since rules are not first class citizens in catalog,
		// we need to generate the localID for the rule based on the event
		// Product Viewed -> Product Viewed Rule -> product_viewed_rule
		ruleLocalID, err := idNamer.Name(namer.ScopeName{
			Name:  fmt.Sprintf("%s %s", event.Name, "Rule"),
			Scope: TypeEventRule,
		})
		if err != nil {
			return fmt.Errorf("generating localID for event rule %s: %w", event.Name, err)
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
			state.PropertyResourceType,
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

		ruleProps = append(ruleProps, &localcatalog.TPRuleProperty{
			Ref:        propRef,
			Required:   prop.Required,
			Properties: nestedProps,
		})
	}

	return ruleProps, nil
}
