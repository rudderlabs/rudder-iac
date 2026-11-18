package destination

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importreport"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dashboardHTTPConfig is an HTTP destination as the webapp stores it: modelled
// keys, empty values it persists for cleared fields, and two parts the
// definition cannot hold.
const dashboardHTTPConfig = `{
	"apiUrl": "https://example.com/hook",
	"auth": "bearerTokenAuth",
	"method": "POST",
	"format": "JSON",
	"headers": [{"from": "X-Env", "to": "prod", "description": "kept by nobody"}],
	"queryParams": [],
	"consentManagement": {},
	"ketchConsentPurposes": {},
	"oneTrustCookieCategories": {},
	"someLegacyBlock": {"a": 1, "b": 2}
}`

func httpRegistry(t *testing.T) *definitions.Registry {
	t.Helper()
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))
	return registry
}

func TestUnmodelledConfig(t *testing.T) {
	t.Parallel()

	registered, err := httpRegistry(t).GetByAPIType("HTTP", 1)
	require.NoError(t, err)

	t.Run("lists parts local config cannot hold, by path and shape", func(t *testing.T) {
		got, err := unmodelledConfig(registered, json.RawMessage(dashboardHTTPConfig))
		require.NoError(t, err)

		assert.Equal(t, []importreport.Lost{
			{Path: "headers[0].description", Value: "(value)"},
			{Path: "someLegacyBlock", Value: "(object, 2 keys)"},
		}, got)
	})

	t.Run("never lists a value", func(t *testing.T) {
		got, err := unmodelledConfig(registered, json.RawMessage(`{"apiUrl":"https://example.com","auth":"noAuth","secretLookingKey":"hunter2"}`))
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.NotContains(t, got[0].Value, "hunter2")
	})

	t.Run("a config with only modelled keys and empty values loses nothing", func(t *testing.T) {
		got, err := unmodelledConfig(registered, json.RawMessage(`{"apiUrl":"https://example.com","auth":"noAuth","headers":[],"queryParams":[],"ketchConsentPurposes":{}}`))
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestProvider_ImportReport(t *testing.T) {
	t.Parallel()

	p := NewProvider(nil, httpRegistry(t))
	collection := resources.NewRemoteResources()
	collection.Set(DestinationResourceType, map[string]*resources.RemoteResource{
		"d1": {ID: "d1", Data: &RemoteDestination{Destination: &client.Destination{
			ID: "d1", Name: "Lossy", Type: "HTTP", Version: 1, Config: json.RawMessage(dashboardHTTPConfig),
		}}},
		"d2": {ID: "d2", Data: &RemoteDestination{Destination: &client.Destination{
			ID: "d2", Name: "Clean", Type: "HTTP", Version: 1, Config: json.RawMessage(`{"apiUrl":"https://example.com","auth":"noAuth"}`),
		}}},
	})

	got, err := p.ImportReport(collection, []importmanifest.ImportEntry{
		{URN: "destination:lossy", RemoteID: "d1"},
		{URN: "destination:clean", RemoteID: "d2"},
	})
	require.NoError(t, err)

	assert.Equal(t, []importreport.Entry{{
		URN:      "destination:lossy",
		RemoteID: "d1",
		Name:     "Lossy",
		Lost: []importreport.Lost{
			{Path: "headers[0].description", Value: "(value)"},
			{Path: "someLegacyBlock", Value: "(object, 2 keys)"},
		},
	}}, got)
}

func TestProvider_CheckImports(t *testing.T) {
	t.Parallel()

	var fetched []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/v2/destinations/")
		fetched = append(fetched, id)
		config := dashboardHTTPConfig
		if id == "clean" {
			config = `{"apiUrl":"https://example.com","auth":"noAuth"}`
		}
		_, _ = w.Write([]byte(`{"destination":{"id":"` + id + `","name":"n","type":"HTTP","version":1,"enabled":true,"config":` + config + `}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := client.New("test-token", client.WithBaseURL(srv.URL))
	require.NoError(t, err)
	p := NewProvider(c, httpRegistry(t))

	importOp := func(id string) *planner.Operation {
		return &planner.Operation{
			Type: planner.Import,
			Resource: resources.NewResource(id, DestinationResourceType, resources.ResourceData{}, nil,
				resources.WithRawData(&DestinationResource{ID: id, Type: "http"}),
				resources.WithResourceImportMetadata(id, "ws-1")),
		}
	}

	lost := []importreport.Lost{
		{Path: "headers[0].description", Value: "(value)"},
		{Path: "someLegacyBlock", Value: "(object, 2 keys)"},
	}

	tests := []struct {
		name        string
		report      *importreport.Report
		importID    string
		wantRefused []string
	}{
		{
			name:        "no report refuses and names every path",
			report:      &importreport.Report{},
			importID:    "lossy",
			wantRefused: []string{"destination:lossy", "headers[0].description (value)", "someLegacyBlock (object, 2 keys)"},
		},
		{
			name:        "an unacknowledged entry refuses",
			report:      &importreport.Report{Entries: []importreport.Entry{{RemoteID: "lossy", Lost: lost}}},
			importID:    "lossy",
			wantRefused: []string{"destination:lossy"},
		},
		{
			name:     "an acknowledged entry lets the adopt through",
			report:   &importreport.Report{Entries: []importreport.Entry{{RemoteID: "lossy", Lost: lost, Acknowledged: true}}},
			importID: "lossy",
		},
		{
			name:        "a key added in the dashboard after the review refuses again",
			report:      &importreport.Report{Entries: []importreport.Entry{{RemoteID: "lossy", Lost: lost[:1], Acknowledged: true}}},
			importID:    "lossy",
			wantRefused: []string{"someLegacyBlock (object, 2 keys)"},
		},
		{
			name:     "a destination that loses nothing needs no entry",
			report:   &importreport.Report{},
			importID: "clean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.CheckImports(t.Context(), &planner.Plan{Operations: []*planner.Operation{importOp(tt.importID)}}, nil, tt.report)
			if len(tt.wantRefused) == 0 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, want := range tt.wantRefused {
				assert.Contains(t, err.Error(), want)
			}
			assert.NotContains(t, err.Error(), "hunter2")
		})
	}

	t.Run("only importable destinations are fetched", func(t *testing.T) {
		fetched = nil
		update := &planner.Operation{
			Type: planner.Update,
			Resource: resources.NewResource("lossy", DestinationResourceType, resources.ResourceData{}, nil,
				resources.WithRawData(&DestinationResource{ID: "lossy", Type: "http"}),
				resources.WithResourceImportMetadata("lossy", "ws-1")),
		}
		require.NoError(t, p.CheckImports(t.Context(), &planner.Plan{Operations: []*planner.Operation{update}}, nil, &importreport.Report{}))
		assert.Empty(t, fetched)
	})
}
