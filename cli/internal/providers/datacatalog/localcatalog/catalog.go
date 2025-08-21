package localcatalog

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

var (
	log = logger.New("localcatalog")
)

// entity group is logical grouping of entities defined
// as metadata->name in respective yaml file
type EntityGroup string

// Create a reverse lookup based on the groupName and identifier per entity
type DataCatalog struct {
	Properties    map[EntityGroup][]Property    `json:"properties"`
	Events        map[EntityGroup][]Event       `json:"events"`
	TrackingPlans map[EntityGroup]*TrackingPlan `json:"trackingPlans"` // Only one tracking plan per entity group
	CustomTypes   map[EntityGroup][]CustomType  `json:"customTypes"`   // Custom types grouped by entity group
	Categories    map[EntityGroup][]Category    `json:"categories"`    // Categories grouped by entity group
}

func (dc *DataCatalog) Property(groupName string, id string) *Property {
	if props, ok := dc.Properties[EntityGroup(groupName)]; ok {
		for _, prop := range props {
			if prop.LocalID == id {
				return &prop
			}
		}
	}
	return nil
}

func (dc *DataCatalog) Event(groupName string, id string) *Event {
	if events, ok := dc.Events[EntityGroup(groupName)]; ok {
		for _, event := range events {
			if event.LocalID == id {
				return &event
			}
		}
	}
	return nil
}

// Category returns a category by group name and ID
func (dc *DataCatalog) Category(groupName string, id string) *Category {
	if categories, ok := dc.Categories[EntityGroup(groupName)]; ok {
		for _, category := range categories {
			if category.LocalID == id {
				return &category
			}
		}
	}
	return nil
}

// CustomType returns a custom type by group name and ID
func (dc *DataCatalog) CustomType(groupName string, id string) *CustomType {
	if types, ok := dc.CustomTypes[EntityGroup(groupName)]; ok {
		for _, customType := range types {
			if customType.LocalID == id {
				return &customType
			}
		}
	}
	return nil
}

func (dc *DataCatalog) TPEventRule(tpGroup, ruleID string) *TPRule {
	tp, ok := dc.TrackingPlans[EntityGroup(tpGroup)]
	if !ok {
		return nil
	}

	for _, rule := range tp.Rules {
		if rule.LocalID == ruleID && rule.Type == "event_rule" {
			return rule
		}
	}

	return nil
}

func (dc *DataCatalog) TPEventRules(tpGroup string) ([]*TPRule, bool) {
	tp, ok := dc.TrackingPlans[EntityGroup(tpGroup)]
	if !ok {
		return nil, false
	}

	var toReturn []*TPRule
	for _, rule := range tp.Rules {
		if rule.Type != "event_rule" {
			continue
		}
		toReturn = append(toReturn, rule)
	}

	return toReturn, true
}

func New() *DataCatalog {
	return &DataCatalog{
		Properties:    map[EntityGroup][]Property{},
		Events:        map[EntityGroup][]Event{},
		TrackingPlans: map[EntityGroup]*TrackingPlan{},
		CustomTypes:   map[EntityGroup][]CustomType{},
		Categories:    map[EntityGroup][]Category{},
	}
}

func (dc *DataCatalog) LoadSpec(path string, s *specs.Spec) error {
	if err := extractEntities(s, dc); err != nil {
		return fmt.Errorf("extracting data catalog entity from file: %s : %w", path, err)
	}

	return nil
}

// extractEntities parses the entity from file bytes
// and updates the datacatalog struct with it.
func extractEntities(s *specs.Spec, dc *DataCatalog) error {
	// TODO: properly handle metadata - ensuring schema and types
	name, ok := s.Metadata["name"].(string)
	if !ok {
		name = ""
	}
	switch s.Kind {
	case "properties":
		properties, err := ExtractProperties(s)
		if err != nil {
			return fmt.Errorf("extracting properties: %w", err)
		}
		dc.Properties[EntityGroup(name)] = append(dc.Properties[EntityGroup(name)], properties...)

	case "events":
		events, err := ExtractEvents(s)
		if err != nil {
			return fmt.Errorf("extracting property entity: %w", err)
		}
		dc.Events[EntityGroup(name)] = append(dc.Events[EntityGroup(name)], events...)

	case "categories":
		categories, err := ExtractCategories(s)
		if err != nil {
			return fmt.Errorf("extracting categories: %w", err)
		}
		dc.Categories[EntityGroup(name)] = append(dc.Categories[EntityGroup(name)], categories...)

	case "tp":
		tp, err := ExtractTrackingPlan(s)
		if err != nil {
			return fmt.Errorf("extracting tracking plan: %w", err)
		}

		if _, exists := dc.TrackingPlans[EntityGroup(name)]; exists {
			return fmt.Errorf("duplicate tracking plan with metadata.name '%s' found - only one tracking plan per entity group is allowed", name)
		}
		dc.TrackingPlans[EntityGroup(name)] = &tp

	case "custom-types":
		customTypes, err := ExtractCustomTypes(s)
		if err != nil {
			return fmt.Errorf("extracting custom types: %w", err)
		}
		dc.CustomTypes[EntityGroup(name)] = append(dc.CustomTypes[EntityGroup(name)], customTypes...)

	default:
		return fmt.Errorf("unknown kind: %s", s.Kind)
	}

	return nil
}
