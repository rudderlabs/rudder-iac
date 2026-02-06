package rules

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomTypeReferences(t *testing.T) {
	t.Parallel()

	t.Run("legacy format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(customTypeLegacyReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#/custom-types/user-data/user-id", wantMatch: true},
			{name: "with underscores", reference: "#/custom-types/group_1/id_2", wantMatch: true},
			{name: "single char", reference: "#/custom-types/a/b", wantMatch: true},
			{name: "mixed case and numbers", reference: "#/custom-types/my-group/MY_ID123", wantMatch: true},

			// Invalid cases
			{name: "missing id", reference: "#/custom-types/group/", wantMatch: false},
			{name: "missing group", reference: "#/custom-types//id", wantMatch: false},
			{name: "missing leading slash", reference: "#custom-types/group/id", wantMatch: false},
			{name: "too many segments", reference: "#/custom-types/group/id/extra", wantMatch: false},
			{name: "space in group", reference: "#/custom-types/gro up/id", wantMatch: false},
			{name: "invalid char in id", reference: "#/custom-types/group/i@d", wantMatch: false},
			{name: "empty", reference: "", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})

	t.Run("new format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(customTypeReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#custom-types:user-id", wantMatch: true},
			{name: "with underscores", reference: "#custom-types:MY_ID_123", wantMatch: true},
			{name: "single char", reference: "#custom-types:a", wantMatch: true},

			// Invalid cases
			{name: "missing id", reference: "#custom-types:", wantMatch: false},
			{name: "space in id", reference: "#custom-types:id with space", wantMatch: false},
			{name: "wrong prefix with slash", reference: "#/custom-types:id", wantMatch: false},
			{name: "missing hash", reference: "custom-types:id", wantMatch: false},
			{name: "invalid char", reference: "#custom-types:id@123", wantMatch: false},
			{name: "empty", reference: "", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})
}

func TestPropertyReferences(t *testing.T) {
	t.Parallel()

	t.Run("legacy format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(propertyLegacyReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#/properties/user-props/email", wantMatch: true},
			{name: "single char", reference: "#/properties/g1/p1", wantMatch: true},
			{name: "with underscores", reference: "#/properties/group_1/prop_id", wantMatch: true},

			// Invalid cases
			{name: "missing id", reference: "#/properties/group/", wantMatch: false},
			{name: "missing group", reference: "#/properties//id", wantMatch: false},
			{name: "space in group", reference: "#/properties/gro up/id", wantMatch: false},
			{name: "invalid char", reference: "#/properties/group/id@123", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})

	t.Run("new format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(propertyReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#properties:email-address", wantMatch: true},
			{name: "with underscores", reference: "#properties:user_id", wantMatch: true},

			// Invalid cases
			{name: "missing id", reference: "#properties:", wantMatch: false},
			{name: "space in id", reference: "#properties:id with space", wantMatch: false},
			{name: "invalid char", reference: "#properties:id@123", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})
}

func TestEventReferences(t *testing.T) {
	t.Parallel()

	t.Run("legacy format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(eventLegacyReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#/events/user-events/login", wantMatch: true},
			{name: "with underscores", reference: "#/events/analytics/page_view", wantMatch: true},

			// Invalid cases
			{name: "missing id", reference: "#/events/group/", wantMatch: false},
			{name: "too many segments", reference: "#/events/group/id/extra", wantMatch: false},
			{name: "invalid char", reference: "#/events/group/id@", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})

	t.Run("new format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(eventReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#events:user-login", wantMatch: true},
			{name: "with underscores and numbers", reference: "#events:PAGE_VIEW_123", wantMatch: true},

			// Invalid cases
			{name: "missing id", reference: "#events:", wantMatch: false},
			{name: "space in id", reference: "#events:user login", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})
}

func TestCategoryReferences(t *testing.T) {
	t.Parallel()

	t.Run("legacy format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(categoryLegacyReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#/categories/user-cats/profile", wantMatch: true},
			{name: "with underscores", reference: "#/categories/cat_1/id_2", wantMatch: true},

			// Invalid cases
			{name: "missing id", reference: "#/categories/group/", wantMatch: false},
			{name: "missing group", reference: "#/categories//id", wantMatch: false},
			{name: "wrong kind", reference: "#/category/group/id", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})

	t.Run("new format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(categoryReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#categories:profile-updates", wantMatch: true},
			{name: "with underscores", reference: "#categories:user_profile", wantMatch: true},

			// Invalid cases
			{name: "missing id", reference: "#categories:", wantMatch: false},
			{name: "wrong format", reference: "#category:id", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})
}

