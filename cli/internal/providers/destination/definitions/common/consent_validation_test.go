package common_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
)

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
