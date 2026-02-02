package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

// extractRefs extracts Reference fields from ValidationResults
func extractRefs(results []rules.ValidationResult) []string {
	refs := make([]string, len(results))
	for i, result := range results {
		refs[i] = result.Reference
	}
	return refs
}

// extractMsgs extracts Message fields from ValidationResults
func extractMsgs(results []rules.ValidationResult) []string {
	msgs := make([]string, len(results))
	for i, result := range results {
		msgs[i] = result.Message
	}
	return msgs
}

func TestNewEventSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewEventSpecSyntaxValidRule()

	assert.Equal(t, "datacatalog/events/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "event spec syntax must be valid", rule.Description())
	assert.Equal(t, []string{"events"}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid, "Rule should have valid examples")
	assert.NotEmpty(t, examples.Invalid, "Rule should have invalid examples")
}

func TestEventSpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.EventSpec
	}{
		{
			name: "complete spec with all fields populated",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
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
			name: "event with valid category reference",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{
						LocalID:     "page_viewed",
						Type:        "track",
						CategoryRef: stringPtr("#/categories/user-events/navigation"),
					},
				},
			},
		},
		{
			name: "multiple valid events with different types",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{LocalID: "page_viewed", Type: "track"},
					{LocalID: "user_identified", Type: "identify"},
					{LocalID: "screen_opened", Type: "screen"},
				},
			},
		},
		{
			name: "empty events array is valid",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{},
			},
		},
		{
			name: "nil events array is valid",
			spec: localcatalog.EventSpec{
				Events: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateEventSpec(
				"events",
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)
			assert.Empty(t, results, "Valid spec should not produce validation errors")
		})
	}
}

func TestEventSpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.EventSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "event missing id",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{
						Type: "track",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "event missing event_type",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{
						LocalID: "page_viewed",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/event_type"},
			expectedMsgs:   []string{"'event_type' is required"},
		},
		{
			name: "description too short",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{
						LocalID:     "page_viewed",
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
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{
						LocalID:     "page_viewed",
						Type:        "track",
						Description: string(make([]byte, 2001)),
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/description"},
			expectedMsgs:   []string{"'description' length must be less than or equal to 2000"},
		},
		{
			name: "invalid category reference format",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{
						LocalID:     "page_viewed",
						Type:        "track",
						CategoryRef: stringPtr("invalid-reference"),
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/0/category"},
			expectedMsgs:   []string{"'category' is not a valid reference format"},
		},
		{
			name: "multiple events with errors at different indices",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{LocalID: "valid_event", Type: "track"},
					{Type: "track"}, // missing id
					{LocalID: "missing_type"},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/events/1/id", "/events/2/event_type"},
			expectedMsgs:   []string{"'id' is required", "'event_type' is required"},
		},
		{
			name: "large array with error at last index",
			spec: localcatalog.EventSpec{
				Events: []localcatalog.Event{
					{LocalID: "event_1", Type: "track"},
					{LocalID: "event_2", Type: "track"},
					{LocalID: "event_3", Type: "track"},
					{LocalID: "event_4", Type: "track"},
					{LocalID: "event_5", Type: "track"},
					{LocalID: "event_6", Type: "track"},
					{LocalID: "event_7", Type: "track"},
					{LocalID: "event_8", Type: "track"},
					{LocalID: "event_9", Type: "track"},
					{LocalID: "event_10", Type: "track"},
					{
						Type: "track",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/events/10/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateEventSpec(
				"events",
				specs.SpecVersionV0_1,
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

// stringPtr is a helper function to create string pointers for testing
func stringPtr(s string) *string {
	return &s
}
