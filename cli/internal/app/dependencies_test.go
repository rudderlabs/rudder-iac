package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeProvidersIncludesGAProviders(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	c, err := client.New("test-token")
	require.NoError(t, err)

	composite, providers, err := composeProviders(c)
	require.NoError(t, err)
	require.NotNil(t, providers.DataGraph)
	require.NotNil(t, providers.Account)
	require.NotNil(t, providers.Destination)

	cp, ok := composite.(*provider.CompositeProvider)
	require.True(t, ok)
	assert.Same(t, providers.DataGraph, cp.Providers["datagraph"])
	assert.Same(t, providers.Account, cp.Providers["account"])
	assert.Same(t, providers.Destination, cp.Providers["destination"])
	assert.Equal(t, providerOrder, cp.Order)
}

// A provider whose matchers consult another provider's matches, or whose
// specs reference its resources, must come after it in providerOrder.
func TestProviderOrderSatisfiesProviderEdges(t *testing.T) {
	t.Parallel()

	for dependent, dependencies := range providerEdges {
		for _, dependency := range dependencies {
			var (
				at     = slices.Index(providerOrder, dependent)
				before = slices.Index(providerOrder, dependency)
			)
			require.NotEqual(t, -1, at, "%s is not in providerOrder", dependent)
			require.NotEqual(t, -1, before, "%s is not in providerOrder", dependency)
			assert.Less(t, before, at, "%s must precede %s in providerOrder", dependency, dependent)
		}
	}
}

// The recording server stands in for the backend so the remote load is
// observable without credentials. Experimental mode is off: the table and
// connection kinds must register without any flag. InitConfig mutates
// process-global viper, so later tests that read config must re-init.
func TestComposeProvidersRegistersRETLTableAndConnection(t *testing.T) {
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "false")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	var hitConnections atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v2/retl-connections") {
			hitConnections.Store(true)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	c, err := client.New("test-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	_, providers, err := composeProviders(c)
	require.NoError(t, err)
	assert.Contains(t, providers.RETL.SupportedKinds(), connection.ResourceKind)
	assert.Contains(t, providers.RETL.SupportedKinds(), table.ResourceKind)

	_, err = providers.RETL.LoadResourcesFromRemote(t.Context())
	require.NoError(t, err)
	assert.True(t, hitConnections.Load())
}

func TestNewDestinationRegistryFlagMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                   string
		unverifiedDestinations bool
		wantTypes              []string
	}{
		{
			name:                   "unverifiedDestinations disabled registers verified destinations",
			unverifiedDestinations: false,
			wantTypes:              []string{"active_campaign", "adj", "adobe_analytics", "am", "attentive_tag", "bq", "bqstream", "braze", "confluent_cloud", "customerio", "facebook_conversions", "facebook_pixel", "firebase", "ga", "ga4", "gcs", "googleads", "googlepubsub", "googlesheets", "gtm", "hs", "http", "intercom", "iterable", "kafka", "kinesis", "linkedin_insight_tag", "marketo", "mp", "postgres", "posthog", "qualtrics", "redis", "rs", "s3", "s3_datalake", "salesforce", "sentry", "slack", "snowflake", "snowpipe_streaming", "statsig", "tiktok_ads", "vwo", "webhook", "zendesk"},
		},
		{
			name:                   "unverifiedDestinations enabled registers verified and unverified destinations",
			unverifiedDestinations: true,
			wantTypes:              []string{"active_campaign", "adj", "adobe_analytics", "am", "attentive_tag", "bingads_offline_conversions", "bq", "bqstream", "braze", "confluent_cloud", "customerio", "customerio_audience", "facebook_conversions", "facebook_pixel", "firebase", "ga", "ga4", "gcs", "google_adwords_offline_conversions", "googleads", "googlepubsub", "googlesheets", "gtm", "hs", "http", "intercom", "iterable", "kafka", "kinesis", "linkedin_ads", "linkedin_insight_tag", "marketo", "mp", "postgres", "posthog", "qualtrics", "redis", "rs", "s3", "s3_datalake", "salesforce", "sentry", "slack", "snowflake", "snowpipe_streaming", "statsig", "tiktok_ads", "vwo", "webhook", "zendesk"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Config{
				ExperimentalFlags: config.ExperimentalConfig{
					UnverifiedDestinations: tc.unverifiedDestinations,
				},
			}

			registry, err := newDestinationRegistry(cfg)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTypes, registry.SupportedTypes())
		})
	}
}

// Every definition that models connection_mode must also convert it. A config
// field without the matching converter properties passes validation but drops
// the key from the API payload, erasing whatever the user set upstream on the
// first apply — so assert the mapping fleet-wide rather than per definition.
func TestDestinationConnectionModeIsConverted(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		ExperimentalFlags: config.ExperimentalConfig{
			UnverifiedDestinations: true,
		},
	}

	registry, err := newDestinationRegistry(cfg)
	require.NoError(t, err)

	apiSourceTypes := common.LocalToAPISourceTypes()

	for _, destType := range registry.SupportedTypes() {
		versions, err := registry.Versions(destType)
		require.NoError(t, err)

		for _, version := range versions {
			def, err := registry.Get(destType, version)
			require.NoError(t, err)

			if !modelsConnectionMode(def) {
				continue
			}

			for _, sourceType := range def.SupportedSourceTypes() {
				modes, err := def.ConnectionModes(sourceType)
				require.NoError(t, err)
				require.NotEmpty(t, modes, "%s: no connection modes for %s", destType, sourceType)

				api, err := def.LocalToAPI(map[string]any{
					"connection_mode": map[string]any{sourceType: modes[0]},
				})
				require.NoError(t, err)

				assert.Equal(t,
					map[string]any{apiSourceTypes[sourceType]: modes[0]},
					api["connectionMode"],
					"%s: connection_mode.%s is not converted", destType, sourceType,
				)
			}
		}
	}
}

