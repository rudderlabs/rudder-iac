package connection

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const localID = "users-to-webhook"

// stateData is what the syncer hands Update and Delete: the prior resource
// state's Input — the canonical config and the enabled flag — merged with its
// Output, the remote identifiers the lifecycle wrote.
func stateData(t *testing.T, config ConfigSpec) resources.ResourceData {
	t.Helper()

	data, err := configToData(config)
	require.NoError(t, err)
	return resources.ResourceData{
		EnabledKey:       true,
		ConfigKey:        data,
		IDKey:            "conn-remote-1",
		SourceIDKey:      "src-1",
		DestinationIDKey: "dst-1",
	}
}

func newLifecycleHandler(client retlClient.RETLStore) *Handler {
	return NewHandler(client, "retl", nil)
}

// remoteConnection is the row the API reports back for the create fixtures.
func remoteConnection(id string, request *retlClient.CreateRETLConnectionRequest) *retlClient.RETLConnection {
	return &retlClient.RETLConnection{
		ID:            id,
		SourceID:      request.SourceID,
		DestinationID: request.DestinationID,
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("posts the connection then claims it", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		output, err := newLifecycleHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))
		require.NoError(t, err)

		// The claim is a second call on purpose: the per-flow allow-list
		// rejects externalId in the create body.
		assert.Equal(t, []string{"CreateConnection", "SetConnectionExternalId"}, mock.Calls)
		require.Len(t, mock.CreateCalls, 1)
		body, err := json.Marshal(mock.CreateCalls[0])
		require.NoError(t, err)
		assert.JSONEq(t, `{
			"sourceId": "src-1",
			"destinationId": "dst-1",
			"enabled": true,
			"schedule": {"type": "basic", "everyMinutes": 30},
			"syncBehaviour": "upsert",
			"identifiers": [{"from": "id", "to": "user_id"}],
			"mappings": [{"from": "email", "to": "traits.email"}]
		}`, string(body))

		assert.Equal(t,
			[]retlClient.SetRETLConnectionExternalIDRequest{{ID: "conn-remote-1", ExternalID: localID}},
			mock.SetExternalIDCalls,
		)
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-1",
		}, output)
	})

	t.Run("refuses to call the api with an unresolved endpoint", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = &resources.PropertyRef{URN: "destination:webhook"}

		mock := &MockConnectionClient{}
		_, err := newLifecycleHandler(mock).Create(context.Background(), localID, data)

		assert.ErrorContains(t, err, `building create request for rETL connection "users-to-webhook"`)
		assert.ErrorContains(t, err, `"destination" is not a resolved endpoint id`)
		assert.Empty(t, mock.Calls)
	})

	t.Run("surfaces create errors", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			CreateFunc: func(_ *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
				return nil, errors.New("source and destination are already connected")
			},
		}
		_, err := newLifecycleHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `creating rETL connection "users-to-webhook": source and destination are already connected`)
		assert.Equal(t, []string{"CreateConnection"}, mock.Calls)
	})
}

