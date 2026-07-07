package destination_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test helpers ---

// newTestClient builds a *client.Client pointed at the httptest server.
func newTestClient(t *testing.T, baseURL string) *client.Client {
	t.Helper()
	c, err := client.New("test-token", client.WithBaseURL(baseURL))
	require.NoError(t, err)
	return c
}

// testRegistry builds a registry with a webhook and a GA4 definition, mirroring
// the shapes in definitions/export_test.go (which are only visible inside the
// definitions package's own test binary).
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

// resolvedRef builds a resolved PropertyRef simulating what the apply framework
// produces before calling Update.
func resolvedRef(urn, value string) *resources.PropertyRef {
	return &resources.PropertyRef{
		URN:        urn,
		Property:   "id",
		IsResolved: true,
		Value:      value,
	}
}

// urnResolver is a minimal URNResolver for MapRemoteToState tests.
type urnResolver struct {
	transformationURNByID map[string]string // remote ID -> URN
}

func (r urnResolver) GetURNByID(resourceType string, remoteID string) (string, error) {
	if resourceType == ttypes.TransformationResourceType {
		if urn, ok := r.transformationURNByID[remoteID]; ok {
			return urn, nil
		}
		return "", resources.ErrRemoteResourceExternalIdNotFound
	}
	return "", resources.ErrRemoteResourceExternalIdNotFound
}

// --- ExtractResourcesFromSpec ---

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
	assert.Equal(t, resources.URN("my-transform", ttypes.TransformationResourceType), resource.Transformation.URN)
	assert.Equal(t, "id", resource.Transformation.Property)
}

func TestHandlerImpl_ExtractResourcesFromSpecNoTransformation(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, definitions.NewRegistry())

	extracted, err := h.Impl.ExtractResourcesFromSpec("destinations/webhook.yaml", &destination.DestinationSpec{
		ID:                "webhook-1",
		DisplayName:       "Webhook One",
		Type:              "WEBHOOK",
		Enabled:           false,
		DefinitionVersion: 1,
		Config:            map[string]any{"webhook_url": "https://example.com/hook"},
	})
	require.NoError(t, err)

	resource := extracted["webhook-1"]
	require.NotNil(t, resource)
	assert.Nil(t, resource.Transformation)
}

func TestHandlerImpl_ExtractResourcesFromSpecInvalidTransformationRef(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, definitions.NewRegistry())

	_, err := h.Impl.ExtractResourcesFromSpec("destinations/bad.yaml", &destination.DestinationSpec{
		ID:             "bad",
		DisplayName:    "Bad",
		Type:           "GA4",
		Transformation: "#source:my-source", // wrong kind
		Config:         map[string]any{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transformation reference")
}

// --- Create ---

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
				"id": "dst-1",
				"externalId": "ga4-production",
				"name": "Production GA4",
				"type": "GA4",
				"enabled": true,
				"config": {"apiSecret":"secret-value","measurementId":"G-123"}
			}
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Create(ctx, &destination.DestinationResource{
		ID:                "ga4-production",
		DisplayName:       "Production GA4",
		Type:              "GA4",
		Enabled:           true,
		DefinitionVersion: 1,
		Config: map[string]any{
			"api_secret":     "secret-value",
			"measurement_id": "G-123",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: ""}, state)

	// Verify the request body carried camelCase config and the external ID.
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(createBody), &payload))
	assert.Equal(t, "Production GA4", payload["name"])
	assert.Equal(t, "GA4", payload["type"])
	assert.Equal(t, true, payload["enabled"])
	assert.Equal(t, "ga4-production", payload["externalId"])
	config, _ := payload["config"].(map[string]any)
	assert.Equal(t, "secret-value", config["apiSecret"])
	assert.Equal(t, "G-123", config["measurementId"])
}