// modelsConnectionMode reports whether the definition's config struct declares
// the key, distinguished by the closed allowlist rejecting it when it does not.
func modelsConnectionMode(def *definitions.RegisteredDefinition) bool {
	for _, err := range def.ValidateConfig(map[string]any{"connection_mode": map[string]any{}}) {
		if err.Path == "/connection_mode" && strings.Contains(err.Message, "unknown config field") {
			return false
		}
	}
	return true
}

type workspaceHTTPClientFunc func(*http.Request) (*http.Response, error)

func (f workspaceHTTPClientFunc) Do(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNewDepsDiscoversEUWorkspace(t *testing.T) {
	t.Setenv("RUDDERSTACK_API_URL", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	var requests atomic.Int32
	httpClient := workspaceHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		assert.Equal(t, client.BASE_URL+"/v2/workspace", r.URL.String())
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"workspace":{"id":"ws-eu","region":"EU"}}`))}, nil
	})

	c, err := newDeps(t.Context(), "test-token", true, client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	assert.Equal(t, int32(1), requests.Load())
	assert.Equal(t, &client.Workspace{ID: "ws-eu", Region: "EU"}, c.Workspace())
	assert.Equal(t, client.BASE_URL_EU, c.Client().URL(""))
}

func TestNewDepsDiscoveryUsesCallerContext(t *testing.T) {
	t.Setenv("RUDDERSTACK_API_URL", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	httpClient := workspaceHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})

	d, err := newDeps(ctx, "test-token", true, client.WithHTTPClient(httpClient))
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, d)
}

func TestNewDepsDiscoveryHasDeadline(t *testing.T) {
	t.Setenv("RUDDERSTACK_API_URL", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	httpClient := workspaceHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(10*time.Second), deadline, time.Second)
		return nil, context.DeadlineExceeded
	})

	d, err := newDeps(t.Context(), "test-token", true, client.WithHTTPClient(httpClient))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, d)
}

func TestNewDepsKeepsExplicitAPIURL(t *testing.T) {
	t.Setenv("RUDDERSTACK_API_URL", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	const apiURL = "https://api.custom.test"
	// A direct workspace request through the client must still use the explicit URL.
	testutils.SetConfig(t, "apiURL", apiURL)

	var requests atomic.Int32
	httpClient := workspaceHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		assert.Equal(t, apiURL+"/v2/workspace", r.URL.String())
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"workspace":{"id":"ws-eu","region":"EU"}}`))}, nil
	})

	d, err := newDeps(t.Context(), "test-token", true, client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	assert.Nil(t, d.Workspace())
	assert.Equal(t, int32(0), requests.Load())
	workspace, err := d.Client().Workspaces.GetByAuthToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "EU", workspace.Region)
	assert.Equal(t, int32(1), requests.Load())
	assert.Equal(t, apiURL, d.Client().URL(""))
}

func TestNewDepsSkipsWorkspaceLookupForExplicitAPIURL(t *testing.T) {
	t.Setenv("RUDDERSTACK_API_URL", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	testutils.SetConfig(t, "apiURL", "https://api.custom.test")

	var requests atomic.Int32
	httpClient := workspaceHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, fmt.Errorf("unexpected workspace request")
	})

	d, err := newDeps(t.Context(), "test-token", true, client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	assert.Nil(t, d.Workspace())
	assert.Equal(t, "https://api.custom.test", d.Client().URL(""))
	assert.Equal(t, int32(0), requests.Load())
}

func TestNewDepsRejectsMissingWorkspace(t *testing.T) {
	t.Setenv("RUDDERSTACK_API_URL", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	httpClient := workspaceHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})

	d, err := newDeps(t.Context(), "test-token", true, client.WithHTTPClient(httpClient))
	require.ErrorContains(t, err, "fetching workspace information: workspace is missing")
	assert.Nil(t, d)
}

func TestNewDepsPropagatesWorkspaceLookupFailure(t *testing.T) {
	t.Setenv("RUDDERSTACK_API_URL", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	var requests atomic.Int32
	httpClient := workspaceHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"message":"forbidden"}`))}, nil
	})

	d, err := newDeps(t.Context(), "test-token", true, client.WithHTTPClient(httpClient))
	require.ErrorContains(t, err, "fetching workspace information: http status code: 403")
	assert.Nil(t, d)
	assert.Equal(t, int32(1), requests.Load())
}

func TestNewOfflineDepsSkipsWorkspaceRequest(t *testing.T) {
	t.Setenv("RUDDERSTACK_API_URL", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	var requests atomic.Int32
	httpClient := workspaceHTTPClientFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, fmt.Errorf("unexpected workspace request")
	})

	d, err := newDeps(t.Context(), "offline", false, client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	assert.Nil(t, d.Workspace())
	assert.Equal(t, int32(0), requests.Load())
}

func TestNewOfflineDepsNeedsNoToken(t *testing.T) {
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	_, err := NewDeps()
	require.ErrorContains(t, err, "access token is required")

	_, err = NewOfflineDeps()
	require.NoError(t, err)
}
