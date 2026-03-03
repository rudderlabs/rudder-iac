package rules

import (
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

func TestNewEventSpecSyntaxValidRule_V1Patterns(t *testing.T) {
	t.Parallel()

	rule := NewEventSpecSyntaxValidRule()

	patterns := rule.AppliesTo()
	assert.Contains(t, patterns, rules.MatchKindVersion("events", specs.SpecVersionV1),
		"Rule should include V1 match pattern")
}

func TestEventSpecV1SyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.EventSpecV1
	}{
		{
			name: "complete track event with all fields",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{
						LocalID:     "page_viewed",
						Name:        "Page Viewed",
						Type:        "track",
						Description: "User viewed a page on the website",
					},
					{
						LocalID:     "product_clicked",
						Name:        "Product Clicked",
						Type:        "track",
						Description: "User clicked on a product",
					},
				},
			},
		},
		{
			name: "track event with valid V1 category reference",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{
						LocalID:     "page_viewed",
						Name:        "Page Viewed",
						Type:        "track",
						CategoryRef: stringPtr("#categories:navigation"),
					},
				},
			},
		},
		{
			name: "non-track events without name",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "user_identified", Type: "identify"},
					{LocalID: "page_opened", Type: "page"},
					{LocalID: "screen_opened", Type: "screen"},
					{LocalID: "user_grouped", Type: "group"},
				},
			},
		},
		{
			name: "track event with minimum name length",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "e1", Name: "A", Type: "track"},
				},
			},
		},
		{
			name: "track event with maximum name length",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "e1", Name: strings.Repeat("a", 64), Type: "track"},
				},
			},
		},
		{
			name: "event with valid description starting with letter",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{
						LocalID:     "e1",
						Type:        "identify",
						Description: "Identifies a user in the system",
					},
				},
			},
		},
		{
			name: "empty events array is valid",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{},
			},
		},
		{
			name: "nil events array is valid",
			spec: localcatalog.EventSpecV1{
				Events: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateEventSpecV1(
				"events",
				specs.SpecVersionV1,
				map[string]any{},
				tt.spec,
			)
			assert.Empty(t, results, "Valid spec should not produce validation errors")
		})
	}
}

func TestEventSpecV1SyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.EventSpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "missing id",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{Name: "Page Viewed", Type: "track"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "missing event_type",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "page_viewed"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/event_type"},
			expectedMsgs:   []string{"'event_type' is required"},
		},
		{
			name: "invalid event_type",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "e1", Type: "invalid"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/event_type"},
			expectedMsgs:   []string{"'event_type' must be one of [track screen identify group page]"},
		},
		{
			name: "invalid event_type with name does not produce name error",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "e1", Name: "Some Name", Type: "pages"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/event_type"},
			expectedMsgs:   []string{"'event_type' must be one of [track screen identify group page]"},
		},
		{
			name: "track event with empty name",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "e1", Type: "track"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/name"},
			expectedMsgs:   []string{"name must be between 1 and 64 characters for track events"},
		},
		{
			name: "track event with name exceeding 64 chars",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "e1", Name: strings.Repeat("a", 65), Type: "track"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/name"},
			expectedMsgs:   []string{"name must be between 1 and 64 characters for track events"},
		},
		{
			name: "non-track event with name provided",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "e1", Name: "Should Not Have Name", Type: "identify"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/name"},
			expectedMsgs:   []string{"name should be empty for non-track events"},
		},
		{
			name: "description too short",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{
						LocalID:     "e1",
						Name:        "Valid Name",
						Type:        "track",
						Description: "ab",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/description"},
			expectedMsgs:   []string{"'description' length must be greater than or equal to 3"},
		},
		{
			name: "description too long",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{
						LocalID:     "e1",
						Name:        "Valid Name",
						Type:        "track",
						Description: "A" + strings.Repeat("a", 2000),
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/description"},
			expectedMsgs:   []string{"'description' length must be less than or equal to 2000"},
		},
		{
			name: "description not starting with letter",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{
						LocalID:     "e1",
						Name:        "Valid Name",
						Type:        "track",
						Description: "123 not starting with letter",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/description"},
			expectedMsgs:   []string{"'description' is not valid: must start with a letter [a-zA-Z]"},
		},
		{
			name: "invalid V1 category reference format",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{
						LocalID:     "e1",
						Name:        "Valid Name",
						Type:        "track",
						CategoryRef: stringPtr("invalid-reference"),
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/category"},
			expectedMsgs:   []string{"'category' is not valid: must be of pattern #categories:<id>"},
		},
		{
			name: "legacy category reference rejected in V1",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{
						LocalID:     "e1",
						Name:        "Valid Name",
						Type:        "track",
						CategoryRef: stringPtr("#/categories/user-events/navigation"),
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/category"},
			expectedMsgs:   []string{"'category' is not valid: must be of pattern #categories:<id>"},
		},
		{
			name: "multiple events with errors at different indices",
			spec: localcatalog.EventSpecV1{
				Events: []localcatalog.EventV1{
					{LocalID: "valid_event", Name: "Valid", Type: "track"},
					{LocalID: "e2", Type: "track"},
					{LocalID: "e3", Name: "Bad", Type: "identify"},
					{LocalID: "e4"},
				},
			},
			expectedErrors: 3,
			expectedRefs: []string{
				"/events/1/name",
				"/events/2/name",
				"/events/3/event_type",
			},
			expectedMsgs: []string{
				"name must be between 1 and 64 characters for track events",
				"name should be empty for non-track events",
				"'event_type' is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateEventSpecV1(
				"events",
				specs.SpecVersionV1,
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors, "Unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)

				assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "Validation error references don't match")
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Validation error messages don't match")
			}
		})
	}
}
