package localcatalog

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var (
	PropRegex    = regexp.MustCompile(`^#\/properties\/(.*)\/(.*)$`)
	EventRegex   = regexp.MustCompile(`^#\/events\/(.*)\/(.*)$`)
	IncludeRegex = regexp.MustCompile(`^#\/tp\/(.*)\/event_rule\/(.*)$`)
)

type CatalogResourceFetcher interface {
	Event(group, id string) *Event
	Property(group, id string) *Property
	TPEventRule(group, id string) *TPRule
	TPEventRules(group string) ([]*TPRule, bool)
}

type TrackingPlan struct {
	Name        string    `json:"display_name"`
	LocalID     string    `json:"id"`
	Description string    `json:"description"`
	Rules       []*TPRule `json:"rules"`
	// Event and Props underneath event on the tracking plan
	// This is automatically generated by the code when expanding refs
	EventProps []*TPEvent `json:"event_props"`
}

type TPEvent struct {
	Name           string             `json:"name"`
	LocalID        string             `json:"id"`
	Description    string             `json:"description"`
	Type           string             `json:"type"`
	AllowUnplanned bool               `json:"allow_unplanned"`
	Properties     []*TPEventProperty `json:"properties"`
}

func (e *TPEvent) PropertyByLocalID(localID string) *TPEventProperty {
	for _, prop := range e.Properties {
		if prop.LocalID == localID {
			return prop
		}
	}
	return nil
}

type TPEventProperty struct {
	Name        string                 `json:"name"`
	LocalID     string                 `json:"id"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Required    bool                   `json:"required"`
}

type TPRule struct {
	Type       string            `json:"type"`
	LocalID    string            `json:"id"`
	Event      *TPRuleEvent      `json:"event"`
	Properties []*TPRuleProperty `json:"properties"`
	Includes   *TPRuleIncludes   `json:"includes"`
}

type TPRuleEvent struct {
	Ref            string `json:"$ref"`
	AllowUnplanned bool   `json:"allow_unplanned"`
}

type TPRuleProperty struct {
	Ref      string `json:"$ref"`
	Required bool   `json:"required"`
}

type TPRuleIncludes struct {
	Ref string `json:"$ref"`
}

// ExpandRefs simply expands the references being held
// when reading the tracking plan with the actual events and properties
func (tp *TrackingPlan) ExpandRefs(dc *DataCatalog) error {
	log.Debug("expanding refs for the tracking plan", "id", tp.LocalID)

	expandedEvents := make([]*TPEvent, 0)
	for _, rule := range tp.Rules {

		switch {
		case rule.Event != nil:
			tpEvent, err := expandEventRefs(rule, dc)
			if err != nil {
				return fmt.Errorf("expanding event refs on the rule: %s in tracking plan: %s, err:%w",
					rule.LocalID,
					tp.LocalID,
					err)
			}
			expandedEvents = append(expandedEvents, tpEvent)

		case rule.Includes != nil:
			tpEvents, err := expandIncludeRefs(rule, dc)
			if err != nil {
				return fmt.Errorf("expanding include refs on the rule: %s in tracking plan: %s, err:%w",
					rule.LocalID,
					tp.LocalID,
					err)
			}
			expandedEvents = append(expandedEvents, tpEvents...)

		default:
			return fmt.Errorf("both the event and includes section in the rule:%s for tp: %s are nil", rule.LocalID, tp.LocalID)
		}

	}
	tp.EventProps = expandedEvents
	return nil
}

// expandIncludeRefs expands the include references in the tracking plan rule definition
// TODO: Make this function recursive to allow for multiple levels of include
func expandIncludeRefs(rule *TPRule, fetcher CatalogResourceFetcher) ([]*TPEvent, error) {
	log.Debug("expanding include refs within the rule", "ruleID", rule.LocalID)

	if rule.Includes == nil {
		return nil, fmt.Errorf("empty rule includes")
	}

	matches := IncludeRegex.FindStringSubmatch(rule.Includes.Ref)
	if len(matches) != 3 {
		return nil, fmt.Errorf("includes ref: %s invalid as failed regex match", rule.Includes.Ref)
	}

	tpGroup, ruleID := matches[1], matches[2]
	rules := make([]*TPRule, 0)

	if ruleID == "*" {
		eventRules, _ := fetcher.TPEventRules(tpGroup)
		rules = append(rules, eventRules...) // fetch all the tp rules in the group
	} else {
		rules = append(rules, fetcher.TPEventRule(tpGroup, ruleID)) // fetch the specific rule with the tpGroup
	}

	toReturn := make([]*TPEvent, 0)
	// Assume rules are now actual rules and not indirections
	for _, rule := range rules {

		if rule.Event == nil {
			continue
		}

		// This rule should have event ref only
		// which we can expand now
		event, err := expandEventRefs(rule, fetcher)
		if err != nil {
			return nil, fmt.Errorf("expanding event ref of the expanded include rule: %s failed, err: %w", rule.LocalID, err)
		}

		toReturn = append(toReturn, event)
	}

	return toReturn, nil
}

// expandEventRefs expands the direct event references in the tracking plan rule definition
func expandEventRefs(rule *TPRule, fetcher CatalogResourceFetcher) (*TPEvent, error) {
	log.Debug("expanding event refs within the rule", "ruleID", rule.LocalID)

	if rule.Event == nil {
		return nil, fmt.Errorf("empty rule event")
	}

	matches := EventRegex.FindStringSubmatch(rule.Event.Ref)
	if len(matches) != 3 {
		return nil, fmt.Errorf("event ref: %s invalid as failed regex match", rule.Event.Ref)
	}

	eventGroup, eventID := matches[1], matches[2]
	event := fetcher.Event(eventGroup, eventID)
	if event == nil {
		return nil, fmt.Errorf("looking up event: %s in group: %s failed", eventID, eventGroup)
	}

	toReturn := TPEvent{
		Name:           event.Name,
		LocalID:        event.LocalID,
		Description:    event.Description,
		Type:           event.Type,
		AllowUnplanned: rule.Event.AllowUnplanned,
		Properties:     make([]*TPEventProperty, 0),
	}

	// Load the properties from the data catalog
	// into corresponding event on the tracking plan
	for _, prop := range rule.Properties {
		matches = PropRegex.FindStringSubmatch(prop.Ref)
		if len(matches) != 3 {
			return nil, fmt.Errorf("property ref: %s invalid as failed regex match", prop.Ref)
		}

		propertyGroup, propertyID := matches[1], matches[2]
		property := fetcher.Property(propertyGroup, propertyID)
		if property == nil {
			return nil, fmt.Errorf("looking up property: %s in group: %s failed", propertyID, propertyGroup)
		}

		toReturn.Properties = append(toReturn.Properties, &TPEventProperty{
			Name:        property.Name,
			LocalID:     property.LocalID,
			Description: property.Description,
			Type:        property.Type,
			Required:    prop.Required,
			Config:      property.Config,
		})
	}

	return &toReturn, nil
}

func ExtractTrackingPlan(rd *ResourceDefinition) (TrackingPlan, error) {
	log.Debug("extracting tracking plan from resource definition", "metadata.name", rd.Metadata.Name)

	// The spec is the tracking plan in its enterity
	tp := TrackingPlan{}

	byt, err := json.Marshal(rd.Spec)
	if err != nil {
		return TrackingPlan{}, fmt.Errorf("marshalling the spec")
	}

	if err := json.Unmarshal(byt, &tp); err != nil {
		return TrackingPlan{}, fmt.Errorf("unmarshalling the spec into tracking plan")
	}

	return tp, nil
}
