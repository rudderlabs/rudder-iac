package validate

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

type DuplicateNameIDKeysValidator struct {
}

func (dv *DuplicateNameIDKeysValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	log.Info("validating duplicate name and id keys on the entities in catalog")

	var errors []ValidationError

	var (
		propName = make(map[string]*localcatalog.Property)
		propID   = make(map[string]*localcatalog.Property)
	)

	// Checking duplicate id and name keys in properties
	for group, props := range dc.Properties {
		for _, prop := range props {

			if lookup, ok := propName[prop.Name]; ok {
				// If name and type on the property are same, then it's a duplicate
				if lookup.Type == prop.Type {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("duplicate name key %s", prop.Name),
						Reference: fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID),
					})
				}
			}

			if _, ok := propID[prop.LocalID]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate id key %s", prop.LocalID),
					Reference: fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID),
				})
			}

			propName[prop.Name] = &prop
			propID[prop.LocalID] = &prop
		}
	}

	var (
		eventName = make(map[string]interface{})
		eventID   = make(map[string]interface{})
	)

	// Checking duplicate id and name keys in events
	for group, events := range dc.Events {
		for _, event := range events {

			if event.Type != "track" {
				continue
			}

			if _, ok := eventName[event.Name]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate name key %s", event.Name),
					Reference: fmt.Sprintf("#/events/%s/%s", group, event.LocalID),
				})
			}

			if _, ok := eventID[event.LocalID]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate id key %s", event.LocalID),
					Reference: fmt.Sprintf("#/events/%s/%s", group, event.LocalID),
				})
			}

			eventName[event.Name] = nil
			eventID[event.LocalID] = nil
		}
	}

	var (
		tpName   = make(map[string]interface{})
		tpID     = make(map[string]interface{})
		tpRuleID = make(map[string]interface{})
	)

	// Checking duplicate id and name keys of trackingplans
	for group, tp := range dc.TrackingPlans {
		if _, ok := tpName[tp.Name]; ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("duplicate name key %s", tp.Name),
				Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
			})
		}

		if _, ok := tpID[tp.LocalID]; ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("duplicate id key %s", tp.LocalID),
				Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
			})
		}

		tpName[tp.Name] = nil
		tpID[tp.LocalID] = nil

		for _, rule := range tp.Rules {
			if _, ok := tpRuleID[rule.LocalID]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate id key %s", rule.LocalID),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
			}

			tpRuleID[rule.LocalID] = nil
		}
	}

	return errors
}
