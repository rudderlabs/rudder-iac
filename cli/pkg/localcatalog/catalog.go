package localcatalog

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
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

// New creates a DataCatalog from a map of specs. It extracts all entities
// (properties, events, tracking plans) from the provided specs and organizes
// them in the catalog structure.
func New(specs map[string]*specs.Spec) (*DataCatalog, error) {
	dc := &DataCatalog{
		Properties:    map[EntityGroup][]Property{},
		Events:        map[EntityGroup][]Event{},
		TrackingPlans: map[EntityGroup]*TrackingPlan{},
		CustomTypes:   map[EntityGroup][]CustomType{},
	}

	for path, spec := range specs {
		if err := extractEntities(spec, dc); err != nil {
			return nil, fmt.Errorf("extracting data catalog entity from file: %s : %w", path, err)
		}
	}

	// Once the entities are extracted, we need to inflate the references
	return dc, nil
}

// extractEntities parses the entity from file bytes
// and updates the datacatalog struct with it.
func extractEntities(s *specs.Spec, dc *DataCatalog) error {
	// TODO: properly handle metadata - ensuring schema and types
	name := s.Metadata["name"].(string)
	switch s.Kind {
	case "properties":
		properties, err := ExtractProperties(s)
		if err != nil {
			return fmt.Errorf("extracting properties: %w", err)
		}
		dc.Properties[EntityGroup(name)] = properties

	case "events":
		events, err := ExtractEvents(s)
		if err != nil {
			return fmt.Errorf("extracting property entity: %w", err)
		}
		dc.Events[EntityGroup(name)] = events

	case "tp":
		tp, err := ExtractTrackingPlan(s)
		if err != nil {
			return fmt.Errorf("extracting tracking plan: %w", err)
		}

		dc.TrackingPlans[EntityGroup(name)] = &tp

	case "custom-types":
		customTypes, err := ExtractCustomTypes(s)
		if err != nil {
			return fmt.Errorf("extracting custom types: %w", err)
		}
		dc.CustomTypes[EntityGroup(name)] = customTypes

	default:
		return fmt.Errorf("unknown kind: %s", s.Kind)
	}

	return nil
}
