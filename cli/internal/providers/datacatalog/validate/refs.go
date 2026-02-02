package validate

import (
	"errors"
	"fmt"
	"strings"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/samber/lo"
)

var errInvalidRefFormat = fmt.Errorf("invalid reference format")

// RefValidator checks the references in tracking plan to other
// events and properties in data catalog and verifies if the refs are valid
type RefValidator struct {
}

func (rv *RefValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	log.Info("validating references lookup in entities in the catalog")

	errs := make([]ValidationError, 0)

	for _, tp := range dc.TrackingPlans {
		for _, rule := range tp.Rules {
			errs = append(
				errs,
				rv.handleRefs(rule, fmt.Sprintf("#tp:%s", tp.LocalID), dc)...,
			)
		}
	}

	// Validate custom type references
	for _, customType := range dc.CustomTypes {
		if customType.Type == "object" {
			reference := fmt.Sprintf("#%s:%s", catalog.KindCustomTypes, customType.LocalID)

			// Step 1: Validate references in custom type properties
			for i, prop := range customType.Properties {
				matches := catalog.PropRegex.FindStringSubmatch(prop.Property)
				if len(matches) != 2 {
					errs = append(errs, ValidationError{
						Reference: reference,
						error:     fmt.Errorf("property reference at index %d has invalid format. Should be '#property:<id>'", i),
					})
					continue
				}

				propID := matches[1]
				if property := dc.Property(propID); property == nil {
					errs = append(errs, ValidationError{
						Reference: reference,
						error:     fmt.Errorf("property reference '%s' at index %d not found in catalog", prop.Property, i),
					})
				}
			}

			// Step 2: Validate references in custom type variants
			errs = append(
				errs,
				rv.validateVariantsReferencesV1(
					customType.Variants,
					reference,
					dc,
					func(id string) bool {
						return lo.ContainsBy(customType.Properties, func(p catalog.CustomTypePropertyV1) bool {
							// customType.Properties have been validated above, so we can safely assume the ref is valid
							matches := catalog.PropRegex.FindStringSubmatch(p.Property)
							return matches[1] == id
						})
					})...)
		}
	}

	// Validate property type custom type references
	for _, prop := range dc.Properties {
		reference := fmt.Sprintf("#%s:%s", catalog.KindProperties, prop.LocalID)

		// Check if the type field contains a custom type reference
		if strings.HasPrefix(prop.Type, "#custom-type:") {
			matches := catalog.CustomTypeRegex.FindStringSubmatch(prop.Type)
			if len(matches) != 2 {
				errs = append(errs, ValidationError{
					Reference: reference,
					error:     fmt.Errorf("custom type reference in type field has invalid format. Should be '#custom-type:<id>'"),
				})
				continue
			}

			// Validate custom type existence
			customTypeID := matches[1]
			if customType := dc.CustomType(customTypeID); customType == nil {
				errs = append(errs, ValidationError{
					Reference: reference,
					error:     fmt.Errorf("custom type reference '%s' not found in catalog", prop.Type),
				})
			}
		}

		// Validate types array - should not contain custom type references
		if len(prop.Types) > 0 {
			for idx, typeVal := range prop.Types {
				if strings.HasPrefix(typeVal, "#custom-type:") {
					errs = append(errs, ValidationError{
						Reference: reference,
						error:     fmt.Errorf("types[%d]: custom type references are not allowed in types array, use single 'type' field for custom types", idx),
					})
				}
			}
		}

		// Check for custom type reference in item_type field (single)
		if prop.ItemType != "" && strings.HasPrefix(prop.ItemType, "#custom-type:") {
			matches := catalog.CustomTypeRegex.FindStringSubmatch(prop.ItemType)
			if len(matches) != 2 {
				errs = append(errs, ValidationError{
					Reference: reference,
					error:     fmt.Errorf("custom type reference in item_type has invalid format. Should be '#custom-type:<id>'"),
				})
			} else {
				// Validate custom type existence
				customTypeID := matches[1]
				if customType := dc.CustomType(customTypeID); customType == nil {
					errs = append(errs, ValidationError{
						Reference: reference,
						error:     fmt.Errorf("custom type reference '%s' in item_type not found in catalog", prop.ItemType),
					})
				}
			}
		}

		// Check for custom type references in item_types field (multiple)
		if len(prop.ItemTypes) > 0 {
			for idx, itemType := range prop.ItemTypes {
				// Check if the item type is a custom type reference
				if strings.HasPrefix(itemType, "#custom-type:") {
					matches := catalog.CustomTypeRegex.FindStringSubmatch(itemType)
					if len(matches) != 2 {
						errs = append(errs, ValidationError{
							Reference: reference,
							error:     fmt.Errorf("custom type reference in item_types at idx: %d has invalid format. Should be '#custom-type:<id>'", idx),
						})
						continue
					}

					// Validate custom type existence
					customTypeID := matches[1]
					if customType := dc.CustomType(customTypeID); customType == nil {
						errs = append(errs, ValidationError{
							Reference: reference,
							error:     fmt.Errorf("custom type reference '%s' in item_types at idx: %d not found in catalog", itemType, idx),
						})
					}
				}
			}
		}
	}

	// Validate event category references
	for _, event := range dc.Events {
		reference := fmt.Sprintf("#%s:%s", catalog.KindEvents, event.LocalID)

		// Check if the event has a category reference
		if event.CategoryRef != nil {
			// Validate category reference format
			matches := catalog.CategoryRegex.FindStringSubmatch(*event.CategoryRef)
			if len(matches) != 2 {
				errs = append(errs, ValidationError{
					Reference: reference,
					error:     fmt.Errorf("the category field value is invalid. It should always be a reference and must follow the format '#category:<id>'"),
				})
				continue
			}

			// Validate category existence
			categoryID := matches[1]
			if category := dc.Category(categoryID); category == nil {
				errs = append(errs, ValidationError{
					Reference: reference,
					error:     fmt.Errorf("category reference '%s' not found in catalog", *event.CategoryRef),
				})
			}
		}
	}

	return errs
}

