package entity

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/samber/lo"
)

var (
	validRuleTypes = []string{"includes", "event_rule"}
)

var (
	_ TypedCatalogEntityValidator[*localcatalog.TrackingPlan] = &TrackingPlanEntityValidator{}
	_ ValidationRule[*localcatalog.TrackingPlan]              = &TrackingPlanRequiredKeysRule{}
	_ ValidationRule[*localcatalog.TrackingPlan]              = &TrackingPlanRefRule{}
	_ ValidationRule[*localcatalog.TrackingPlan]              = &TrackingPlanDuplicateKeysRule{}
)

type TrackingPlanEntityValidator struct {
	rules []ValidationRule[*localcatalog.TrackingPlan]
}

func (tv *TrackingPlanEntityValidator) RegisterRule(rule ValidationRule[*localcatalog.TrackingPlan]) {
	tv.rules = append(tv.rules, rule)
}

func (tv *TrackingPlanEntityValidator) Validate(dc *localcatalog.DataCatalog) []ValidationError {

	var errors []ValidationError

	for group, tp := range dc.TrackingPlans {

		reference := fmt.Sprintf(
			"#/tp/%s/%s",
			group,
			tp.LocalID,
		)

		for _, rule := range tv.rules {
			errors = append(errors, rule.Validate(reference, tp, dc)...)
		}
	}

	return errors
}

type TrackingPlanRequiredKeysRule struct {
}

func (rule *TrackingPlanRequiredKeysRule) Validate(
	ref string,
	tp *localcatalog.TrackingPlan,
	dc *localcatalog.DataCatalog) (errs []ValidationError) {

	if tp.LocalID == "" {
		errs = append(errs, ValidationError{
			Err:        ErrMissingRequiredKeysID,
			Reference:  ref,
			EntityType: TrackingPlan,
		})
	}

	if tp.Name == "" {
		errs = append(errs, ValidationError{
			Err:        ErrMissingRequiredKeysName,
			Reference:  ref,
			EntityType: TrackingPlan,
		})
	}

	for _, rule := range tp.Rules {

		if rule.LocalID == "" {
			errs = append(errs, ValidationError{
				Err:        ErrMissingRequiredKeysRuleID,
				Reference:  ref,
				EntityType: TrackingPlan,
			})
		}

		if !lo.Contains(validRuleTypes, rule.Type) {
			errs = append(errs, ValidationError{
				Err:        fmt.Errorf("%w: %s", ErrInvalidTrackingPlanEventRuleType, rule.LocalID),
				Reference:  ref,
				EntityType: TrackingPlan,
			})
		}

		if rule.Event == nil {
			errs = append(errs, ValidationError{
				Err:        fmt.Errorf("%w: %s", ErrMissingRequiredKeysRuleEvent, rule.LocalID),
				Reference:  ref,
				EntityType: TrackingPlan,
			})
		}

	}
	return
}

// TrackingPlanRefRule validates the references to events / properties used in the trackingplan
type TrackingPlanRefRule struct {
}

func (rule *TrackingPlanRefRule) Validate(ref string, tp *localcatalog.TrackingPlan, dc *localcatalog.DataCatalog) (errs []ValidationError) {

	lo.ForEach(tp.Rules, func(r *localcatalog.TPRule, idx int) {
		if r.Event != nil {
			group, eventID, err := localcatalog.ExpandEventRef(r.Event.Ref)
			if err != nil {
				errs = append(errs, ValidationError{
					Err:        fmt.Errorf("%w: rule: %s event: %s", ErrInvalidRefFormat, r.LocalID, r.Event.Ref),
					Reference:  ref,
					EntityType: TrackingPlan,
				})
				return
			}

			event := rule.eventFromRef(group, eventID, dc)
			if event == nil {
				errs = append(errs, ValidationError{
					Err:        fmt.Errorf("%w: rule: %s event: %s", ErrMissingEntityFromRef, r.LocalID, r.Event.Ref),
					Reference:  ref,
					EntityType: TrackingPlan,
				})
				return
			}

			if lo.Contains(NonTrackEventTypes, event.Type) && r.Event.IdentitySection == "" {
				errs = append(errs, ValidationError{
					Err:        fmt.Errorf("%w: rule: %s event: %s", ErrMissingIdentityApplied, r.LocalID, r.Event.Ref),
					Reference:  ref,
					EntityType: TrackingPlan,
				})
			}

			if !lo.Contains(NonTrackEventTypes, event.Type) && r.Event.IdentitySection != "" {
				errs = append(errs, ValidationError{
					Err:        fmt.Errorf("%w: rule: %s event: %s", ErrInvalidIdentityApplied, r.LocalID, r.Event.Ref),
					Reference:  ref,
					EntityType: TrackingPlan,
				})
			}

		}

		for _, prop := range r.Properties {

			group, propID, err := localcatalog.ExpandPropertyRef(prop.Ref)
			if err != nil {
				errs = append(errs, ValidationError{
					Err:        fmt.Errorf("%w: rule: %s property: %s", ErrInvalidRefFormat, r.LocalID, prop.Ref),
					Reference:  ref,
					EntityType: TrackingPlan,
				})
				return
			}

			if property := rule.propertyFromRef(group, propID, dc); property == nil {
				errs = append(errs, ValidationError{
					Err:        fmt.Errorf("%w: rule: %s property: %s", ErrMissingEntityFromRef, r.LocalID, prop.Ref),
					Reference:  ref,
					EntityType: TrackingPlan,
				})

			}
		}
	})

	return errs
}

