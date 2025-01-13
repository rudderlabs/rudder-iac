package entity

import (
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/samber/lo"
)

var (
	validRuleTypes = []string{"includes", "event_rule"}
)

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
				Err:        ErrInvalidTrackingPlanEventRuleType,
				Reference:  ref,
				EntityType: TrackingPlan,
			})
		}

		if rule.Event == nil {
			errs = append(errs, ValidationError{
				Err:        ErrMissingRequiredKeysRuleEvent,
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
					Err:        ErrInvalidRefFormat,
					Reference:  r.Event.Ref,
					EntityType: TrackingPlan,
				})
				return
			}

			event := rule.eventFromRef(group, eventID, dc)
			if event == nil {
				errs = append(errs, ValidationError{
					Err:        ErrMissingEntityFromRef,
					Reference:  r.Event.Ref,
					EntityType: TrackingPlan,
				})
			}

		}

		for _, prop := range r.Properties {

			group, propID, err := localcatalog.ExpandPropertyRef(prop.Ref)
			if err != nil {
				errs = append(errs, ValidationError{
					Err:        ErrInvalidRefFormat,
					Reference:  prop.Ref,
					EntityType: TrackingPlan,
				})
				return
			}

			if property := rule.propertyFromRef(group, propID, dc); property == nil {
				errs = append(errs, ValidationError{
					Err:        ErrMissingEntityFromRef,
					Reference:  prop.Ref,
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
