package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestResourceKindVersionValidRule_Validate(t *testing.T) {
	t.Parallel()

	supportedKinds := []string{"properties", "events"}
	supportedVersions := []string{"rudder/0.1", "rudder/v1"}
	supportedPatterns := []rules.MatchPattern{
		rules.MatchKindVersion("properties", "rudder/0.1"),
		rules.MatchKindVersion("properties", "rudder/v1"),
		rules.MatchKindVersion("events", "rudder/v1"),
	}

	t.Run("valid combo produces no errors", func(t *testing.T) {
		rule := NewResourceKindVersionValidRule(supportedKinds, supportedVersions, supportedPatterns)
		results := rule.Validate(&rules.ValidationContext{
			Kind:    "properties",
			Version: "rudder/v1",
		})
		assert.Empty(t, results)
	})

	t.Run("unknown kind returns empty - gatekeeper handles it", func(t *testing.T) {
		rule := NewResourceKindVersionValidRule(supportedKinds, supportedVersions, supportedPatterns)
		results := rule.Validate(&rules.ValidationContext{
			Kind:    "blah",
			Version: "rudder/v1",
		})
		assert.Empty(t, results)
	})

	t.Run("invalid combo returns error", func(t *testing.T) {
		rule := NewResourceKindVersionValidRule(supportedKinds, supportedVersions, supportedPatterns)
		results := rule.Validate(&rules.ValidationContext{
			Kind:    "events",
			Version: "rudder/0.1",
		})
		assert.Len(t, results, 1)
		assert.Equal(t, "/kind", results[0].Reference)
		assert.Equal(t, "kind 'events' is not supported with version 'rudder/0.1'", results[0].Message)
	})

	t.Run("unknown version returns empty - gatekeeper handles it", func(t *testing.T) {
		// gatekeeper is spec-syntax-valid rule which checks for
		// valid kind and version
		rule := NewResourceKindVersionValidRule(supportedKinds, supportedVersions, supportedPatterns)
		results := rule.Validate(&rules.ValidationContext{
			Kind:    "properties",
			Version: "rudder/v2",
		})
		assert.Empty(t, results)
	})
}

func TestResourceKindVersionValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewResourceKindVersionValidRule(nil, nil, nil)

	t.Run("rule metadata is correct", func(t *testing.T) {
		assert.Equal(t, "project/resource-kind-version-valid", rule.ID())
		assert.Equal(t, rules.Error, rule.Severity())
		assert.Equal(t, "resource kind must be supported with the specified version", rule.Description())
		assert.Equal(t, []rules.MatchPattern{rules.MatchAll()}, rule.AppliesTo())
	})
}
