package localcatalog

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

var (
	PropRegex       = regexp.MustCompile(`^#\/properties\/(.*)\/(.*)$`)
	EventRegex      = regexp.MustCompile(`^#\/events\/(.*)\/(.*)$`)
	IncludeRegex    = regexp.MustCompile(`^#\/tp\/(.*)\/event_rule\/(.*)$`)
	CustomTypeRegex = regexp.MustCompile(`^#\/custom-types\/(.*)\/(.*)$`)
	CategoryRegex   = regexp.MustCompile(`^#\/categories\/(.*)\/(.*)$`)
)

type CatalogResourceFetcher interface {
	Event(group, id string) *Event
	Property(group, id string) *Property
	Category(group, id string) *Category
	CustomType(group, id string) *CustomType
	TPEventRule(group, id string) *TPRule
	TPEventRules(group string) ([]*TPRule, bool)
}

type TrackingPlan struct {
	Name        string    `json:"display_name"`
	LocalID     string    `json:"id"`
	Description string    `json:"description,omitempty"`
	Rules       []*TPRule `json:"rules,omitempty"`
}

type TPRule struct {
	Type       string            `json:"type"`
	LocalID    string            `json:"id"`
	Event      *TPRuleEvent      `json:"event"`
	Properties []*TPRuleProperty `json:"properties,omitempty"`
	Includes   *TPRuleIncludes   `json:"includes,omitempty"`
	Variants   Variants          `json:"variants,omitempty"`
}

type TPRuleEvent struct {
	Ref             string `json:"$ref"`
	AllowUnplanned  bool   `json:"allow_unplanned"`
	IdentitySection string `json:"identity_section"`
}

type TPRuleProperty struct {
	Ref                  string            `json:"$ref"`
	Required             bool              `json:"required"`
	AdditionalProperties *bool             `json:"additionalProperties,omitempty"`
	Properties           []*TPRuleProperty `json:"properties,omitempty"`
}

type TPRuleIncludes struct {
	Ref string `json:"$ref"`
}

func ExtractTrackingPlan(s *specs.Spec) (TrackingPlan, error) {
	log.Debug("extracting tracking plan from resource definition", "metadata.name", s.Metadata["name"])

	// The spec is the tracking plan in its enterity
	tp := TrackingPlan{}

	byt, err := json.Marshal(s.Spec)
	if err != nil {
		return TrackingPlan{}, fmt.Errorf("marshalling the spec")
	}

	if err := strictUnmarshal(byt, &tp); err != nil {
		return TrackingPlan{}, fmt.Errorf("unmarshalling the spec into tracking plan: %w", err)
	}

	return tp, nil
}