func TestTrackingPlanReferences(t *testing.T) {
	t.Parallel()

	t.Run("legacy format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(trackingPlanLegacyReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#/tp/mobile/ios-plan", wantMatch: true},
			{name: "with underscores", reference: "#/tp/web/main_plan", wantMatch: true},

			// Invalid cases
			{name: "wrong kind", reference: "#/tracking-plan/group/id", wantMatch: false},
			{name: "missing id", reference: "#/tp/group/", wantMatch: false},
			{name: "missing group", reference: "#/tp//id", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})

	t.Run("new format", func(t *testing.T) {
		t.Parallel()

		regex := regexp.MustCompile(trackingPlanReferenceRegex)

		tests := []struct {
			name      string
			reference string
			wantMatch bool
		}{
			// Valid cases
			{name: "basic reference", reference: "#tracking-plan:ios-main", wantMatch: true},
			{name: "with underscores", reference: "#tracking-plan:web_plan_v2", wantMatch: true},

			// Invalid cases
			{name: "wrong prefix", reference: "#tp:plan", wantMatch: false},
			{name: "missing id", reference: "#tracking-plan:", wantMatch: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := regex.MatchString(tt.reference)
				assert.Equal(t, tt.wantMatch, got, "reference: %s", tt.reference)
			})
		}
	})
}

func TestConstantValues(t *testing.T) {
	t.Run("tag constants", func(t *testing.T) {
		tests := []struct {
			name     string
			constant string
			expected string
		}{
			{"customTypeLegacyReferenceTag", customTypeLegacyReferenceTag, "legacy_custom_type_ref"},
			{"customTypeReferenceTag", customTypeReferenceTag, "custom_type_ref"},
			{"propertyLegacyReferenceTag", propertyLegacyReferenceTag, "legacy_property_ref"},
			{"propertyReferenceTag", propertyReferenceTag, "property_ref"},
			{"eventLegacyReferenceTag", eventLegacyReferenceTag, "legacy_event_ref"},
			{"eventReferenceTag", eventReferenceTag, "event_ref"},
			{"categoryLegacyReferenceTag", categoryLegacyReferenceTag, "legacy_category_ref"},
			{"categoryReferenceTag", categoryReferenceTag, "category_ref"},
			{"trackingPlanLegacyReferenceTag", trackingPlanLegacyReferenceTag, "legacy_tracking_plan_ref"},
			{"trackingPlanReferenceTag", trackingPlanReferenceTag, "tracking_plan_ref"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.constant)
			})
		}
	})

	t.Run("message constants", func(t *testing.T) {
		tests := []struct {
			name     string
			constant string
			expected string
		}{
			{"customTypeLegacyReferenceMessage", customTypeLegacyReferenceMessage, "must be of pattern #/custom-types/<group>/<id>"},
			{"customTypeReferenceMessage", customTypeReferenceMessage, "must be of pattern #custom-types:<id>"},
			{"propertyLegacyReferenceMessage", propertyLegacyReferenceMessage, "must be of pattern #/properties/<group>/<id>"},
			{"propertyReferenceMessage", propertyReferenceMessage, "must be of pattern #properties:<id>"},
			{"eventLegacyReferenceMessage", eventLegacyReferenceMessage, "must be of pattern #/events/<group>/<id>"},
			{"eventReferenceMessage", eventReferenceMessage, "must be of pattern #events:<id>"},
			{"categoryLegacyReferenceMessage", categoryLegacyReferenceMessage, "must be of pattern #/categories/<group>/<id>"},
			{"categoryReferenceMessage", categoryReferenceMessage, "must be of pattern #categories:<id>"},
			{"trackingPlanLegacyReferenceMessage", trackingPlanLegacyReferenceMessage, "must be of pattern #/tp/<group>/<id>"},
			{"trackingPlanReferenceMessage", trackingPlanReferenceMessage, "must be of pattern #tracking-plan:<id>"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.constant)
			})
		}
	})
}