// TestCreatePartialRecovery covers the window between the POST and the claim.
// The row exists by then, so every branch has to decide from what the remote
// actually says — never from the assumption that a failed claim means a failed
// create.
func TestCreatePartialRecovery(t *testing.T) {
	t.Parallel()

	timeout := func(_ context.Context, _ *retlClient.SetRETLConnectionExternalIDRequest) error {
		return errors.New("context deadline exceeded")
	}

	t.Run("a timed out claim that the server applied is a success", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			SetExternalIDFunc: timeout,
			GetFunc: func(id string) (*retlClient.RETLConnection, error) {
				return &retlClient.RETLConnection{ID: id, SourceID: "src-1", DestinationID: "dst-1", ExternalID: localID}, nil
			},
		}
		output, err := newLifecycleHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))
		require.NoError(t, err)

		assert.Equal(t, []string{"CreateConnection", "SetConnectionExternalId", "GetConnection"}, mock.Calls)
		assert.Equal(t, []string{"conn-remote-1"}, mock.GetCalls)
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-1",
		}, output)
	})

	t.Run("a confirmed unclaimed row is deleted so the next apply is clean", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{SetExternalIDFunc: timeout}
		_, err := newLifecycleHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		require.Error(t, err)
		assert.Equal(t, []string{"CreateConnection", "SetConnectionExternalId", "GetConnection", "DeleteConnection"}, mock.Calls)
		assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls)
		assert.Len(t, mock.CreateCalls, 1)
		assert.ErrorContains(t, err, "context deadline exceeded")
		assert.ErrorContains(t, err, "connection conn-remote-1 carried no external id and was deleted")
	})

	t.Run("an inconclusive read back leaves the row in place", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			SetExternalIDFunc: timeout,
			GetFunc: func(_ string) (*retlClient.RETLConnection, error) {
				return nil, errors.New("service unavailable")
			},
		}
		_, err := newLifecycleHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		require.Error(t, err)
		assert.Equal(t, []string{"CreateConnection", "SetConnectionExternalId", "GetConnection"}, mock.Calls)
		assert.Empty(t, mock.DeleteCalls)
		assert.ErrorContains(t, err, "context deadline exceeded")
		assert.ErrorContains(t, err, "service unavailable")
		assert.ErrorContains(t, err, "reading connection conn-remote-1 back failed too")
		assert.ErrorContains(t, err, "import or delete it before the next apply")
	})

	t.Run("a row claimed by someone else is never touched", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			SetExternalIDFunc: timeout,
			GetFunc: func(id string) (*retlClient.RETLConnection, error) {
				return &retlClient.RETLConnection{ID: id, ExternalID: "orders-to-webhook"}, nil
			},
		}
		_, err := newLifecycleHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		require.Error(t, err)
		assert.Equal(t, []string{"CreateConnection", "SetConnectionExternalId", "GetConnection"}, mock.Calls)
		assert.Empty(t, mock.DeleteCalls)
		assert.ErrorContains(t, err, `connection conn-remote-1 already carries external id "orders-to-webhook"`)
		assert.ErrorContains(t, err, "reconcile or import it before the next apply")
	})

	// The claim above only returns a timeout-shaped error. Here the caller's
	// context is genuinely dead by the time the claim returns, which is what
	// the recovery has to survive: it runs detached, so the read-back and the
	// compensating delete still reach the server.
	t.Run("a claim killed by the caller's context still gets compensated", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mock := &MockConnectionClient{
			SetExternalIDFunc: func(ctx context.Context, _ *retlClient.SetRETLConnectionExternalIDRequest) error {
				cancel()
				return ctx.Err()
			},
		}
		_, err := newLifecycleHandler(mock).Create(ctx, localID, graphData(t, jsonMapperConfig()))

		require.Error(t, err)
		assert.Equal(t, []string{"CreateConnection", "SetConnectionExternalId", "GetConnection", "DeleteConnection"}, mock.Calls)
		assert.Equal(t, []string{"conn-remote-1"}, mock.GetCalls)
		assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls)
		assert.ErrorContains(t, err, "context canceled")
		assert.ErrorContains(t, err, "connection conn-remote-1 carried no external id and was deleted")
	})

	t.Run("a failed cleanup reports both failures and the orphan", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			SetExternalIDFunc: timeout,
			DeleteFunc: func(_ string) error {
				return errors.New("forbidden")
			},
		}
		_, err := newLifecycleHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		require.Error(t, err)
		assert.Equal(t, []string{"CreateConnection", "SetConnectionExternalId", "GetConnection", "DeleteConnection"}, mock.Calls)
		assert.ErrorContains(t, err, "context deadline exceeded")
		assert.ErrorContains(t, err, "forbidden")
		assert.ErrorContains(t, err, "connection conn-remote-1 carries no external id but deleting it failed too")
		assert.ErrorContains(t, err, "orphaned")
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("equal config and enabled make no api calls", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		output, err := newLifecycleHandler(mock).Update(context.Background(), localID,
			graphData(t, jsonMapperConfig()), stateData(t, jsonMapperConfig()))
		require.NoError(t, err)

		assert.Empty(t, mock.Calls)
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-1",
		}, output)
	})

	t.Run("a mutable change is a single put", func(t *testing.T) {
		t.Parallel()

		config := jsonMapperConfig()
		config.Mappings = append(config.Mappings, MappingSpec{From: "plan", To: "traits.plan"})
		data := graphData(t, config)
		data[EnabledKey] = false

		mock := &MockConnectionClient{
			UpdateFunc: func(id string, _ *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
				return &retlClient.RETLConnection{ID: id, SourceID: "src-1", DestinationID: "dst-1"}, nil
			},
		}
		output, err := newLifecycleHandler(mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))
		require.NoError(t, err)

		assert.Equal(t, []string{"UpdateConnection"}, mock.Calls)
		require.Len(t, mock.UpdateCalls, 1)
		assert.Equal(t, "conn-remote-1", mock.UpdateCalls[0].ID)

		// The immutable fields are absent from the body: the API refuses them
		// even unchanged. Constants travel as an explicit empty list, which is
		// what clears them.
		body, err := json.Marshal(mock.UpdateCalls[0].Request)
		require.NoError(t, err)
		assert.JSONEq(t, `{
			"enabled": false,
			"schedule": {"type": "basic", "everyMinutes": 30},
			"identifiers": [{"from": "id", "to": "user_id"}],
			"mappings": [
				{"from": "email", "to": "traits.email"},
				{"from": "plan", "to": "traits.plan"}
			],
			"constants": []
		}`, string(body))

		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-1",
		}, output)
	})

	t.Run("surfaces update errors", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[EnabledKey] = false

		mock := &MockConnectionClient{
			UpdateFunc: func(_ string, _ *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
				return nil, errors.New("schedule is invalid")
			},
		}
		_, err := newLifecycleHandler(mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `updating rETL connection "users-to-webhook": schedule is invalid`)
	})

	t.Run("errors when state lacks the remote id", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		_, err := newLifecycleHandler(mock).Update(context.Background(), localID,
			graphData(t, jsonMapperConfig()), resources.ResourceData{})

		assert.ErrorContains(t, err, "missing id in state")
		assert.Empty(t, mock.Calls)
	})

	// A config that does not decode must never cost a connection: the checks
	// run before the delete a replacement would otherwise start with.
	t.Run("validates the whole entry before deleting anything", func(t *testing.T) {
		t.Parallel()

		state := stateData(t, jsonMapperConfig())
		state[SourceIDKey] = "src-9"

		tests := map[string]func(resources.ResourceData){
			"an unresolved endpoint": func(data resources.ResourceData) {
				data[SourceKey] = &resources.PropertyRef{URN: "retl-source-sql-model:users"}
			},
			"a missing enabled flag": func(data resources.ResourceData) { delete(data, EnabledKey) },
			"a config that does not decode": func(data resources.ResourceData) {
				data[ConfigKey] = map[string]any{"identifiers": "id"}
			},
		}

		for name, corrupt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				data := graphData(t, jsonMapperConfig())
				corrupt(data)

				mock := &MockConnectionClient{}
				_, err := newLifecycleHandler(mock).Update(context.Background(), localID, data, state)

				assert.ErrorContains(t, err, `connection "users-to-webhook"`)
				assert.Empty(t, mock.Calls)
			})
		}
	})

	// An endpoint id state cannot answer for must never read as "the endpoint
	// changed": that answer starts a replacement by deleting a live connection.
	t.Run("refuses state whose endpoint ids are unusable", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			corrupt func(resources.ResourceData)
			wantErr string
		}{
			"a missing source id":           {func(s resources.ResourceData) { delete(s, SourceIDKey) }, "missing sourceId in state"},
			"an empty source id":            {func(s resources.ResourceData) { s[SourceIDKey] = "" }, "missing sourceId in state"},
			"a source id of the wrong type": {func(s resources.ResourceData) { s[SourceIDKey] = 1 }, "missing sourceId in state"},
			"a missing destination id":      {func(s resources.ResourceData) { delete(s, DestinationIDKey) }, "missing destinationId in state"},
			"an empty destination id":       {func(s resources.ResourceData) { s[DestinationIDKey] = "" }, "missing destinationId in state"},
			"a destination id of the wrong type": {
				func(s resources.ResourceData) { s[DestinationIDKey] = []string{"dst-1"} },
				"missing destinationId in state",
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				state := stateData(t, jsonMapperConfig())
				test.corrupt(state)

				mock := &MockConnectionClient{}
				_, err := newLifecycleHandler(mock).Update(context.Background(), localID,
					graphData(t, jsonMapperConfig()), state)

				assert.ErrorContains(t, err, `connection "users-to-webhook": `+test.wantErr)
				assert.Empty(t, mock.Calls)
			})
		}
	})

	t.Run("a stored config that does not decode stops the update", func(t *testing.T) {
		t.Parallel()

		state := stateData(t, jsonMapperConfig())
		state[ConfigKey] = "upsert"

		mock := &MockConnectionClient{}
		_, err := newLifecycleHandler(mock).Update(context.Background(), localID, graphData(t, jsonMapperConfig()), state)

		assert.ErrorContains(t, err, "reading stored connection config")
		assert.Empty(t, mock.Calls)
	})
}

