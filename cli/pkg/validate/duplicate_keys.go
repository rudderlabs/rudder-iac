package validate

import (
	"fmt"

	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

type DuplicateNameIDKeysValidator struct {
}

func (dv *DuplicateNameIDKeysValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	var errors []ValidationError

	var (
		propName = make(map[string]interface{})
		propID   = make(map[string]interface{})
	)

	// Checking duplicate id and name keys in properties
	for group, props := range dc.Properties {
		for _, prop := range props {

			if _, ok := propName[prop.Name]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate name key %s", prop.Name),
					Reference: fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID),
				})
			}

			if _, ok := propID[prop.LocalID]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate id key %s", prop.LocalID),
					Reference: fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID),
				})
			}

			propName[prop.Name] = nil
			propID[prop.LocalID] = nil
		}
	}

	var (
		eventName = make(map[string]interface{})
		eventID   = make(map[string]interface{})
	)

	// Checking duplicate id and name keys in events
	for group, events := range dc.Events {
		for _, event := range events {

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
		tpName = make(map[string]interface{})
		tpID   = make(map[string]interface{})
	)

	// Checking duplicate id and name keys of trackingplans
	for group, tp := range dc.TrackingPlans {
		if _, ok := tpName[tp.Name]; ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("duplicate name key %s", tp.Name),
				Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
			})
		}

		if _, ok := eventID[tp.LocalID]; ok {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("duplicate id key %s", tp.LocalID),
				Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
			})
		}

		tpName[tp.Name] = nil
		tpID[tp.LocalID] = nil
	}

	return errors
}