func TestHandlerImpl_CreateConnectsTransformation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var (
		createCalled       bool
		connectCalled      bool
		connectDestination string
		connectTransform   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/destinations":
			createCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-9","name":"x","type":"WEBHOOK","enabled":true,"config":{"webhookUrl":"https://h"}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-9/transformation":
			connectCalled = true
			connectDestination = "dst-9"
			body, _ := io.ReadAll(r.Body)
			var p map[string]any
			_ = json.Unmarshal(body, &p)
			connectTransform, _ = p["transformationId"].(string)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-9","transformationId":"trans-1"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Create(ctx, &destination.DestinationResource{
		ID:                "webhook-1",
		DisplayName:       "x",
		Type:              "WEBHOOK",
		Enabled:           true,
		DefinitionVersion: 1,
		Transformation:    resolvedRef(resources.URN("my-transform", ttypes.TransformationResourceType), "trans-1"),
		Config:            map[string]any{"webhook_url": "https://h"},
	})
	require.NoError(t, err)
	assert.True(t, createCalled)
	assert.True(t, connectCalled)
	assert.Equal(t, "dst-9", connectDestination)
	assert.Equal(t, "trans-1", connectTransform)
	assert.Equal(t, &destination.DestinationState{ID: "dst-9", TransformationID: "trans-1"}, state)
}

// --- Update ---

func TestHandlerImpl_UpdateRejectsTypeChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)
	c := newTestClient(t, "https://unused.example")
	h := destination.NewHandler(c, registry)

	_, err := h.Impl.Update(ctx,
		&destination.DestinationResource{ID: "x", Type: "GA4", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationResource{ID: "x", Type: "WEBHOOK", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationState{ID: "dst-1"},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type change is not supported")
}

func TestHandlerImpl_UpdateConfigChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var updateBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, "/v2/destinations/dst-1", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		updateBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","name":"renamed","type":"GA4","enabled":false,"config":{"apiSecret":"new-secret","measurementId":"G-999"}}}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Update(ctx,
		&destination.DestinationResource{
			ID: "ga4-production", DisplayName: "renamed", Type: "GA4", Enabled: false,
			DefinitionVersion: 1,
			Config:            map[string]any{"api_secret": "new-secret", "measurement_id": "G-999"},
		},
		&destination.DestinationResource{ID: "ga4-production", Type: "GA4", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationState{ID: "dst-1"},
	)
	require.NoError(t, err)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: ""}, state)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(updateBody), &payload))
	assert.Equal(t, "renamed", payload["name"])
	assert.Equal(t, false, payload["enabled"])
	config, _ := payload["config"].(map[string]any)
	assert.Equal(t, "new-secret", config["apiSecret"])
	assert.Equal(t, "G-999", config["measurementId"])
}

func TestHandlerImpl_UpdateConnectsTransformationWhenPreviouslyNone(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var (
		updateCalled  bool
		connectCalled bool
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			updateCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"GA4","enabled":true,"config":{}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/transformation":
			connectCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-7"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Update(ctx,
		&destination.DestinationResource{
			ID: "ga4", DisplayName: "GA4", Type: "GA4", Enabled: true, DefinitionVersion: 1,
			Transformation: resolvedRef(resources.URN("t", ttypes.TransformationResourceType), "trans-7"),
			Config:         map[string]any{},
		},
		&destination.DestinationResource{ID: "ga4", Type: "GA4", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationState{ID: "dst-1", TransformationID: ""},
	)
	require.NoError(t, err)
	assert.True(t, updateCalled)
	assert.True(t, connectCalled)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: "trans-7"}, state)
}

func TestHandlerImpl_UpdateReplacesTransformationLink(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var connectCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"GA4","enabled":true,"config":{}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/transformation":
			connectCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-8"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Update(ctx,
		&destination.DestinationResource{
			ID: "ga4", DisplayName: "GA4", Type: "GA4", Enabled: true, DefinitionVersion: 1,
			Transformation: resolvedRef(resources.URN("t", ttypes.TransformationResourceType), "trans-8"),
			Config:         map[string]any{},
		},
		&destination.DestinationResource{ID: "ga4", Type: "GA4", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationState{ID: "dst-1", TransformationID: "trans-old"},
	)
	require.NoError(t, err)
	assert.True(t, connectCalled)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: "trans-8"}, state)
}

func TestHandlerImpl_UpdateDisconnectsTransformationWhenRefRemoved(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var disconnectCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"GA4","enabled":true,"config":{}}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v2/destinations/dst-1/transformation":
			disconnectCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-old"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Update(ctx,
		&destination.DestinationResource{
			ID: "ga4", DisplayName: "GA4", Type: "GA4", Enabled: true, DefinitionVersion: 1,
			Transformation: nil,
			Config:         map[string]any{},
		},
		&destination.DestinationResource{ID: "ga4", Type: "GA4", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationState{ID: "dst-1", TransformationID: "trans-old"},
	)
	require.NoError(t, err)
	assert.True(t, disconnectCalled)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: ""}, state)
}

