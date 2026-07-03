package destination_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerImpl_ExtractResourcesFromSpec(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, definitions.NewRegistry())

	extracted, err := h.Impl.ExtractResourcesFromSpec("destinations/ga4.yaml", &destination.DestinationSpec{
		ID:                "ga4-production",
		DisplayName:       "Production GA4",
		Type:              "GA4",
		Enabled:           true,
		DefinitionVersion: 1,
		Transformation:    "#transformation:my-transform",
		Config: map[string]any{
			"api_secret": "secret",
		},
	})
	require.NoError(t, err)

	resource := extracted["ga4-production"]
	require.NotNil(t, resource)
	assert.Equal(t, "ga4-production", resource.ID)
	assert.Equal(t, "Production GA4", resource.DisplayName)
	assert.Equal(t, "GA4", resource.Type)
	assert.True(t, resource.Enabled)
	assert.Equal(t, int64(1), resource.DefinitionVersion)
	assert.Equal(t, map[string]any{"api_secret": "secret"}, resource.Config)
	require.NotNil(t, resource.Transformation)
	assert.Equal(t, resources.URN("my-transform", types.TransformationResourceType), resource.Transformation.URN)
	assert.Equal(t, "id", resource.Transformation.Property)
}

func TestHandlerImpl_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var createBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/destinations", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		createBody = string(body)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"destination": {
				"id": "dst-123",
				"name": "Production Webhook",
				"type": "WEBHOOK",
				"enabled": true,
				"externalId": "webhook-production",
				"config": {"webhookUrl":"https://example.com"}
			}
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)
	state, err := h.Impl.Create(ctx, &destination.DestinationResource{
		ID:                "webhook-production",
		DisplayName:       "Production Webhook",
		Type:              "WEBHOOK",
		Enabled:           true,
		DefinitionVersion: 1,
		Config: map[string]any{
			"webhook_url": "https://example.com",
		},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"name": "Production Webhook",
		"type": "WEBHOOK",
		"enabled": true,
		"externalId": "webhook-production",
		"config": {"webhookUrl":"https://example.com"}
	}`, createBody)
	assert.Equal(t, &destination.DestinationState{
		ID:               "dst-123",
		TransformationID: "",
	}, state)
}

func TestHandlerImpl_CreateConnectsTransformation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/destinations":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-123","type":"WEBHOOK","enabled":true}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-123/transformation":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{"transformationId":"trans-456"}`, string(body))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"destinationId":"dst-123",
				"transformationId":"trans-456",
				"createdAt":"2020-01-01T01:01:01Z",
				"updatedAt":"2020-01-02T01:01:01Z"
			}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)
	state, err := h.Impl.Create(ctx, &destination.DestinationResource{
		ID:                "webhook-production",
		DisplayName:       "Production Webhook",
		Type:              "WEBHOOK",
		Enabled:           true,
		DefinitionVersion: 1,
		Transformation: &resources.PropertyRef{
			URN:        resources.URN("my-transform", types.TransformationResourceType),
			Property:   "id",
			IsResolved: true,
			Value:      "trans-456",
		},
		Config: map[string]any{
			"webhook_url": "https://example.com",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, &destination.DestinationState{
		ID:               "dst-123",
		TransformationID: "trans-456",
	}, state)
	assert.Equal(t, []string{
		"POST /v2/destinations",
		"PUT /v2/destinations/dst-123/transformation",
	}, calls)
}

func TestHandlerImpl_UpdateRejectsTypeChange(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))
	_, err := h.Impl.Update(context.Background(),
		&destination.DestinationResource{Type: "GA4"},
		&destination.DestinationResource{Type: "WEBHOOK"},
		&destination.DestinationState{ID: "dst-1"},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "immutable")
}