func (rv *RefValidator) handleRefs(rule *catalog.TPRuleV1, baseReference string, fetcher catalog.CatalogResourceFetcher) []ValidationError {
	errs := make([]ValidationError, 0)

	if rule.Event != "" {
		matches := catalog.EventRegex.FindStringSubmatch(rule.Event)
		if len(matches) != 2 {
			errs = append(errs, ValidationError{
				Reference: rule.Event,
				error:     errInvalidRefFormat,
			})
		} else {
			if event := fetcher.Event(matches[1]); event == nil {
				errs = append(errs, ValidationError{
					Reference: rule.Event,
					error:     fmt.Errorf("no event found from reference"),
				})
			}
		}
	}
	if rule.Properties != nil {
		for _, prop := range rule.Properties {
			matches := catalog.PropRegex.FindStringSubmatch(prop.Ref)
			if len(matches) != 2 {
				errs = append(errs, ValidationError{
					Reference: prop.Ref,
					error:     errInvalidRefFormat,
				})
			} else {
				if property := fetcher.Property(matches[1]); property == nil {
					errs = append(errs, ValidationError{
						Reference: prop.Ref,
						error:     fmt.Errorf("no property found from reference"),
					})
				}
			}

			if len(prop.Properties) > 0 {
				errs = append(errs, rv.validateNestedPropertyRefs(prop, fmt.Sprintf("%s/rules/%s", baseReference, rule.LocalID), fetcher)...)
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

	if rule.Variants != nil {
		errs = append(
			errs,
			rv.validateVariantsReferences(
				rule.Variants,
				fmt.Sprintf("%s/event_rule/%s", baseReference, rule.LocalID),
				fetcher,
				func(id string) bool {
					return lo.ContainsBy(rule.Properties, func(p *catalog.TPRuleProperty) bool {
						// rule.Properties have been validated above, so we can safely assume the ref is valid
						matches := catalog.PropRegex.FindStringSubmatch(p.Ref)
						return matches[1] == id
					})
				},
			)...,
		)
	}

	return errs
}

func (rv *RefValidator) validateNestedPropertyRefs(prop *catalog.TPRuleProperty, ruleRef string, dc catalog.CatalogResourceFetcher) []ValidationError {
	errs := make([]ValidationError, 0)

	for _, nestedProp := range prop.Properties {
		matches := catalog.PropRegex.FindStringSubmatch(nestedProp.Ref)
		if len(matches) != 2 {
			errs = append(errs, ValidationError{
				Reference: nestedProp.Ref,
				error:     fmt.Errorf("property reference '%s' has invalid format in rule '%s'. Should be '#property:<id>'", nestedProp.Ref, ruleRef),
			})
			continue
		}

		propID := matches[1]
		if property := dc.Property(propID); property == nil {
			errs = append(errs, ValidationError{
				Reference: nestedProp.Ref,
				error:     fmt.Errorf("property reference '%s' in rule '%s' not found in catalog", nestedProp.Ref, ruleRef),
			})
		}

		if len(nestedProp.Properties) > 0 {
			errs = append(errs, rv.validateNestedPropertyRefs(nestedProp, ruleRef, dc)...)
		}
	}

	return errs
}

func (rv *RefValidator) validateVariantsReferences(
	variants catalog.Variants,
	reference string,
	fetcher catalog.CatalogResourceFetcher,
	rulePropertyExists func(id string) bool,
) []ValidationError {
	errs := make([]ValidationError, 0)

	for idx, variant := range variants {
		variantReference := fmt.Sprintf("%s/variants[%d]", reference, idx)

		matches := catalog.PropRegex.FindStringSubmatch(variant.Discriminator)
		if len(matches) != 2 {
			errs = append(errs, ValidationError{
				Reference: variantReference,
				error:     errors.New("discriminator reference has invalid format, should be #property:<id>"),
			})
		} else {
			propID := matches[1]
			if prop := fetcher.Property(propID); prop != nil {
				if !strings.HasPrefix(prop.Type, "#custom-type:") {
					if !strings.Contains(prop.Type, "string") && !strings.Contains(prop.Type, "integer") && !strings.Contains(prop.Type, "boolean") {
						errs = append(errs, ValidationError{
							Reference: variantReference,
							error:     fmt.Errorf("discriminator reference '%s' has invalid type, should be one of 'string', 'integer', 'boolean'", variant.Discriminator),
						})
					}
				}
			}

			if !rulePropertyExists(propID) {
				errs = append(errs, ValidationError{
					Reference: variantReference,
					error:     fmt.Errorf("discriminator reference '%s' not found in rule properties", variant.Discriminator),
				})
			}
		}

		for idx, variantCase := range variant.Cases {
			caseReference := fmt.Sprintf("%s/cases[%d]", variantReference, idx)

			for idx, propRef := range variantCase.Properties {
				propReference := fmt.Sprintf("%s/properties[%d]", caseReference, idx)

				matches := catalog.PropRegex.FindStringSubmatch(propRef.Ref)
				if len(matches) != 2 {
					errs = append(errs, ValidationError{
						Reference: propReference,
						error:     fmt.Errorf("property reference has invalid format. Should be '#property:<id>'"),
					})
					continue
				}

				propID := matches[1]
				if property := fetcher.Property(propID); property == nil {
					errs = append(errs, ValidationError{
						Reference: propReference,
						error:     fmt.Errorf("property reference '%s' not found in catalog", propRef.Ref),
					})
				}
			}
		}

		for idx, propRef := range variant.Default {
			defaultReference := fmt.Sprintf("%s/default/properties[%d]", variantReference, idx)

			matches := catalog.PropRegex.FindStringSubmatch(propRef.Ref)
			if len(matches) != 2 {
				errs = append(errs, ValidationError{
					Reference: defaultReference,
					error:     fmt.Errorf("default property reference has invalid format. Should be '#property:<id>'"),
				})
				continue
			}

			propID := matches[1]
			if property := fetcher.Property(propID); property == nil {
				errs = append(errs, ValidationError{
					Reference: defaultReference,
					error:     fmt.Errorf("default property reference '%s' not found in catalog", propRef.Ref),
				})
			}
		}
	}

	return errs
}

// validateVariantsReferencesV1 validates V1 variant references (for custom types)
func (rv *RefValidator) validateVariantsReferencesV1(
	variants catalog.VariantsV1,
	reference string,
	fetcher catalog.CatalogResourceFetcher,
	rulePropertyExists func(id string) bool,
) []ValidationError {
	errs := make([]ValidationError, 0)

	for idx, variant := range variants {
		variantReference := fmt.Sprintf("%s/variants[%d]", reference, idx)

		matches := catalog.PropRegex.FindStringSubmatch(variant.Discriminator)
		if len(matches) != 2 {
			errs = append(errs, ValidationError{
				Reference: variantReference,
				error:     errors.New("discriminator reference has invalid format, should be #property:<id>"),
			})
		} else {
			propID := matches[1]
			if prop := fetcher.Property(propID); prop != nil {
				if !strings.HasPrefix(prop.Type, "#custom-type:") {
					if !strings.Contains(prop.Type, "string") && !strings.Contains(prop.Type, "integer") && !strings.Contains(prop.Type, "boolean") {
						errs = append(errs, ValidationError{
							Reference: variantReference,
							error:     fmt.Errorf("discriminator reference '%s' has invalid type, should be one of 'string', 'integer', 'boolean'", variant.Discriminator),
						})
					}
				}
			}

			if !rulePropertyExists(propID) {
				errs = append(errs, ValidationError{
					Reference: variantReference,
					error:     fmt.Errorf("discriminator reference '%s' not found in rule properties", variant.Discriminator),
				})
			}
		}

		for idx, variantCase := range variant.Cases {
			caseReference := fmt.Sprintf("%s/cases[%d]", variantReference, idx)

			for idx, propRef := range variantCase.Properties {
				propReference := fmt.Sprintf("%s/properties[%d]", caseReference, idx)

				matches := catalog.PropRegex.FindStringSubmatch(propRef.Property)
				if len(matches) != 2 {
					errs = append(errs, ValidationError{
						Reference: propReference,
						error:     fmt.Errorf("property reference has invalid format. Should be '#property:<id>'"),
					})
					continue
				}

				propID := matches[1]
				if property := fetcher.Property(propID); property == nil {
					errs = append(errs, ValidationError{
						Reference: propReference,
						error:     fmt.Errorf("property reference '%s' not found in catalog", propRef.Property),
					})
				}
			}
		}

		for idx, propRef := range variant.Default.Properties {
			defaultReference := fmt.Sprintf("%s/default/properties[%d]", variantReference, idx)

			matches := catalog.PropRegex.FindStringSubmatch(propRef.Property)
			if len(matches) != 2 {
				errs = append(errs, ValidationError{
					Reference: defaultReference,
					error:     fmt.Errorf("default property reference has invalid format. Should be '#property:<id>'"),
				})
				continue
			}

			propID := matches[1]
			if property := fetcher.Property(propID); property == nil {
				errs = append(errs, ValidationError{
					Reference: defaultReference,
					error:     fmt.Errorf("default property reference '%s' not found in catalog", propRef.Property),
				})
			}
		}
	}

	return errs
}