func TestHandlerImpl_UpdateNoLinkCallWhenUnchanged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"GA4","enabled":true,"config":{}}}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Update(ctx,
		&destination.DestinationResource{
			ID: "ga4", DisplayName: "GA4", Type: "GA4", Enabled: true, DefinitionVersion: 1,
			Transformation: resolvedRef(resources.URN("t", ttypes.TransformationResourceType), "trans-same"),
			Config:         map[string]any{},
		},
		&destination.DestinationResource{ID: "ga4", Type: "GA4", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationState{ID: "dst-1", TransformationID: "trans-same"},
	)
	require.NoError(t, err)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: "trans-same"}, state)
}

// --- Delete ---

func TestHandlerImpl_DeleteDisconnectsTransformationFirst(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var (
		disconnectCalled bool
		deleteCalled     bool
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/v2/destinations/dst-1/transformation":
			disconnectCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-old"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v2/destinations/dst-1":
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	err := h.Impl.Delete(ctx, "ga4",
		&destination.DestinationResource{ID: "ga4", Type: "GA4", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationState{ID: "dst-1", TransformationID: "trans-old"},
	)
	require.NoError(t, err)
	assert.True(t, disconnectCalled, "transformation should be disconnected before delete")
	assert.True(t, deleteCalled)
}

func TestHandlerImpl_DeleteWithoutTransformation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/v2/destinations/dst-1":
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	err := h.Impl.Delete(ctx, "ga4",
		&destination.DestinationResource{ID: "ga4", Type: "GA4", DefinitionVersion: 1, Config: map[string]any{}},
		&destination.DestinationState{ID: "dst-1", TransformationID: ""},
	)
	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

// --- MapRemoteToState ---

func TestHandlerImpl_MapRemoteToState(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	h := destination.NewHandler(nil, registry)

	resolver := urnResolver{transformationURNByID: map[string]string{
		"trans-1": resources.URN("my-transform", ttypes.TransformationResourceType),
	}}

	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:             "dst-1",
		ExternalID:     "ga4-production",
		Name:           "Production GA4",
		Type:           "GA4",
		Version:        1,
		IsEnabled:      true,
		Config:         []byte(`{"apiSecret":"secret-value","measurementId":"G-123"}`),
		Transformation: &client.DestinationTransformationLink{ID: "trans-1"},
	}}

	resource, state, err := h.Impl.MapRemoteToState(remote, resolver)
	require.NoError(t, err)
	require.NotNil(t, resource)
	require.NotNil(t, state)

	assert.Equal(t, &destination.DestinationResource{
		ID:                "ga4-production",
		DisplayName:       "Production GA4",
		Type:              "GA4",
		Enabled:           true,
		DefinitionVersion: 1,
		Transformation: &resources.PropertyRef{
			URN:      resources.URN("my-transform", ttypes.TransformationResourceType),
			Property: "id",
		},
		Config: map[string]any{
			"api_secret":     "secret-value",
			"measurement_id": "G-123",
		},
	}, resource)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: "trans-1"}, state)
}

func TestHandlerImpl_MapRemoteToStateNoTransformation(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	h := destination.NewHandler(nil, registry)

	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:         "dst-2",
		ExternalID: "webhook-1",
		Name:       "Hook",
		Type:       "WEBHOOK",
		Version:    1,
		IsEnabled:  true,
		Config:     []byte(`{"webhookUrl":"https://h"}`),
	}}

	resource, state, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.NoError(t, err)
	require.NotNil(t, resource)
	assert.Nil(t, resource.Transformation)
	assert.Equal(t, "", state.TransformationID)
}

func TestHandlerImpl_MapRemoteToStateTransformationNotCLIManaged(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	h := destination.NewHandler(nil, registry)

	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:             "dst-3",
		ExternalID:     "ga4-2",
		Name:           "GA4",
		Type:           "GA4",
		Version:        1,
		Config:         []byte(`{"apiSecret":"s"}`),
		Transformation: &client.DestinationTransformationLink{ID: "ui-trans"},
	}}

	resource, state, err := h.Impl.MapRemoteToState(remote, urnResolver{}) // no URN for "ui-trans"
	require.NoError(t, err)
	// Foreign link is dropped entirely: spec ref nil and ID empty, so a later
	// unrelated Update never disconnects the user's UI-managed transformation.
	assert.Nil(t, resource.Transformation)
	assert.Equal(t, "", state.TransformationID)
}