// TestUpdateReplacesOnImmutableChange walks every field the API refuses on a
// PUT. Each one becomes a delete followed by the two-call create, and the error
// names the field so the plan output explains why the connection was replaced.
func TestUpdateReplacesOnImmutableChange(t *testing.T) {
	t.Parallel()

	objectMapping := func(object string) ConfigSpec {
		config := objectMappingConfig()
		config.Object = ptr(object)
		return config
	}
	withEvent := func(name string) ConfigSpec {
		config := jsonMapperConfig()
		config.Event = &EventSpec{Type: "track", Name: name}
		return config
	}
	withCursor := func(column string) ConfigSpec {
		config := jsonMapperConfig()
		config.CursorColumn = column
		return config
	}
	withBehaviour := func(behaviour string) ConfigSpec {
		config := jsonMapperConfig()
		config.SyncBehaviour = behaviour
		return config
	}

	tests := []struct {
		name     string
		desired  ConfigSpec
		stored   ConfigSpec
		sourceID string
		wantHint string
	}{
		{name: "source", desired: jsonMapperConfig(), stored: jsonMapperConfig(), sourceID: "src-9", wantHint: "after source changed"},
		{name: "sync behaviour", desired: withBehaviour("mirror"), stored: jsonMapperConfig(), wantHint: "after sync_behaviour changed"},
		{name: "cursor column", desired: withCursor("updated_at"), stored: jsonMapperConfig(), wantHint: "after cursor_column changed"},
		{name: "object", desired: objectMapping("Lead"), stored: objectMapping("Contact"), wantHint: "after object changed"},
		{name: "event", desired: withEvent("login"), stored: withEvent("signup"), wantHint: "after event changed"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			data := graphData(t, test.desired)
			state := stateData(t, test.stored)
			if test.sourceID != "" {
				state[SourceIDKey] = test.sourceID
			}

			// The hint only shows up when the recreate fails, so prove the
			// field name is the one reported by failing it.
			failing := &MockConnectionClient{
				CreateFunc: func(_ *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
					return nil, errors.New("plan gate")
				},
			}
			_, err := newLifecycleHandler(failing).Update(context.Background(), localID, data, state)
			assert.ErrorContains(t, err, test.wantHint)
			assert.ErrorContains(t, err, "the previous connection was deleted")
			assert.ErrorContains(t, err, "plan gate")

			// The replacement is exactly one delete and one create: no PUT was
			// attempted first, and the failed create left no second delete.
			assert.Equal(t, []string{"DeleteConnection", "CreateConnection"}, failing.Calls)
		})
	}
}

