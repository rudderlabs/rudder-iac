package validate

import (
	"fmt"

	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

var invalidRefFormat = fmt.Errorf("invalid reference format")

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
				rv.handleRefs(rule, dc)...,
			)
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
				error:     invalidRefFormat,
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
					error:     invalidRefFormat,
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
				error:     invalidRefFormat,
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