func TestHandlerImpl_MapRemoteToStateUnregisteredTypeErrors(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	h := destination.NewHandler(nil, registry)

	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:         "dst-x",
		ExternalID: "s3-1",
		Name:       "S3",
		Type:       "S3", // not registered
		Config:     []byte(`{}`),
	}}

	_, _, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unregistered type")
}

func TestHandlerImpl_MapRemoteToStateEmptyExternalIDErrors(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	h := destination.NewHandler(nil, registry)

	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:     "dst-noext",
		Name:   "NoExt",
		Type:   "GA4",
		Config: []byte(`{}`),
	}}

	resource, state, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty external ID")
	assert.Nil(t, resource)
	assert.Nil(t, state)
}

// --- LoadRemoteResources ---

func TestHandlerImpl_LoadRemoteResources(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/destinations", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"destinations": [
				{"id":"dst-1","externalId":"ga4-prod","name":"GA4","type":"GA4","config":{}},
				{"id":"dst-2","name":"Unmanaged","type":"GA4","config":{}}
			],
			"paging": {"total": 2}
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	remotes, err := h.Impl.LoadRemoteResources(ctx)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	assert.Equal(t, "dst-1", remotes[0].ID)
	assert.Equal(t, "ga4-prod", remotes[0].ExternalID)
}

func TestHandlerImpl_LoadRemoteResourcesErrorsOnUnregisteredManagedType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"destinations": [
				{"id":"dst-1","externalId":"s3-1","name":"S3","type":"S3","config":{}}
			],
			"paging": {"total": 1}
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	_, err := h.Impl.LoadRemoteResources(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unregistered type")
}

// --- LoadImportableResources ---

func TestHandlerImpl_LoadImportableResourcesFiltersUnregisteredTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"destinations": [
				{"id":"dst-1","name":"GA4-unmanaged","type":"GA4","version":1,"config":{}},
				{"id":"dst-2","externalId":"ga4-managed","name":"GA4-managed","type":"GA4","version":1,"config":{}},
				{"id":"dst-3","name":"S3-unmanaged","type":"S3","config":{}},
				{"id":"dst-4","name":"GA4-unregistered-version","type":"GA4","version":2,"config":{}}
			],
			"paging": {"total": 4}
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	remotes, err := h.Impl.LoadImportableResources(ctx)
	require.NoError(t, err)
	require.Len(t, remotes, 1, "only unmanaged + registered (type, version) pairs pass")
	assert.Equal(t, "dst-1", remotes[0].ID)
	assert.Equal(t, "", remotes[0].ExternalID)
}

// --- Import ---

// callTracker records the order of API calls a mock server receives, so tests
// can assert on ordering invariants (e.g. SetExternalID must run after Update).
type callTracker struct {
	calls []string
}

func (c *callTracker) record(name string) {
	c.calls = append(c.calls, name)
}

func TestHandlerImpl_Import_ReplacesLinkAndSetsExternalIDAfterUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)
	tracker := &callTracker{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1":
			tracker.record("get")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","name":"GA4","type":"GA4","enabled":true,"config":{"apiSecret":"old","measurementId":"G-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1/transformation":
			tracker.record("get-transformation")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-old"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			tracker.record("update")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"GA4","enabled":true,"config":{}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/transformation":
			tracker.record("connect-transformation")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-new"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/external-id":
			tracker.record("set-external-id")
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Import(ctx, &destination.DestinationResource{
		ID: "ga4-production", DisplayName: "Production GA4", Type: "GA4", Enabled: true, DefinitionVersion: 1,
		Transformation: resolvedRef(resources.URN("t", ttypes.TransformationResourceType), "trans-new"),
		Config:         map[string]any{"api_secret": "new-secret", "measurement_id": "G-2"},
	}, "dst-1")
	require.NoError(t, err)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: "trans-new"}, state)
	assert.Equal(t,
		[]string{"get", "get-transformation", "update", "connect-transformation", "set-external-id"},
		tracker.calls,
		"SetExternalID must run after Update reconciles config and the transformation link",
	)
}

