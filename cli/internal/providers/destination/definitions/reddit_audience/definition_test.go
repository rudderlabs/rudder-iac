package redditaudience_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	redditaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/reddit_audience"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(redditaudience.NewDefinition()))

	registered, err := registry.Get("reddit_audience", 1)
	require.NoError(t, err)
	return registered
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(redditaudience.NewDefinition()))

	registered, err := registry.Get("reddit_audience", 1)
	require.NoError(t, err)

	assert.Equal(t, "reddit_audience", registered.Type)
	assert.Equal(t, "REDDIT_AUDIENCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	assert.Equal(t, []string{"warehouse"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("warehouse")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud"}, modes)

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("REDDIT_AUDIENCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestRedditAudienceConfigValidation(t *testing.T) {
	t.Parallel()

	registered := register(t)

	t.Run("missing rudder_account_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.Len(t, errors, 1)
		assert.Equal(t, "/rudder_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("rudder_account_id rejects overlong and multiline values", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{strings.Repeat("a", 101), "bad\nvalue"} {
			errors := registered.ValidateConfig(map[string]any{"rudder_account_id": value})
			require.Len(t, errors, 1, value)
			assert.Equal(t, "/rudder_account_id", errors[0].Path)
		}
	})

	// Upstream wraps ad_account_id's constraint in a {{ … || … }} alternative,
	// unlike rudder_account_id's bare regex, so a template is accepted here.
	t.Run("ad_account_id accepts a template and rejects a multiline literal", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "reddit-account",
			"ad_account_id":     "{{ .REDDIT_AD_ACCOUNT_ID || t2_abc123 }}",
		}))

		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "reddit-account",
			"ad_account_id":     "t2\nabc123",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/ad_account_id", errors[0].Path)
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "reddit-account",
			"connection_mode":   map[string]any{"warehouse": "device"},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("consent_management rejected as unknown", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id":  "reddit-account",
			"consent_management": map[string]any{"warehouse": []any{}},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/consent_management", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	// The spec shipped in the README.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "reddit-audience-account",
			"ad_account_id":     "t2_abc123",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "reddit-account",
			"not_a_field":       true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})
}

func TestRedditAudienceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := redditaudience.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name:      "minimal",
			LocalJSON: `{"rudder_account_id": "reddit-account"}`,
			APIJSON:   `{"rudderAccountId": "reddit-account"}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"rudder_account_id": "reddit-account",
				"ad_account_id": "t2_abc123",
				"connection_mode": {"warehouse": "cloud"}
			}`,
			APIJSON: `{
				"rudderAccountId": "reddit-account",
				"adAccountId": "t2_abc123",
				"connectionMode": {"warehouse": "cloud"}
			}`,
		},
	})
}
