package common_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
)

// Validate checks consent values against the registered consent_value
// pattern, which docs destination prints, so both must accept the same values.
func TestConsentValuePatternAgreesWithValidate(t *testing.T) {
	t.Parallel()

	pattern, ok := funcs.LookupPattern(common.ConsentValuePattern)
	assert.True(t, ok, "consent_value is registered")
	assert.Equal(t, funcs.PatternInfo{
		Regex:   `^(\{\{.*\|\|.*\}\}|env\..+|.{0,100})$`,
		Message: "must be at most 100 characters",
	}, pattern)

	cases := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "an empty value", value: "", valid: true},
		{name: "100 characters", value: strings.Repeat("a", 100), valid: true},
		{name: "101 characters", value: strings.Repeat("a", 101)},
		{name: "a line break", value: "a\nb"},
		{name: "a long template with a fallback", value: "{{ .CONSENT || " + strings.Repeat("a", 100) + " }}", valid: true},
		{name: "a long template without a fallback", value: "{{ ." + strings.Repeat("A", 100) + " }}"},
		{name: "a long env value", value: "env." + strings.Repeat("A", 100), valid: true},
		{name: "env. alone", value: "env.", valid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, funcs.MatchPattern(common.ConsentValuePattern, tc.value), "the registered pattern")

			var want []common.ConsentValidationError
			if !tc.valid {
				want = []common.ConsentValidationError{{Path: "/0/consents/0", Message: "'consent' must be at most 100 characters"}}
			}
			got := common.ValidateConsentEntries([]common.ConsentEntry{{Provider: "ketch", Consents: []string{tc.value}}})
			assert.Equal(t, want, got, "validate")
		})
	}
}

func TestValidateConsentEntriesRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	errors := common.ValidateConsentEntries([]common.ConsentEntry{{
		Provider: "unknown",
	}})

	require.Len(t, errors, 1)
	assert.Equal(t, common.ConsentValidationError{
		Path:    "/0/provider",
		Message: "'provider' must be one of [custom iubenda ketch oneTrust]",
	}, errors[0])
}

func TestValidateConsentEntriesRequiresResolutionForCustomProvider(t *testing.T) {
	t.Parallel()

	errors := common.ValidateConsentEntries([]common.ConsentEntry{{
		Provider: "custom",
	}})

	require.Len(t, errors, 1)
	assert.Equal(t, common.ConsentValidationError{
		Path:    "/0/resolution_strategy",
		Message: "'resolution_strategy' is required when 'provider' is custom",
	}, errors[0])
}

func TestValidateConsentEntriesRejectsInvalidCustomResolution(t *testing.T) {
	t.Parallel()

	errors := common.ValidateConsentEntries([]common.ConsentEntry{{
		Provider:           "custom",
		ResolutionStrategy: "xor",
	}})

	require.Len(t, errors, 1)
	assert.Equal(t, common.ConsentValidationError{
		Path:    "/0/resolution_strategy",
		Message: "'resolution_strategy' must be one of [and or]",
	}, errors[0])
}

func TestValidateConsentEntriesRejectsDuplicateProvider(t *testing.T) {
	t.Parallel()

	errors := common.ValidateConsentEntries([]common.ConsentEntry{
		{Provider: "oneTrust"},
		{Provider: "oneTrust"},
	})

	require.Len(t, errors, 1)
	assert.Equal(t, common.ConsentValidationError{
		Path:    "/1/provider",
		Message: "only one consent entry can be configured per provider",
	}, errors[0])
}

func TestValidateConsentEntriesRejectsLongPlainConsent(t *testing.T) {
	t.Parallel()

	errors := common.ValidateConsentEntries([]common.ConsentEntry{{
		Provider: "oneTrust",
		Consents: []string{strings.Repeat("a", 101)},
	}})

	require.Len(t, errors, 1)
	assert.Equal(t, common.ConsentValidationError{
		Path:    "/0/consents/0",
		Message: "'consent' must be at most 100 characters",
	}, errors[0])
}

func TestValidateConsentEntriesAcceptsTemplateAndEnvironmentConsents(t *testing.T) {
	t.Parallel()

	errors := common.ValidateConsentEntries([]common.ConsentEntry{{
		Provider: "oneTrust",
		Consents: []string{
			"{{ .CONSENT_CATEGORY || analytics }}",
			"env.CONSENT_CATEGORY",
		},
	}})

	assert.Empty(t, errors)
}

func TestValidateConsentEntriesRejectsMissingProvider(t *testing.T) {
	t.Parallel()

	errors := common.ValidateConsentEntries([]common.ConsentEntry{{}})

	require.Len(t, errors, 1)
	assert.Equal(t, common.ConsentValidationError{
		Path:    "/0/provider",
		Message: "'provider' must be one of [custom iubenda ketch oneTrust]",
	}, errors[0])
}

func TestValidateConsentEntriesRejectsEmptyProvider(t *testing.T) {
	t.Parallel()

	errors := common.ValidateConsentEntries([]common.ConsentEntry{{
		Provider: "",
	}})

	require.Len(t, errors, 1)
	assert.Equal(t, common.ConsentValidationError{
		Path:    "/0/provider",
		Message: "'provider' must be one of [custom iubenda ketch oneTrust]",
	}, errors[0])
}