func TestHandlerImpl_Import_NoExistingLinkTreatsGetTransformationErrorAsNoLink(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var connectCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","name":"WH","type":"WEBHOOK","enabled":true,"config":{"webhookUrl":"https://h"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1/transformation":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"WEBHOOK","enabled":true,"config":{}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/transformation":
			connectCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-new"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/external-id":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Import(ctx, &destination.DestinationResource{
		ID: "webhook-1", DisplayName: "WH", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
		Transformation: resolvedRef(resources.URN("t", ttypes.TransformationResourceType), "trans-new"),
		Config:         map[string]any{"webhook_url": "https://h"},
	}, "dst-1")
	require.NoError(t, err)
	assert.True(t, connectCalled, "a no-link GetTransformation response must still allow connecting a desired link")
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: "trans-new"}, state)
}

func TestHandlerImpl_Import_DisconnectsTransformationWhenSpecHasNone(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var disconnectCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","name":"WH","type":"WEBHOOK","enabled":true,"config":{"webhookUrl":"https://h"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1/transformation":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-old"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"WEBHOOK","enabled":true,"config":{}}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v2/destinations/dst-1/transformation":
			disconnectCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-old"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/external-id":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Import(ctx, &destination.DestinationResource{
		ID: "webhook-1", DisplayName: "WH", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
		Config: map[string]any{"webhook_url": "https://h"},
	}, "dst-1")
	require.NoError(t, err)
	assert.True(t, disconnectCalled)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: ""}, state)
}

func TestHandlerImpl_Import_NoLinkChangeSkipsTransformationCall(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","name":"WH","type":"WEBHOOK","enabled":true,"config":{"webhookUrl":"https://h"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1/transformation":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"trans-same"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"WEBHOOK","enabled":true,"config":{}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/external-id":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s (transformation link should be a no-op)", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	state, err := h.Impl.Import(ctx, &destination.DestinationResource{
		ID: "webhook-1", DisplayName: "WH", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
		Transformation: resolvedRef(resources.URN("t", ttypes.TransformationResourceType), "trans-same"),
		Config:         map[string]any{"webhook_url": "https://h"},
	}, "dst-1")
	require.NoError(t, err)
	assert.Equal(t, &destination.DestinationState{ID: "dst-1", TransformationID: "trans-same"}, state)
}

func TestHandlerImpl_Import_GetErrorPropagates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	_, err := h.Impl.Import(ctx, &destination.DestinationResource{ID: "x", Type: "WEBHOOK", DefinitionVersion: 1, Config: map[string]any{}}, "dst-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting destination during import")
}

func TestHandlerImpl_Import_UpdateErrorSkipsSetExternalID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var setExternalIDCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","name":"WH","type":"WEBHOOK","enabled":true,"config":{"webhookUrl":"https://h"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1/transformation":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/external-id":
			setExternalIDCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	_, err := h.Impl.Import(ctx, &destination.DestinationResource{
		ID: "webhook-1", Type: "WEBHOOK", DefinitionVersion: 1, Config: map[string]any{"webhook_url": "https://h"},
	}, "dst-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updating destination during import")
	assert.False(t, setExternalIDCalled, "external ID must not be set when Update fails")
}

// --- FormatForExport ---

// stubResolver is a minimal resolver.ReferenceResolver for FormatForExport tests.
type stubResolver struct {
	fn func(entityType, remoteID string) (string, error)
}

func (r stubResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	return r.fn(entityType, remoteID)
}

func TestHandlerImpl_FormatForExport_EmptyCollection(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))

	entities, entries, err := h.Impl.FormatForExport(map[string]*destination.RemoteDestination{}, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, entities)
	assert.Nil(t, entries)
}