func TestHandlerImpl_UpdateDisconnectsTransformation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-123":
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-123","type":"WEBHOOK","enabled":true}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v2/destinations/dst-123/transformation":
			_, _ = w.Write([]byte(`{
				"destinationId":"dst-123",
				"transformationId":"trans-old",
				"createdAt":"2020-01-01T01:01:01Z",
				"updatedAt":"2020-01-02T01:01:01Z"
			}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)
	state, err := h.Impl.Update(ctx,
		&destination.DestinationResource{
			DisplayName:       "Production Webhook",
			Type:              "WEBHOOK",
			Enabled:           true,
			DefinitionVersion: 1,
			Config: map[string]any{
				"webhook_url": "https://example.com/updated",
			},
		},
		&destination.DestinationResource{
			DisplayName:       "Production Webhook",
			Type:              "WEBHOOK",
			Enabled:           true,
			DefinitionVersion: 1,
			Transformation: &resources.PropertyRef{
				URN:        resources.URN("my-transform", types.TransformationResourceType),
				Property:   "id",
				IsResolved: true,
				Value:      "trans-old",
			},
			Config: map[string]any{
				"webhook_url": "https://example.com",
			},
		},
		&destination.DestinationState{
			ID:               "dst-123",
			TransformationID: "trans-old",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, &destination.DestinationState{
		ID:               "dst-123",
		TransformationID: "",
	}, state)
	assert.Equal(t, []string{
		"PUT /v2/destinations/dst-123",
		"DELETE /v2/destinations/dst-123/transformation",
	}, calls)
}

func TestHandlerImpl_DeleteDisconnectsTransformationFirst(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/transformation") {
			_, _ = w.Write([]byte(`{
				"destinationId":"dst-123",
				"transformationId":"trans-456",
				"createdAt":"2020-01-01T01:01:01Z",
				"updatedAt":"2020-01-02T01:01:01Z"
			}`))
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, testRegistry(t))
	err := h.Impl.Delete(ctx, "webhook-production", &destination.DestinationResource{}, &destination.DestinationState{
		ID:               "dst-123",
		TransformationID: "trans-456",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"DELETE /v2/destinations/dst-123/transformation",
		"DELETE /v2/destinations/dst-123",
	}, calls)
}

func TestHandlerImpl_MapRemoteToState(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	h := destination.NewHandler(nil, registry)

	collection := resources.NewRemoteResources()
	collection.Set(types.TransformationResourceType, map[string]*resources.RemoteResource{
		"trans-456": {
			ID:         "trans-456",
			ExternalID: "my-transform",
			Data:       struct{}{},
		},
	})

	resource, state, err := h.Impl.MapRemoteToState(&destination.RemoteDestination{
		Destination: &client.Destination{
			ID:         "dst-123",
			ExternalID: "webhook-production",
			Name:       "Production Webhook",
			Type:       "WEBHOOK",
			IsEnabled:  true,
			Config:     json.RawMessage(`{"webhookUrl":"https://example.com"}`),
			Transformation: &client.DestinationTransformationLink{
				ID: "trans-456",
			},
		},
	}, collection)
	require.NoError(t, err)
	require.NotNil(t, resource)
	require.NotNil(t, state)

	assert.Equal(t, &destination.DestinationResource{
		ID:                "webhook-production",
		DisplayName:       "Production Webhook",
		Type:              "WEBHOOK",
		Enabled:           true,
		DefinitionVersion: 1,
		Config: map[string]any{
			"webhook_url": "https://example.com",
		},
	}, &destination.DestinationResource{
		ID:                resource.ID,
		DisplayName:       resource.DisplayName,
		Type:              resource.Type,
		Enabled:           resource.Enabled,
		DefinitionVersion: resource.DefinitionVersion,
		Config:            resource.Config,
	})
	require.NotNil(t, resource.Transformation)
	assert.Equal(t, resources.URN("my-transform", types.TransformationResourceType), resource.Transformation.URN)
	assert.Equal(t, "id", resource.Transformation.Property)
	assert.Equal(t, &destination.DestinationState{
		ID:               "dst-123",
		TransformationID: "trans-456",
	}, state)
}

func TestHandlerImpl_MapRemoteToStateUnregisteredTypeErrors(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))
	_, _, err := h.Impl.MapRemoteToState(&destination.RemoteDestination{
		Destination: &client.Destination{
			ID:         "dst-123",
			ExternalID: "unknown-production",
			Type:       "UNKNOWN",
			Config:     json.RawMessage(`{}`),
		},
	}, resources.NewRemoteResources())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unregistered type")
}

func TestHandlerImpl_LoadRemoteResources(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	t.Run("returns managed destinations", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/destinations", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"destinations": [
					{
						"id": "dst-managed",
						"externalId": "webhook-production",
						"name": "Managed",
						"type": "WEBHOOK",
						"enabled": true,
						"config": {"webhookUrl":"https://example.com"}
					},
					{
						"id": "dst-unmanaged",
						"name": "Unmanaged",
						"type": "WEBHOOK",
						"enabled": true,
						"config": {"webhookUrl":"https://example.com"}
					}
				],
				"paging": {"total": 2}
			}`))
		}))
		t.Cleanup(srv.Close)

		h := destination.NewHandler(newTestClient(t, srv.URL), registry)
		remotes, err := h.Impl.LoadRemoteResources(ctx)
		require.NoError(t, err)
		require.Len(t, remotes, 1)
		assert.Equal(t, "dst-managed", remotes[0].ID)
		assert.Equal(t, "webhook-production", remotes[0].ExternalID)
	})

	t.Run("errors on unregistered managed type", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"destinations": [{
					"id": "dst-unknown",
					"externalId": "unknown-production",
					"name": "Unknown",
					"type": "UNKNOWN",
					"enabled": true,
					"config": {}
				}],
				"paging": {"total": 1}
			}`))
		}))
		t.Cleanup(srv.Close)

		h := destination.NewHandler(newTestClient(t, srv.URL), registry)
		_, err := h.Impl.LoadRemoteResources(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unregistered type")
	})
}

func TestHandlerImpl_LoadImportableResourcesFiltersUnregisteredTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"destinations": [
				{
					"id": "dst-importable",
					"name": "Importable",
					"type": "WEBHOOK",
					"enabled": true,
					"config": {"webhookUrl":"https://example.com"}
				},
				{
					"id": "dst-unknown",
					"name": "Unknown",
					"type": "UNKNOWN",
					"enabled": true,
					"config": {}
				}
			],
			"paging": {"total": 2}
		}`))
	}))
	t.Cleanup(srv.Close)

	h := destination.NewHandler(newTestClient(t, srv.URL), registry)
	remotes, err := h.Impl.LoadImportableResources(ctx)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	assert.Equal(t, "dst-importable", remotes[0].ID)
}