// TestUpdateReplacementIdentifiers covers the two outcomes the backend can
// produce for a recreate: the soft-deleted row comes back with its remote id
// when the pair is unchanged, and a new pair gets a new one. Either way the
// create response is what lands in state.
func TestUpdateReplacementIdentifiers(t *testing.T) {
	t.Parallel()

	// replacedPair is a connection pointed at a different destination: the
	// endpoint change is what forces the replacement.
	replacedPair := func(t *testing.T) (resources.ResourceData, resources.ResourceData) {
		t.Helper()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-2"
		return data, stateData(t, jsonMapperConfig())
	}

	t.Run("recreating the same pair may revive its remote id", func(t *testing.T) {
		t.Parallel()

		// Both endpoints stay put; an immutable config field is what forces the
		// replacement, so the backend sees the pair it just soft-deleted.
		desired := jsonMapperConfig()
		desired.SyncBehaviour = "mirror"

		mock := &MockConnectionClient{
			CreateFunc: func(request *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
				return remoteConnection("conn-remote-1", request), nil
			},
		}
		output, err := newLifecycleHandler(mock).Update(context.Background(), localID,
			graphData(t, desired), stateData(t, jsonMapperConfig()))
		require.NoError(t, err)

		assert.Equal(t, []string{"DeleteConnection", "CreateConnection", "SetConnectionExternalId"}, mock.Calls)
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-1",
		}, output)
		assert.Equal(t,
			[]retlClient.SetRETLConnectionExternalIDRequest{{ID: "conn-remote-1", ExternalID: localID}},
			mock.SetExternalIDCalls,
		)
	})

	t.Run("a new pair returns a new remote id", func(t *testing.T) {
		t.Parallel()

		data, state := replacedPair(t)
		mock := &MockConnectionClient{
			CreateFunc: func(request *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
				return remoteConnection("conn-remote-2", request), nil
			},
		}
		output, err := newLifecycleHandler(mock).Update(context.Background(), localID, data, state)
		require.NoError(t, err)

		assert.Equal(t, []string{"DeleteConnection", "CreateConnection", "SetConnectionExternalId"}, mock.Calls)
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-2",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-2",
		}, output)
		assert.Equal(t,
			[]retlClient.SetRETLConnectionExternalIDRequest{{ID: "conn-remote-2", ExternalID: localID}},
			mock.SetExternalIDCalls,
		)
	})

	t.Run("a failed delete stops the replacement before the create", func(t *testing.T) {
		t.Parallel()

		data, state := replacedPair(t)
		mock := &MockConnectionClient{
			DeleteFunc: func(_ string) error { return errors.New("forbidden") },
		}
		_, err := newLifecycleHandler(mock).Update(context.Background(), localID, data, state)

		assert.EqualError(t, err, `deleting rETL connection "users-to-webhook": forbidden`)
		assert.Equal(t, []string{"DeleteConnection"}, mock.Calls)
		assert.Empty(t, mock.CreateCalls)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("deletes only the connection", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		require.NoError(t, newLifecycleHandler(mock).Delete(context.Background(), localID, stateData(t, jsonMapperConfig())))

		assert.Equal(t, []string{"DeleteConnection"}, mock.Calls)
		assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls)
	})

	t.Run("errors when state lacks the remote id", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		err := newLifecycleHandler(mock).Delete(context.Background(), localID, resources.ResourceData{})

		assert.ErrorContains(t, err, "missing id in state")
		assert.Empty(t, mock.Calls)
	})

	t.Run("surfaces server errors", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			DeleteFunc: func(_ string) error { return errors.New("forbidden") },
		}
		err := newLifecycleHandler(mock).Delete(context.Background(), localID, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `deleting rETL connection "users-to-webhook": forbidden`)
	})
}
