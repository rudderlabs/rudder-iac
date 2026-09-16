package connection

import (
	"context"
	"errors"
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const localID = "users-to-webhook"

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

		mock := &MockConnectionClient{}
		output, err := NewHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))
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
		_, err := NewHandler(mock).Create(context.Background(), localID, data)

		assert.ErrorContains(t, err, `building create request for rETL connection "users-to-webhook"`)
		assert.Empty(t, mock.CreateCalls)
	})

	t.Run("surfaces create errors", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			CreateFunc: func(_ *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
				return nil, errors.New("source and destination are already connected")
			},
		}
		_, err := NewHandler(mock).Create(context.Background(), localID, graphData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `creating rETL connection "users-to-webhook": source and destination are already connected`)
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("equal config and enabled make no api calls", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		output, err := NewHandler(mock).Update(context.Background(), localID,
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
		output, err := NewHandler(mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))
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
		_, err := NewHandler(mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `updating rETL connection "users-to-webhook": schedule is invalid`)
	})

	t.Run("errors when state lacks the remote id", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		_, err := NewHandler(mock).Update(context.Background(), localID, graphData(t, jsonMapperConfig()), resources.ResourceData{})

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
			"a stored config that does not decode": func(_, state resources.ResourceData) {
				state[ConfigKey] = "upsert"
			},
			"a create body the api refuses": func(data, _ resources.ResourceData) {
				configMap(data)["object"] = "Contact"
				configMap(data)["event"] = map[string]any{"type": "track", "name": "signup"}
			},
		}

		for name, corrupt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				data, state := graphData(t, jsonMapperConfig()), stateData(t, jsonMapperConfig())
				state[SourceIDKey] = "src-9"
				corrupt(data, state)

				mock := &MockConnectionClient{}
				_, err := NewHandler(mock).Update(context.Background(), localID, data, state)

				assert.ErrorContains(t, err, `connection "users-to-webhook"`)
				assert.Equal(t, &MockConnectionClient{}, mock)
			})
		}
	})

	// Each change the API refuses on a PUT becomes one delete and one create,
	// and whatever the create returns — a revived id for the same pair, a new
	// one for a new pair — is what lands in state.
	t.Run("replaces on an endpoint or immutable field change", func(t *testing.T) {
		t.Parallel()

		tests := map[string]func(data, state resources.ResourceData){
			"source":         func(_, state resources.ResourceData) { state[SourceIDKey] = "src-9" },
			"destination":    func(data, _ resources.ResourceData) { data[DestinationKey] = "dst-2" },
			"sync behaviour": func(data, _ resources.ResourceData) { configMap(data)["sync_behaviour"] = "mirror" },
			"cursor column":  func(data, _ resources.ResourceData) { configMap(data)["cursor_column"] = "updated_at" },
			"object":         func(data, _ resources.ResourceData) { configMap(data)["object"] = "Contact" },
			"event": func(data, _ resources.ResourceData) {
				configMap(data)["event"] = map[string]any{"type": "track", "name": "signup"}
			},
		}

		for name, change := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				data, state := graphData(t, jsonMapperConfig()), stateData(t, jsonMapperConfig())
				change(data, state)

				var created *retlClient.RETLConnection
				mock := &MockConnectionClient{
					CreateFunc: func(request *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
						created = &retlClient.RETLConnection{ID: "conn-remote-2", SourceID: request.SourceID, DestinationID: request.DestinationID}
						return created, nil
					},
				}
				output, err := NewHandler(mock).Update(context.Background(), localID, data, state)
				require.NoError(t, err)

				assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls)
				require.Len(t, mock.CreateCalls, 1)
				assert.Equal(t, localID, mock.CreateCalls[0].ExternalID)
				assert.Empty(t, mock.UpdateCalls)
				assert.Equal(t, toResourceData(created), output)
			})
		}
	})

	t.Run("a failed delete stops the replacement before the create", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-2"

		mock := &MockConnectionClient{DeleteFunc: func(_ string) error { return errors.New("forbidden") }}
		_, err := NewHandler(mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `deleting rETL connection "users-to-webhook": forbidden`)
		assert.Empty(t, mock.CreateCalls)
	})

	t.Run("a failed recreate says the old connection is gone", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-2"

		mock := &MockConnectionClient{
			CreateFunc: func(_ *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
				return nil, errors.New("plan gate")
			},
		}
		_, err := NewHandler(mock).Update(context.Background(), localID, data, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `recreating rETL connection "users-to-webhook" after an immutable change (the previous connection was deleted): creating rETL connection "users-to-webhook": plan gate`)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("deletes only the connection", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		require.NoError(t, NewHandler(mock).Delete(context.Background(), localID, stateData(t, jsonMapperConfig())))

		assert.Equal(t, &MockConnectionClient{DeleteCalls: []string{"conn-remote-1"}}, mock)
	})

	t.Run("errors when state lacks the remote id", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{}
		err := NewHandler(mock).Delete(context.Background(), localID, resources.ResourceData{})

		assert.ErrorContains(t, err, "missing id in state")
		assert.Empty(t, mock.DeleteCalls)
	})

	t.Run("surfaces server errors", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{DeleteFunc: func(_ string) error { return errors.New("forbidden") }}
		err := NewHandler(mock).Delete(context.Background(), localID, stateData(t, jsonMapperConfig()))

		assert.EqualError(t, err, `deleting rETL connection "users-to-webhook": forbidden`)
	})
}
