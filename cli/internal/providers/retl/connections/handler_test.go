package connections

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type mockStore struct {
	retlClient.RETLStore
	createFunc     func(ctx context.Context, req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	updateFunc     func(ctx context.Context, id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	setExternalIDs []string
}

func (m *mockStore) CreateConnection(ctx context.Context, req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	return m.createFunc(ctx, req)
}

// The API rejects externalId on create, so Create claims it in a second call.
func (m *mockStore) SetConnectionExternalId(ctx context.Context, req *retlClient.SetRETLConnectionExternalIDRequest) error {
	m.setExternalIDs = append(m.setExternalIDs, req.ExternalID)
	return nil
}

func (m *mockStore) UpdateConnection(ctx context.Context, id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	return m.updateFunc(ctx, id, req)
}

func entry(overrides map[string]any) map[string]any {
	e := map[string]any{
		"id":             "users-to-webhook",
		"source":         "#retl-source-table:users-table",
		"destination":    "#destination:webhook-dev",
		"sync_behaviour": "upsert",
		"schedule":       map[string]any{"type": "basic", "every_minutes": 60},
		"identifiers":    []any{map[string]any{"from": "email", "to": "userId"}},
	}
	for k, v := range overrides {
		if v == nil {
			delete(e, k)
			continue
		}
		e[k] = v
	}
	return e
}

func specWith(entries ...map[string]any) *specs.Spec {
	list := make([]any, 0, len(entries))
	for _, e := range entries {
		list = append(list, e)
	}
	return &specs.Spec{
		Version: "rudder/v1",
		Kind:    ResourceKind,
		Spec:    map[string]any{"connections": list},
	}
}

func TestLoadSpecPluralYieldsManyResources(t *testing.T) {
	h := NewHandler(&mockStore{})
	second := entry(map[string]any{"id": "users-to-braze", "destination": "#destination:braze"})
	require.NoError(t, h.LoadSpec("connections.yaml", specWith(entry(nil), second)))

	got, err := h.GetResources()
	require.NoError(t, err)
	assert.Len(t, got, 2, "one plural spec document must yield one resource per entry")
}

func TestParseSpecEmitsOneURNPerEntry(t *testing.T) {
	h := NewHandler(&mockStore{})
	parsed, err := h.ParseSpec("connections.yaml", specWith(entry(nil), entry(map[string]any{"id": "second"})))
	require.NoError(t, err)
	require.Len(t, parsed.URNs, 2)
	assert.Equal(t, "/spec/connections/0/id", parsed.URNs[0].JSONPointerPath)
	assert.Equal(t, "/spec/connections/1/id", parsed.URNs[1].JSONPointerPath)
}

// These rules live only in the Terraform provider's CustomizeDiff and the
// webapp's form today. A raw API caller can violate every one of them.
func TestCrossFieldRules(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[string]any
		wantErr   string
	}{
		{"cursor_column outside upsert", map[string]any{"sync_behaviour": "full", "cursor_column": "updated_at"}, "cursor_column requires sync_behaviour: upsert"},
		{"schedule below the 5 minute floor", map[string]any{"schedule": map[string]any{"type": "basic", "every_minutes": 3}}, "at least 5"},
		{"basic schedule without an interval", map[string]any{"schedule": map[string]any{"type": "basic"}}, "every_minutes is required"},
		{"cron without an expression", map[string]any{"schedule": map[string]any{"type": "cron"}}, "cron_expression is required"},
		{"both event name forms", map[string]any{"event": map[string]any{"type": "track", "name": "a", "name_column": "b"}}, "mutually exclusive"},
		{"source ref of the wrong kind", map[string]any{"source": "#destination:nope"}, "parsing source reference"},
		{"destination ref of the wrong kind", map[string]any{"destination": "#retl-source-table:nope"}, "parsing destination reference"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(&mockStore{})
			err := h.LoadSpec("connections.yaml", specWith(entry(tc.overrides)))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestSourceRefAcceptsBothRetlSourceKinds(t *testing.T) {
	for _, ref := range []string{"#retl-source-table:t", "#retl-source-sql-model:m"} {
		h := NewHandler(&mockStore{})
		require.NoError(t, h.LoadSpec("c.yaml", specWith(entry(map[string]any{"source": ref}))), ref)
	}
}

// The point of the whole merge-semantics discipline: a connection declaring
// only sync_logs must not send a failed_keys section, and one declaring no
// sync_settings at all must send nothing. Sending a fully-populated object
// would reset server-side config the user never mentioned — the bug
// config-backend #6598 fixes for Terraform.
func TestSyncSettingsSendsOnlyDeclaredSections(t *testing.T) {
	t.Run("omitted entirely stays nil", func(t *testing.T) {
		var captured *retlClient.CreateRETLConnectionRequest
		h := NewHandler(&mockStore{createFunc: func(_ context.Context, req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			captured = req
			return &retlClient.RETLConnection{ID: "conn-1"}, nil
		}})
		require.NoError(t, h.LoadSpec("c.yaml", specWith(entry(nil))))
		data := resolvedData(t, h)

		_, err := h.Create(context.Background(), "users-to-webhook", data)
		require.NoError(t, err)
		assert.Nil(t, captured.SyncSettings, "no sync_settings declared, so none must be sent")

		// The API rejects externalId inline: 400 "Fields not allowed for JSON
		// Mapper flow: externalId". It must be claimed in a follow-up call.
		assert.Empty(t, captured.ExternalID, "externalId must not be sent on create")
		assert.Equal(t, []string{"users-to-webhook"}, h.clientSetExternalIDs(),
			"create must claim the external id in a second call")
	})

	t.Run("only the declared section is sent", func(t *testing.T) {
		var captured *retlClient.CreateRETLConnectionRequest
		h := NewHandler(&mockStore{createFunc: func(_ context.Context, req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			captured = req
			return &retlClient.RETLConnection{ID: "conn-1"}, nil
		}})
		require.NoError(t, h.LoadSpec("c.yaml", specWith(entry(map[string]any{
			"sync_settings": map[string]any{"sync_logs": map[string]any{"enabled": true}},
		}))))
		data := resolvedData(t, h)

		_, err := h.Create(context.Background(), "users-to-webhook", data)
		require.NoError(t, err)
		require.NotNil(t, captured.SyncSettings)
		require.NotNil(t, captured.SyncSettings.SyncLogsConfig)
		assert.Nil(t, captured.SyncSettings.FailedKeysConfig, "an undeclared section must not be sent")
		assert.Nil(t, captured.SyncSettings.SyncLogsConfig.LogRetentionInDays, "an undeclared field must not be sent")
	})
}

func TestUpdateRejectsImmutableChange(t *testing.T) {
	h := NewHandler(&mockStore{})
	require.NoError(t, h.LoadSpec("c.yaml", specWith(entry(nil))))
	data := resolvedData(t, h)

	state := resources.ResourceData{IDKey: "conn-1", SyncBehaviourKey: "full"}
	_, err := h.Update(context.Background(), "users-to-webhook", data, state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sync_behaviour cannot be changed")
}

func TestCreateFailsIfRefsWereNotDereferenced(t *testing.T) {
	h := NewHandler(&mockStore{})
	require.NoError(t, h.LoadSpec("c.yaml", specWith(entry(nil))))
	res, err := h.GetResources()
	require.NoError(t, err)

	// Raw graph data still holds PropertyRefs; the syncer resolves them before
	// the lifecycle runs. Calling Create with unresolved refs must not silently
	// send an empty id.
	_, err = h.Create(context.Background(), "users-to-webhook", res[0].Data())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not dereferenced")
}

// resolvedData mimics what the syncer hands the lifecycle: endpoint refs
// replaced by their resolved remote ids.
// clientSetExternalIDs surfaces what the handler asked the store to claim.
func (h *Handler) clientSetExternalIDs() []string {
	return h.client.(*mockStore).setExternalIDs
}

func resolvedData(t *testing.T, h *Handler) resources.ResourceData {
	t.Helper()
	res, err := h.GetResources()
	require.NoError(t, err)
	require.Len(t, res, 1)
	data := res[0].Data()
	data[SourceKey] = "src-remote-1"
	data[DestinationKey] = "dst-remote-1"
	return data
}
