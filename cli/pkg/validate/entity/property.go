package entity

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/samber/lo"
)

var (
	_ TypedCatalogEntityValidator[*localcatalog.Property] = &PropertyEntityValidator{}
	_ ValidationRule[*localcatalog.Property]              = &PropertyRequiredKeysRule{}
	_ ValidationRule[*localcatalog.Property]              = &PropertyDuplicateKeysRule{}
)

var validPropertyTypes = []string{"string", "number", "boolean", "object", "array", "integer"}

type PropertyEntityValidator struct {
	rules []ValidationRule[*localcatalog.Property]
}

func (v *PropertyEntityValidator) RegisterRule(rule ValidationRule[*localcatalog.Property]) {
	v.rules = append(v.rules, rule)
}

func (pv *PropertyEntityValidator) Validate(dc *localcatalog.DataCatalog) []ValidationError {

	var errors []ValidationError

	for group, properties := range dc.Properties {
		for _, property := range properties {

			reference := fmt.Sprintf(
				"#/properties/%s/%s",
				group,
				property.LocalID,
			)

			for _, rule := range pv.rules {
				errors = append(errors, rule.Validate(reference, property, dc)...)
			}
		}
	}

	return errors
}

type PropertyRequiredKeysRule struct {
}

func (rule *PropertyRequiredKeysRule) Validate(reference string, property *localcatalog.Property, dc *localcatalog.DataCatalog) []ValidationError {
	var errors []ValidationError

	if property.LocalID == "" {
		errors = append(errors, ValidationError{
			Err:        ErrMissingRequiredKeysID,
			Reference:  reference,
			EntityType: Property,
		})
	}

	if property.Name == "" {
		errors = append(errors, ValidationError{
			Err:        ErrMissingRequiredKeysName,
			Reference:  reference,
			EntityType: Property,
		})
	}

	propTypes := strings.Split(property.Type, ",")
	if len(propTypes) == 0 || !lo.Every(validPropertyTypes, propTypes) {
		errors = append(errors, ValidationError{
			Err:        ErrInvalidRequiredKeysPropertyType,
			Reference:  reference,
			EntityType: Property,
		})
	}

	return errors
}

type PropertyDuplicateKeysRule struct {
}

func (rule *PropertyDuplicateKeysRule) Validate(reference string, property *localcatalog.Property, dc *localcatalog.DataCatalog) []ValidationError {
	var errors []ValidationError

	properties := rule.propertiesById(property.LocalID, dc)
	if len(properties) > 1 {
		errors = append(errors, ValidationError{
			Err:        ErrDuplicateByID,
			Reference:  reference,
			EntityType: Property,
		})
	}

	properties = rule.propertiesByName(property.Name, dc)
	typeFound := false
	for _, prop := range properties {
		if prop.Type == property.Type {

			// mark the first time the same type is found.
			if !typeFound {
				typeFound = true
				continue
			}

			// If found again, then it's a duplicate
			errors = append(errors, ValidationError{
				Err:        ErrDuplicateByNameType,
				Reference:  reference,
				EntityType: Property,
			})
			break
		}
	}

	return errors
}

func (rule *PropertyDuplicateKeysRule) propertiesById(id string, dc *localcatalog.DataCatalog) []*localcatalog.Property {
	var properties []*localcatalog.Property
	for _, props := range dc.Properties {
		for _, prop := range props {
			if prop.LocalID == id {
				properties = append(properties, prop)
			}
		}
	}
	return properties
}

func (rule *PropertyDuplicateKeysRule) propertiesByName(name string, dc *localcatalog.DataCatalog) []*localcatalog.Property {
	var properties []*localcatalog.Property
	for _, props := range dc.Properties {
		for _, prop := range props {
			if prop.Name == name {
				properties = append(properties, prop)
			}
		}
	}
	return properties
}
