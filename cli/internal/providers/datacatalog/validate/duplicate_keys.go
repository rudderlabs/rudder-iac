package validate

import (
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
)

type DuplicateNameIDKeysValidator struct {
}

func (dv *DuplicateNameIDKeysValidator) Validate(dc *localcatalog.DataCatalog) []ValidationError {
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
				var lookupArrayItemType []any
				var propArrayItemType []any
				if lookup.Type == "array" && lookup.Config != nil {
					if itemTypes, ok := lookup.Config["itemTypes"]; ok {
						if arr, ok := itemTypes.([]any); ok && len(arr) > 0 {
							utils.SortLexicographically(arr)
							lookupArrayItemType = arr
						}
					}
				}
				if prop.Type == "array" && prop.Config != nil {
					if itemTypes, ok := prop.Config["itemTypes"]; ok {
						if arr, ok := itemTypes.([]any); ok && len(arr) > 0 {
							utils.SortLexicographically(arr)
							propArrayItemType = arr
						}
					}
				}

				// If name, type and arrayItemType on the property are the same, then it's a duplicate
				if lookup.Type == prop.Type {
					switch {
					// the property is a duplicate if - 
					// 1. lookupArrayItemType and propArrayItemType are both nil
					// 2. lookupArrayItemType and propArrayItemType are not nil and identical
					case lookupArrayItemType == nil && propArrayItemType == nil:
						errors = append(errors, ValidationError{
							error:     fmt.Errorf("duplicate name key: %s, type: %s", prop.Name, prop.Type),
							Reference: fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID),
						})
					case lookupArrayItemType != nil && propArrayItemType != nil && slices.Equal(lookupArrayItemType, propArrayItemType):
						errors = append(errors, ValidationError{
							error:     fmt.Errorf("duplicate name key: %s, type: %s and arrayItemType: %v", prop.Name, prop.Type, propArrayItemType),
							Reference: fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID),
						})
					}
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
		eventName = make(map[string]any)
		eventID   = make(map[string]any)
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
		tpName = make(map[string]any)
		tpID   = make(map[string]any)
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

		eventRuleIDs := make(map[string]any)
		for _, rule := range tp.Rules {
			if _, ok := eventRuleIDs[rule.LocalID]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate id key %s", rule.LocalID),
					Reference: fmt.Sprintf("#/tp/%s/%s/rules/%s", group, tp.LocalID, rule.LocalID),
				})
			}
			eventRuleIDs[rule.LocalID] = nil
		}

		tpName[tp.Name] = nil
		tpID[tp.LocalID] = nil

	}

	var (
		customTypeName = make(map[string]any)
		customTypeID   = make(map[string]any)
	)

	// Checking duplicate id and name keys in custom types
	for group, customTypes := range dc.CustomTypes {
		for _, customType := range customTypes {
			if _, ok := customTypeName[customType.Name]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate name key %s in custom types", customType.Name),
					Reference: fmt.Sprintf("#/custom-types/%s/%s", group, customType.LocalID),
				})
			}

			if _, ok := customTypeID[customType.LocalID]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate id key %s in custom types", customType.LocalID),
					Reference: fmt.Sprintf("#/custom-types/%s/%s", group, customType.LocalID),
				})
			}

			customTypeName[customType.Name] = nil
			customTypeID[customType.LocalID] = nil
		}
	}

	var (
		categoryName = make(map[string]any)
		categoryID   = make(map[string]any)
	)

	// Checking duplicate id and name keys in categories
	for group, categories := range dc.Categories {
		for _, category := range categories {
			if _, ok := categoryName[category.Name]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate name key %s in categories", category.Name),
					Reference: fmt.Sprintf("#/categories/%s/%s", group, category.LocalID),
				})
			}

			if _, ok := categoryID[category.LocalID]; ok {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate id key %s in categories", category.LocalID),
					Reference: fmt.Sprintf("#/categories/%s/%s", group, category.LocalID),
				})
			}

			categoryName[category.Name] = nil
			categoryID[category.LocalID] = nil
		}
	}

	return errors
}
