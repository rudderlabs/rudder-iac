package localcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackingPlanV1_ExpandRefs(t *testing.T) {
	t.Run("expands single rule with event ref and sets EventProps", func(t *testing.T) {
		eventRef := "#event:signup"
		tp := &TrackingPlanV1{
			LocalID: "my-tp",
			Name:    "My Plan",
			Rules: []*TPRuleV1{
				{
					Type:                 "event_rule",
					LocalID:              "signup-rule",
					Event:                eventRef,
					IdentitySection:      "properties",
					AdditionalProperties: false,
					Properties:           []*TPRulePropertyV1{},
				},
			},
		}
		dc := &DataCatalog{
			Events: []Event{
				{
					LocalID:     "signup",
					Name:        "User Sign Up",
					Type:        "track",
					Description: "User signed up",
				},
			},
			Properties:     []PropertyV1{},
			TrackingPlans:  []*TrackingPlanV1{},
			CustomTypes:    []CustomTypeV1{},
			Categories:     []Category{},
			ReferenceMap:   make(map[string]string),
			ImportMetadata: make(map[string]*WorkspaceRemoteIDMapping),
		}

		err := tp.ExpandRefs(dc)
		require.NoError(t, err)
		require.Len(t, tp.EventProps, 1)
		assert.Equal(t, "User Sign Up", tp.EventProps[0].Name)
		assert.Equal(t, "signup", tp.EventProps[0].LocalID)
		assert.Equal(t, eventRef, tp.EventProps[0].Ref)
		assert.Equal(t, "track", tp.EventProps[0].Type)
		assert.Equal(t, "properties", tp.EventProps[0].IdentitySection)
		assert.False(t, tp.EventProps[0].AllowUnplanned)
	})

	t.Run("expands multiple rules with event refs", func(t *testing.T) {
		tp := &TrackingPlanV1{
			LocalID: "multi-tp",
			Name:    "Multi Event Plan",
			Rules: []*TPRuleV1{
				{Type: "event_rule", LocalID: "r1", Event: "#event:page_viewed", IdentitySection: "properties"},
				{Type: "event_rule", LocalID: "r2", Event: "#event:button_clicked", IdentitySection: "context"},
			},
		}
		dc := &DataCatalog{
			Events: []Event{
				{LocalID: "page_viewed", Name: "Page Viewed", Type: "track"},
				{LocalID: "button_clicked", Name: "Button Clicked", Type: "track"},
			},
			Properties:     []PropertyV1{},
			TrackingPlans:  []*TrackingPlanV1{},
			CustomTypes:    []CustomTypeV1{},
			Categories:     []Category{},
			ReferenceMap:   make(map[string]string),
			ImportMetadata: make(map[string]*WorkspaceRemoteIDMapping),
		}

		err := tp.ExpandRefs(dc)
		require.NoError(t, err)
		require.Len(t, tp.EventProps, 2)
		assert.Equal(t, "Page Viewed", tp.EventProps[0].Name)
		assert.Equal(t, "page_viewed", tp.EventProps[0].LocalID)
		assert.Equal(t, "Button Clicked", tp.EventProps[1].Name)
		assert.Equal(t, "button_clicked", tp.EventProps[1].LocalID)
	})

	t.Run("expands event ref in old format (#/events/group/id)", func(t *testing.T) {
		tp := &TrackingPlanV1{
			LocalID: "old-format-tp",
			Rules: []*TPRuleV1{
				{Type: "event_rule", LocalID: "r1", Event: "#/events/default/login", IdentitySection: "properties"},
			},
		}
		dc := &DataCatalog{
			Events: []Event{
				{LocalID: "login", Name: "User Login", Type: "track"},
			},
			Properties:     []PropertyV1{},
			TrackingPlans:  []*TrackingPlanV1{},
			CustomTypes:    []CustomTypeV1{},
			Categories:     []Category{},
			ReferenceMap:   make(map[string]string),
			ImportMetadata: make(map[string]*WorkspaceRemoteIDMapping),
		}

		err := tp.ExpandRefs(dc)
		require.NoError(t, err)
		require.Len(t, tp.EventProps, 1)
		assert.Equal(t, "User Login", tp.EventProps[0].Name)
		assert.Equal(t, "login", tp.EventProps[0].LocalID)
	})

	t.Run("errors when rule has neither event nor includes", func(t *testing.T) {
		tp := &TrackingPlanV1{
			LocalID: "bad-tp",
			Rules: []*TPRuleV1{
				{
					Type:     "event_rule",
					LocalID:  "empty-rule",
					Event:    "",
					Includes: nil,
				},
			},
		}
		dc := &DataCatalog{
			Events:         []Event{},
			Properties:     []PropertyV1{},
			TrackingPlans:  []*TrackingPlanV1{},
			CustomTypes:    []CustomTypeV1{},
			Categories:     []Category{},
			ReferenceMap:   make(map[string]string),
			ImportMetadata: make(map[string]*WorkspaceRemoteIDMapping),
		}

		err := tp.ExpandRefs(dc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "both the event and includes section")
		assert.Contains(t, err.Error(), "empty-rule")
		assert.Contains(t, err.Error(), "bad-tp")
	})

	t.Run("errors when event ref does not exist in catalog", func(t *testing.T) {
		tp := &TrackingPlanV1{
			LocalID: "missing-event-tp",
			Rules: []*TPRuleV1{
				{Type: "event_rule", LocalID: "r1", Event: "#event:nonexistent", IdentitySection: "properties"},
			},
		}
		dc := &DataCatalog{
			Events:         []Event{},
			Properties:     []PropertyV1{},
			TrackingPlans:  []*TrackingPlanV1{},
			CustomTypes:    []CustomTypeV1{},
			Categories:     []Category{},
			ReferenceMap:   make(map[string]string),
			ImportMetadata: make(map[string]*WorkspaceRemoteIDMapping),
		}

		err := tp.ExpandRefs(dc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "looking up event")
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("errors when event ref format is invalid", func(t *testing.T) {
		tp := &TrackingPlanV1{
			LocalID: "invalid-ref-tp",
			Rules: []*TPRuleV1{
				{Type: "event_rule", LocalID: "r1", Event: "not-a-valid-ref", IdentitySection: "properties"},
			},
		}
		dc := &DataCatalog{
			Events:         []Event{},
			Properties:     []PropertyV1{},
			TrackingPlans:  []*TrackingPlanV1{},
			CustomTypes:    []CustomTypeV1{},
			Categories:     []Category{},
			ReferenceMap:   make(map[string]string),
			ImportMetadata: make(map[string]*WorkspaceRemoteIDMapping),
		}

		err := tp.ExpandRefs(dc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid as failed regex match")
	})
}
