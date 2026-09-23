package connection

import (
	"context"
	"errors"
	"testing"

	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const localID = "users-to-webhook"

// lifecycleHandler builds a handler for the create/update/delete path. The
// create path judges the destination before it writes, so the registry is the
// shared fixture here too; a test that reaches a create gives its client the
// destination catalog with lifecycleClient.
func lifecycleHandler(t *testing.T, client *MockConnectionClient) *Handler {
	t.Helper()
	return NewHandler(client, importDir, testRegistry(t))
}

func lifecycleClient() *MockConnectionClient {
	return &MockConnectionClient{Destinations: remoteDestinations()}
}

// stateData is what the syncer hands Update and Delete: the prior Input (the
// canonical config and enabled flag) merged with the Output (the remote ids).
func stateData(t *testing.T, config ConfigSpec) resources.ResourceData {
	t.Helper()

	data, err := configToMap(config)
	require.NoError(t, err)
	return resources.ResourceData{
		EnabledKey:       true,
		ConfigKey:        data,
		IDKey:            "conn-remote-1",
		SourceIDKey:      "src-1",
		DestinationIDKey: "dst-1",
	}
}

func configMap(data resources.ResourceData) map[string]any {
	return data[ConfigKey].(map[string]any)
}

var remoteIDs = &resources.ResourceData{IDKey: "conn-remote-1", SourceIDKey: "src-1", DestinationIDKey: "dst-1"}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("posts the connection with its external id", func(t *testing.T) {
		t.Parallel()

		mock := lifecycleClient()
		output, err := lifecycleHandler(t, mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))
		require.NoError(t, err)

		require.Len(t, mock.CreateCalls, 1)
		assert.Equal(t, localID, mock.CreateCalls[0].ExternalID)
		assert.Equal(t, remoteIDs, output)
	})

	t.Run("refuses to call the api with an unresolved endpoint", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = &resources.PropertyRef{URN: "destination:webhook"}

		mock := &MockConnectionClient{}
		_, err := lifecycleHandler(t, mock).Create(context.Background(), localID, data)

		assert.ErrorContains(t, err, `building create request for rETL connection "users-to-webhook"`)
		assert.Empty(t, mock.CreateCalls)
	})

	t.Run("surfaces create errors", func(t *testing.T) {
		t.Parallel()

		mock := lifecycleClient()
		mock.CreateFunc = func(_ *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			return nil, errors.New("source and destination are already connected")
		}
		_, err := lifecycleHandler(t, mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `creating rETL connection "users-to-webhook": source and destination are already connected`)
	})

	// What the read path skips must not be created: the row would leave no
	// state, so every later apply plans the same create and the backend refuses
	// it as a duplicate. DEX-917.
	t.Run("refuses a destination the read path would skip", func(t *testing.T) {
		t.Parallel()

		// dst-eventstream is S3: registered, but it takes no warehouse source.
		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-eventstream"

		mock := lifecycleClient()
		_, err := lifecycleHandler(t, mock).Create(context.Background(), localID, data)

		assert.EqualError(t, err, `vetting rETL connection "users-to-webhook": destination type "S3" does not accept warehouse sources`)
		assert.Empty(t, mock.CreateCalls, "the api must not be called for a connection that cannot be read back")
	})

	// The cached catalog is read before the apply starts, so a destination this
	// same apply created is absent from it. Refetching is what keeps the check
	// from refusing a perfectly good create.
	t.Run("refetches the catalog for a destination created this run", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{Destinations: []apiClient.Destination{}}
		handler := lifecycleHandler(t, mock)

		// The first create caches the catalog as it stood before dst-1 existed.
		_, err := handler.Create(context.Background(), localID, graphData(t, jsonMapperConfig()))
		require.EqualError(t, err, `vetting rETL connection "users-to-webhook": destination "dst-1" was not found in this workspace`)

		mock.Destinations = remoteDestinations()
		_, err = handler.Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		require.NoError(t, err)
		require.Len(t, mock.CreateCalls, 1)
		assert.Equal(t, 2, mock.DestinationsCalls, "a miss must be refetched, not believed")
	})

	// The write path used to surface the registry lookup verbatim, naming
	// neither the flag nor any remedy.
	t.Run("names the flag when the destination definition is not registered", func(t *testing.T) {
		t.Parallel()

		// dst-old is HTTP at a version the registry does not carry.
		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-old"

		mock := lifecycleClient()
		_, err := lifecycleHandler(t, mock).Create(context.Background(), localID, data)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `vetting rETL connection "users-to-webhook": destination type "HTTP" version 9 is not registered in this CLI; if it is an unverified destination, set RUDDERSTACK_X_UNVERIFIED_DESTINATIONS=true`)
		assert.Empty(t, mock.CreateCalls)
	})

	// The one skip reason that cannot be vetted before the call: the server may
	// store a config the spec cannot express. Left in place, the row is skipped
	// on read and re-created on every apply — DEX-917's loop — so it is undone.
	t.Run("deletes a connection it cannot read back", func(t *testing.T) {
		t.Parallel()

		mock := lifecycleClient()
		mock.CreateFunc = func(req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			created := echoCreated("conn-remote-1", req)
			created.DestinationConfig = []byte(`{"audienceId":"a-1"}`)
			return created, nil
		}
		_, err := lifecycleHandler(t, mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `rETL connection "users-to-webhook" was created but cannot be read back, so it was deleted again: connection "conn-remote-1": destination-specific configuration has no spec equivalent: connection config cannot be represented as a spec`)
		assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls)
	})

	t.Run("reports both failures when the undo fails too", func(t *testing.T) {
		t.Parallel()

		mock := lifecycleClient()
		mock.CreateFunc = func(req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			created := echoCreated("conn-remote-1", req)
			created.Identifiers = nil
			return created, nil
		}
		mock.DeleteFunc = func(string) error { return errors.New("gateway timeout") }
		_, err := lifecycleHandler(t, mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `rETL connection "users-to-webhook" was created as "conn-remote-1" but cannot be read back (connection "conn-remote-1": no identifiers: connection config cannot be represented as a spec), and deleting it failed: gateway timeout`)
	})

	t.Run("refuses a destination that is not in the workspace at all", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-missing"

		mock := lifecycleClient()
		_, err := lifecycleHandler(t, mock).Create(context.Background(), localID, data)

		assert.EqualError(t, err, `vetting rETL connection "users-to-webhook": destination "dst-missing" was not found in this workspace`)
		assert.Empty(t, mock.CreateCalls)
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("equal config and enabled make no api calls", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		output, err := lifecycleHandler(t, mock).Update(context.Background(), localID,
			graphData(t, jsonMapperConfig()), stateData(t, jsonMapperConfig()))
		require.NoError(t, err)

		assert.Equal(t, &MockConnectionClient{}, mock)
		assert.Equal(t, remoteIDs, output)
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
		output, err := lifecycleHandler(t, mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))
		require.NoError(t, err)

		assert.Equal(t, []string{"conn-remote-1"}, mock.UpdateCalls)
		assert.Empty(t, mock.DeleteCalls)
		assert.Empty(t, mock.CreateCalls)
		assert.Equal(t, remoteIDs, output)
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
		_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `updating rETL connection "users-to-webhook": schedule is invalid`)
	})

	t.Run("errors when state lacks the remote id", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, graphData(t, jsonMapperConfig()), resources.ResourceData{})

		assert.ErrorContains(t, err, "missing id in state")
		assert.Equal(t, &MockConnectionClient{}, mock)
	})

	// Every check runs before the delete a replacement starts with: malformed
	// input, or a create body the API would refuse, must never cost a connection.
	t.Run("validates everything before deleting anything", func(t *testing.T) {
		t.Parallel()

		tests := map[string]func(data, state resources.ResourceData){
			"an unresolved endpoint": func(data, _ resources.ResourceData) {
				data[SourceKey] = &resources.PropertyRef{URN: "retl-source-sql-model:users"}
			},
			"a missing enabled flag":        func(data, _ resources.ResourceData) { delete(data, EnabledKey) },
			"a config that does not decode": func(data, _ resources.ResourceData) { data[ConfigKey] = map[string]any{"identifiers": "id"} },
			"a create body the api refuses": func(data, _ resources.ResourceData) {
				configMap(data)["object"] = "Contact"
				configMap(data)["event"] = map[string]any{"type": "track", "name": "signup"}
			},
			// The client refuses this one rather than the converter, so it only
			// stays out of the API if toCreateRequest checks it before the delete.
			"a schedule the client refuses": func(data, _ resources.ResourceData) {
				delete(configMap(data), "schedule")
			},
		}

		for name, corrupt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				data, state := graphData(t, jsonMapperConfig()), stateData(t, jsonMapperConfig())
				state[SourceIDKey] = "src-9"
				corrupt(data, state)

				mock := &MockConnectionClient{}
				_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, data, state)

				assert.ErrorContains(t, err, `connection "users-to-webhook"`)
				assert.Equal(t, &MockConnectionClient{}, mock)
			})
		}
	})

	// The stored config is the other half of the diff an update computes, so it
	// has to decode before the PUT is built. A replacement never reads it: the
	// create body comes from the desired entry alone.
	t.Run("refuses an update whose stored config does not decode", func(t *testing.T) {
		t.Parallel()

		state := stateData(t, jsonMapperConfig())
		state[ConfigKey] = "upsert"

		mock := &MockConnectionClient{}
		_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, graphData(t, jsonMapperConfig()), state)

		assert.ErrorContains(t, err, `connection "users-to-webhook": reading stored connection config`)
		assert.Equal(t, &MockConnectionClient{}, mock)
	})

	t.Run("rejects an immutable field change without touching the api", func(t *testing.T) {
		t.Parallel()

		tests := map[string]func(config map[string]any){
			"sync_behaviour": func(config map[string]any) { config["sync_behaviour"] = "mirror" },
			"cursor_column":  func(config map[string]any) { config["cursor_column"] = "updated_at" },
			"object":         func(config map[string]any) { config["object"] = "Contact" },
			"event":          func(config map[string]any) { config["event"] = map[string]any{"type": "track", "name": "signup"} },
		}

		for field, change := range tests {
			t.Run(field, func(t *testing.T) {
				t.Parallel()

				data := graphData(t, jsonMapperConfig())
				change(configMap(data))

				mock := &MockConnectionClient{}
				_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

				assert.ErrorContains(t, err, field+" is immutable")
				assert.ErrorContains(t, err, "delete and recreate the connection")
				assert.Equal(t, &MockConnectionClient{}, mock)
			})
		}
	})

	// An endpoint change becomes one delete and one create, and whatever the
	// create returns — a revived id, if the pair being moved to had a row of its
	// own once, or a brand new one — is what lands in state.
	t.Run("replaces on an endpoint change", func(t *testing.T) {
		t.Parallel()

		tests := map[string]func(data, state resources.ResourceData){
			"source":      func(_, state resources.ResourceData) { state[SourceIDKey] = "src-9" },
			"destination": func(data, _ resources.ResourceData) { data[DestinationKey] = "dst-2" },
		}

		for name, change := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				data, state := graphData(t, jsonMapperConfig()), stateData(t, jsonMapperConfig())
				change(data, state)

				var created *retlClient.RETLConnection
				mock := lifecycleClient()
				mock.CreateFunc = func(request *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
					created = echoCreated("conn-remote-2", request)
					return created, nil
				}
				output, err := lifecycleHandler(t, mock).Update(context.Background(), localID, data, state)
				require.NoError(t, err)

				assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls)
				require.Len(t, mock.CreateCalls, 1)
				assert.Equal(t, localID, mock.CreateCalls[0].ExternalID)
				assert.Empty(t, mock.UpdateCalls)
				assert.Equal(t, toResourceData(created), output)
			})
		}
	})

	// The refusal has to land before the delete. Reached through Create, it
	// costs the live connection and leaves nothing in its place. DEX-917.
	t.Run("an unusable destination stops the replacement before the delete", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-eventstream"

		mock := lifecycleClient()
		_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `connection "users-to-webhook": destination type "S3" does not accept warehouse sources`)
		assert.Empty(t, mock.DeleteCalls, "the live connection must survive a refused replacement")
		assert.Empty(t, mock.CreateCalls)
	})

	t.Run("a failed delete stops the replacement before the create", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-2"

		mock := lifecycleClient()
		mock.DeleteFunc = func(_ string) error { return errors.New("forbidden") }
		_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `deleting rETL connection "users-to-webhook": forbidden`)
		assert.Empty(t, mock.CreateCalls)
	})

	t.Run("a failed recreate says the old connection is gone", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-2"

		mock := lifecycleClient()
		mock.CreateFunc = func(_ *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			return nil, errors.New("plan gate")
		}
		_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `recreating rETL connection "users-to-webhook" after an endpoint change (the previous connection was deleted): creating rETL connection "users-to-webhook": plan gate`)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("deletes only the connection", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		require.NoError(t, lifecycleHandler(t, mock).Delete(context.Background(), localID, stateData(t, jsonMapperConfig())))

		assert.Equal(t, &MockConnectionClient{DeleteCalls: []string{"conn-remote-1"}}, mock)
	})

	t.Run("errors when state lacks the remote id", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		err := lifecycleHandler(t, mock).Delete(context.Background(), localID, resources.ResourceData{})

		assert.ErrorContains(t, err, "missing id in state")
		assert.Empty(t, mock.DeleteCalls)
	})

	t.Run("surfaces server errors", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{DeleteFunc: func(_ string) error { return errors.New("forbidden") }}
		err := lifecycleHandler(t, mock).Delete(context.Background(), localID, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `deleting rETL connection "users-to-webhook": forbidden`)
	})
}