func (rule *TrackingPlanRefRule) eventFromRef(groupName, id string, dc *localcatalog.DataCatalog) *localcatalog.Event {

	var (
		events []*localcatalog.Event
		ok     bool
	)

	if events, ok = dc.Events[localcatalog.EntityGroup(groupName)]; !ok {
		return nil
	}

	for _, event := range events {
		if event.LocalID == id {
			return event
		}
	}

	return nil
}

func (rule *TrackingPlanRefRule) propertyFromRef(groupName, id string, dc *localcatalog.DataCatalog) *localcatalog.Property {

	var (
		props []*localcatalog.Property
		ok    bool
	)

	if props, ok = dc.Properties[localcatalog.EntityGroup(groupName)]; !ok {
		return nil
	}

	for _, prop := range props {
		if prop.LocalID == id {
			return prop
		}
	}

	return nil
}

type TrackingPlanDuplicateKeysRule struct {
}

func (rule *TrackingPlanDuplicateKeysRule) Validate(ref string, tp *localcatalog.TrackingPlan, dc *localcatalog.DataCatalog) (errs []ValidationError) {
	tps := rule.getByID(tp.LocalID, dc)
	if len(tps) > 1 {
		errs = append(errs, ValidationError{
			Err:        ErrDuplicateByID,
			Reference:  ref,
			EntityType: TrackingPlan,
		})
	}

	tps = rule.getByName(tp.Name, dc)
	if len(tps) > 1 {
		errs = append(errs, ValidationError{
			Err:        ErrDuplicateByName,
			Reference:  ref,
			EntityType: TrackingPlan,
		})
	}

	for eventRef, rules := range rule.eventRefRulesMap(tp) {
		if len(rules) <= 1 {
			continue
		}

		lo.ForEach(rules, func(r *localcatalog.TPRule, idx int) {
			errs = append(errs, ValidationError{
				Err:        fmt.Errorf("%w: rule: %s event: %s", ErrDuplicateEntityRefs, r.LocalID, eventRef),
				Reference:  ref,
				EntityType: TrackingPlan,
			})
		})
	}

	for _, rule := range tp.Rules {

		var (
			propRefsCount = make(map[string]int)
		)

		lo.ForEach(rule.Properties, func(prop *localcatalog.TPRuleProperty, idx int) {

			if _, ok := propRefsCount[prop.Ref]; !ok {
				propRefsCount[prop.Ref] = 0
			}

			propRefsCount[prop.Ref] = propRefsCount[prop.Ref] + 1
		})

		for propRef, count := range propRefsCount {

			if count <= 1 {
				continue
			}

			errs = append(errs, ValidationError{
				Err:        fmt.Errorf("%w: rule: %s property: %s", ErrDuplicateEntityRefs, rule.LocalID, propRef),
				Reference:  ref,
				EntityType: TrackingPlan,
			})
		}

	}

	return errs
}

func (rule *TrackingPlanDuplicateKeysRule) getByID(id string, dc *localcatalog.DataCatalog) []*localcatalog.TrackingPlan {
	var tps []*localcatalog.TrackingPlan
	for _, tp := range dc.TrackingPlans {
		if tp.LocalID == id {
			tps = append(tps, tp)
		}
	}
	return tps
}

func (rule *TrackingPlanDuplicateKeysRule) getByName(name string, dc *localcatalog.DataCatalog) []*localcatalog.TrackingPlan {
	var tps []*localcatalog.TrackingPlan
	for _, tp := range dc.TrackingPlans {
		if tp.Name == name {
			tps = append(tps, tp)
		}
	}
	return tps
}

func (rule *TrackingPlanDuplicateKeysRule) eventRefRulesMap(tp *localcatalog.TrackingPlan) map[string][]*localcatalog.TPRule {
	refRuleLookup := map[string][]*localcatalog.TPRule{}

	for _, rule := range tp.Rules {
		if rule.Event == nil {
			continue
		}

		if _, ok := refRuleLookup[rule.Event.Ref]; !ok {
			refRuleLookup[rule.Event.Ref] = []*localcatalog.TPRule{}
		}

		refRuleLookup[rule.Event.Ref] = append(refRuleLookup[rule.Event.Ref], rule)
	}

	return refRuleLookup
}