func TestHandlerImpl_FormatForExport_BasicSpecWithoutSecretsOrTransformation(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))

	collection := map[string]*destination.RemoteDestination{
		"webhook-1": {Destination: &client.Destination{
			ID:          "dst-1",
			WorkspaceID: "ws-1",
			Name:        "My Webhook",
			Type:        "WEBHOOK",
			Version:     1,
			IsEnabled:   true,
			Config:      []byte(`{"webhookUrl":"https://example.com/hook"}`),
		}},
	}

	entities, entries, err := h.Impl.FormatForExport(collection, nil, stubResolver{})
	require.NoError(t, err)
	require.Len(t, entities, 1)
	require.Len(t, entries, 1)

	assert.Equal(t, importmanifest.ImportEntry{
		WorkspaceID: "ws-1",
		URN:         resources.URN("webhook-1", destination.DestinationResourceType),
		RemoteID:    "dst-1",
	}, entries[0])

	assert.Equal(t, filepath.Join("destinations", "webhook-1.yaml"), entities[0].RelativePath)

	spec, ok := entities[0].Content.(*specs.Spec)
	require.True(t, ok)
	assert.Equal(t, specs.SpecVersionV1, spec.Version)
	assert.Equal(t, destination.DestinationSpecKind, spec.Kind)
	assert.Equal(t, map[string]any{
		"id":                 "webhook-1",
		"display_name":       "My Webhook",
		"type":               "WEBHOOK",
		"enabled":            true,
		"definition_version": int64(1),
		"config": map[string]any{
			"webhook_url": "https://example.com/hook",
		},
	}, spec.Spec)
	assert.Equal(t, destination.DestinationMetadataName, spec.Metadata["name"])
}

func TestHandlerImpl_FormatForExport_MasksSecretKeysWithExternalIDPrefix(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))

	collection := map[string]*destination.RemoteDestination{
		"ga4-production": {Destination: &client.Destination{
			ID:        "dst-2",
			Name:      "GA4",
			Type:      "GA4",
			Version:   1,
			IsEnabled: true,
			Config:    []byte(`{"apiSecret":"super-secret","measurementId":"G-123"}`),
		}},
	}

	entities, _, err := h.Impl.FormatForExport(collection, nil, stubResolver{})
	require.NoError(t, err)
	require.Len(t, entities, 1)

	spec, ok := entities[0].Content.(*specs.Spec)
	require.True(t, ok)
	config, ok := spec.Spec["config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "{{ .GA4_PRODUCTION_API_SECRET }}", config["api_secret"], "hyphens in the external ID become underscores")
	assert.Equal(t, "G-123", config["measurement_id"], "non-secret keys are left untouched")
}

func TestHandlerImpl_FormatForExport_ResolvesTransformationReference(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))

	collection := map[string]*destination.RemoteDestination{
		"ga4-production": {Destination: &client.Destination{
			ID:             "dst-2",
			Name:           "GA4",
			Type:           "GA4",
			Version:        1,
			Config:         []byte(`{"apiSecret":"s","measurementId":"G-1"}`),
			Transformation: &client.DestinationTransformationLink{ID: "trans-1"},
		}},
	}

	resolver := stubResolver{fn: func(entityType, remoteID string) (string, error) {
		assert.Equal(t, ttypes.TransformationResourceType, entityType)
		assert.Equal(t, "trans-1", remoteID)
		return "#transformation:my-transform", nil
	}}

	entities, _, err := h.Impl.FormatForExport(collection, nil, resolver)
	require.NoError(t, err)
	require.Len(t, entities, 1)

	spec, ok := entities[0].Content.(*specs.Spec)
	require.True(t, ok)
	assert.Equal(t, "#transformation:my-transform", spec.Spec["transformation"])
}

func TestHandlerImpl_FormatForExport_TransformationResolutionErrorFailsExport(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))

	collection := map[string]*destination.RemoteDestination{
		"ga4-production": {Destination: &client.Destination{
			ID:             "dst-2",
			Name:           "GA4",
			Type:           "GA4",
			Version:        1,
			Config:         []byte(`{"apiSecret":"s","measurementId":"G-1"}`),
			Transformation: &client.DestinationTransformationLink{ID: "trans-foreign"},
		}},
	}

	resolver := stubResolver{fn: func(string, string) (string, error) {
		return "", fmt.Errorf("resource not present in resources collection")
	}}

	entities, entries, err := h.Impl.FormatForExport(collection, nil, resolver)
	require.Error(t, err, "an unresolved transformation link must fail the export, not fall back to a raw ID")
	assert.Nil(t, entities)
	assert.Nil(t, entries)
}

func TestHandlerImpl_FormatForExport_UnregisteredTypeVersionErrors(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))

	collection := map[string]*destination.RemoteDestination{
		"s3-1": {Destination: &client.Destination{
			ID:     "dst-3",
			Name:   "S3",
			Type:   "S3",
			Config: []byte(`{}`),
		}},
	}

	_, _, err := h.Impl.FormatForExport(collection, nil, stubResolver{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting destination definition")
}