func TestHandlerImpl_ImportAndExportNotImplemented(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))

	_, err := h.Impl.Import(context.Background(), &destination.DestinationResource{}, "remote-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "import not implemented")

	_, err = h.Impl.FormatForExport(nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "export not implemented")
}

func testRegistry(t *testing.T) *definitions.Registry {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhookTestDefinition()))
	require.NoError(t, registry.Register(ga4TestDefinition()))
	return registry
}

func webhookTestDefinition() *definitions.DestinationDefinition {
	return &definitions.DestinationDefinition{
		Type:    "WEBHOOK",
		Version: 1,
		Properties: []converter.ConfigProperty{
			converter.Simple("webhookUrl", "webhook_url"),
		},
		NewConfig: func() any {
			return &struct {
				WebhookURL string `mapstructure:"webhook_url" validate:"required"`
			}{}
		},
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
	}
}

func ga4TestDefinition() *definitions.DestinationDefinition {
	return &definitions.DestinationDefinition{
		Type:    "GA4",
		Version: 1,
		Properties: []converter.ConfigProperty{
			converter.Simple("apiSecret", "api_secret"),
			converter.Simple("measurementId", "measurement_id"),
		},
		SecretKeys: []string{"api_secret"},
		NewConfig: func() any {
			return &struct {
				APISecret     string `mapstructure:"api_secret" validate:"required"`
				MeasurementID string `mapstructure:"measurement_id"`
			}{}
		},
		SourceTypes: []string{"web", "android"},
		ConnectionModes: map[string][]string{
			"web":     {"cloud", "device", "hybrid"},
			"android": {"cloud", "device"},
		},
	}
}

func newTestClient(t *testing.T, baseURL string) *client.Client {
	t.Helper()

	c, err := client.New("token", client.WithBaseURL(baseURL))
	require.NoError(t, err)
	return c
}
