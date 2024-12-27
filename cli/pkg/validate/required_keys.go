package validate

import (
	"fmt"

	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

type RequiredKeysValidator struct {
}

func (rk *RequiredKeysValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	log.Info("validating required keys on the entities in catalog")

	var errors []ValidationError

	// Properties required keys
	for group, props := range dc.Properties {
		for _, prop := range props {
			reference := fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID)

			if prop.Name == "" || prop.Type == "" || prop.LocalID == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id, name and type fields on property are mandatory"),
					Reference: reference,
				})
			}
		}
	}

	// Events required keys
	for group, events := range dc.Events {
		for _, event := range events {
			reference := fmt.Sprintf("#/events/%s/%s", group, event.LocalID)

			if event.LocalID == "" || event.Type == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id and event_type fields on event are mandatory"),
					Reference: reference,
				})
			}

			if event.Type == "track" {
				if event.Name == "" {
					errors = append(errors, ValidationError{
						error:     fmt.Errorf("display_name field is mandatory on track event"),
						Reference: reference,
					})
				}
			}
		}
	}

	// Tracking Plan required keys
	for group, tp := range dc.TrackingPlans {

		if tp.Name == "" {
			errors = append(errors, ValidationError{
				error:     fmt.Errorf("name field is mandatory on the tracking plan"),
				Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
			})
		}

		for _, rule := range tp.Rules {
			if rule.LocalID == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id field is mandatory on the rules in tracking plan"),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
			}

			if rule.Event == nil && rule.Includes == nil {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("event or includes section within the rules: %s in tracking plan are mandatory", rule.LocalID),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
			}

			if rule.Event != nil && rule.Includes != nil {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("event and includes both section within the rules: %s in tracking plan are not allowed", rule.LocalID),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
			}

			if rule.Event == nil && len(rule.Properties) > 0 {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("properties without events in rule: %s are not allowed", rule.LocalID),
					Reference: fmt.Sprintf("#/tp/%s/%s", group, tp.LocalID),
				})
			}

		}
	}

	return errors
}
