package validate

import (
	"fmt"
	"strings"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

var errInvalidRefFormat = fmt.Errorf("invalid reference format")

// RefValidator checks the references in tracking plan to other
// events and properties in data catalog and verifies if the refs are valid
type RefValidator struct {
}

func (rv *RefValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	log.Info("validating references lookup in entities in the catalog")

	errs := make([]ValidationError, 0)

	// Validate tracking plan references
	for _, tp := range dc.TrackingPlans {
		for _, rule := range tp.Rules {
			errs = append(
				errs,
				rv.handleRefs(rule, dc)...,
			)
		}
	}

	// Validate custom type references
	for group, customTypes := range dc.CustomTypes {
		for _, customType := range customTypes {
			// Only object types with properties need validation
			if customType.Type == "object" && len(customType.Properties) > 0 {
				reference := fmt.Sprintf("#/custom-types/%s/%s", group, customType.LocalID)

				for i, prop := range customType.Properties {

					// Validate property reference format
					matches := catalog.PropRegex.FindStringSubmatch(prop.Ref)
					if len(matches) != 3 {
						errs = append(errs, ValidationError{
							Reference: reference,
							error:     fmt.Errorf("property reference at index %d has invalid format. Should be '#/properties/<group>/<id>'", i),
						})
						continue
					}

					// Validate property existence
					groupName, propID := matches[1], matches[2]
					if property := dc.Property(groupName, propID); property == nil {
						errs = append(errs, ValidationError{
							Reference: reference,
							error:     fmt.Errorf("property reference '%s' at index %d not found in catalog", prop.Ref, i),
						})
					}
				}
			}
		}
	}

	// Validate property type custom type references
	for group, props := range dc.Properties {
		for _, prop := range props {
			reference := fmt.Sprintf("#/properties/%s/%s", group, prop.LocalID)

			// Check if the type field contains a custom type reference
			if strings.HasPrefix(prop.Type, "#/custom-types/") {
				matches := catalog.CustomTypeRegex.FindStringSubmatch(prop.Type)
				if len(matches) != 3 {
					errs = append(errs, ValidationError{
						Reference: reference,
						error:     fmt.Errorf("custom type reference in type field has invalid format. Should be '#/custom-types/<group>/<id>'"),
					})
					continue
				}

				// Validate custom type existence
				customTypeGroup, customTypeID := matches[1], matches[2]
				if customType := dc.CustomType(customTypeGroup, customTypeID); customType == nil {
					errs = append(errs, ValidationError{
						Reference: reference,
						error:     fmt.Errorf("custom type reference '%s' not found in catalog", prop.Type),
					})
				}
			}

			// Check for custom type references in itemTypes when property is of type array
			if prop.Type == "array" && prop.Config != nil {
				if itemTypes, ok := prop.Config["itemTypes"]; ok {
					if itemTypesArray, ok := itemTypes.([]any); ok {
						for idx, itemType := range itemTypesArray {
							itemTypeStr, ok := itemType.(string)
							if !ok {
								continue
							}

							// Check if the item type is a custom type reference
							if strings.HasPrefix(itemTypeStr, "#/custom-types/") {
								matches := catalog.CustomTypeRegex.FindStringSubmatch(itemTypeStr)
								if len(matches) != 3 {
									errs = append(errs, ValidationError{
										Reference: reference,
										error:     fmt.Errorf("custom type reference in itemTypes at idx: %d has invalid format. Should be '#/custom-types/<group>/<id>'", idx),
									})
									continue
								}

								// Validate custom type existence
								customTypeGroup, customTypeID := matches[1], matches[2]
								if customType := dc.CustomType(customTypeGroup, customTypeID); customType == nil {
									errs = append(errs, ValidationError{
										Reference: reference,
										error:     fmt.Errorf("custom type reference '%s' in itemTypes at idx: %d not found in catalog", itemTypeStr, idx),
									})
								}
							}
						}
					}
				}
			}
		}
	}

	return errs
}

func (rv *RefValidator) handleRefs(rule *catalog.TPRule, fetcher catalog.CatalogResourceFetcher) []ValidationError {
	errs := make([]ValidationError, 0)

	if rule.Event != nil {
		matches := catalog.EventRegex.FindStringSubmatch(rule.Event.Ref)
		if len(matches) != 3 {
			errs = append(errs, ValidationError{
				Reference: rule.Event.Ref,
				error:     errInvalidRefFormat,
			})
		} else {
			if event := fetcher.Event(matches[1], matches[2]); event == nil {
				errs = append(errs, ValidationError{
					Reference: rule.Event.Ref,
					error:     fmt.Errorf("no event found from reference"),
				})
			}
		}
	}
	if rule.Properties != nil {
		for _, prop := range rule.Properties {
			matches := catalog.PropRegex.FindStringSubmatch(prop.Ref)
			if len(matches) != 3 {
				errs = append(errs, ValidationError{
					Reference: prop.Ref,
					error:     errInvalidRefFormat,
				})
			} else {
				if property := fetcher.Property(matches[1], matches[2]); property == nil {
					errs = append(errs, ValidationError{
						Reference: prop.Ref,
						error:     fmt.Errorf("no property found from reference"),
					})
				}
			}
		}
	}
	if rule.Includes != nil {
		matches := catalog.IncludeRegex.FindStringSubmatch(rule.Includes.Ref)
		if len(matches) != 3 {
			errs = append(errs, ValidationError{
				Reference: rule.Includes.Ref,
				error:     errInvalidRefFormat,
			})
		} else {
			group, id := matches[1], matches[2]
			if id == "*" {
				_, ok := fetcher.TPEventRules(group)
				if !ok {
					errs = append(errs, ValidationError{
						Reference: rule.Includes.Ref,
						error:     fmt.Errorf("no event rules found from reference"),
					})
				}
			} else {
				eventRule := fetcher.TPEventRule(group, id)
				if eventRule == nil {
					errs = append(errs, ValidationError{
						Reference: rule.Includes.Ref,
						error:     fmt.Errorf("no event rule found from reference"),
					})
				}
			}
		}
	}

	return errs
}
