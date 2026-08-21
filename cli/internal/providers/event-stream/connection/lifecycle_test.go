package connection

import (
	"context"
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// derefData is what the syncer hands the lifecycle after state.Dereference:
// both endpoint refs already resolved to their remote ids.
func derefData(sourceID, destinationID string, enabled bool) resources.ResourceData {
	return resources.ResourceData{
		SourceKey:      sourceID,
		DestinationKey: destinationID,
		EnabledKey:     enabled,
	}
}

// stateData is what the syncer hands Update/Delete: the prior ResourceState's
// Input and Output merged. Input still holds the un-dereferenced refs (elided
// here); Output holds the remote identifiers the lifecycle wrote.
func stateData(remoteID, sourceID, destinationID string, enabled bool) resources.ResourceData {
	return resources.ResourceData{
		EnabledKey:       enabled,
		IDKey:            remoteID,
		SourceIDKey:      sourceID,
		DestinationIDKey: destinationID,
	}
}

func TestCreate(t *testing.T) {
	t.Run("creates the connection with externalId attached", func(t *testing.T) {
		mock := &MockConnectionClient{
			CreateFunc: func(conn *client.Connection) (*client.Connection, error) {
				created := *conn
				created.ID = "conn-remote-1"
				return &created, nil
			},
		}
		h := NewHandler(mock, "event-stream")

		output, err := h.Create(context.Background(), "android-to-s3", derefData("src-remote-1", "dst-remote-1", true))

		require.NoError(t, err)
		require.Len(t, mock.CreateCalls, 1)
		assert.Equal(t, client.Connection{
			ExternalID:    "android-to-s3",
			SourceID:      "src-remote-1",
			DestinationID: "dst-remote-1",
			IsEnabled:     true,
		}, mock.CreateCalls[0])
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-remote-1",
			DestinationIDKey: "dst-remote-1",
		}, output)
	})

	t.Run("surfaces server errors with externalId", func(t *testing.T) {
		mock := &MockConnectionClient{
			CreateFunc: func(_ *client.Connection) (*client.Connection, error) {
				return nil, errors.New("plan does not allow more connections")
			},
		}
		h := NewHandler(mock, "event-stream")

		_, err := h.Create(context.Background(), "android-to-s3", derefData("src-remote-1", "dst-remote-1", true))

		require.Error(t, err)
		assert.EqualError(t, err, `creating event stream connection "android-to-s3": plan does not allow more connections`)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("updates enabled in place", func(t *testing.T) {
		mock := &MockConnectionClient{}
		h := NewHandler(mock, "event-stream")

		output, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-1", "dst-remote-1", false),
			stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		require.NoError(t, err)
		assert.Equal(t, []string{"UpdateConnection"}, mock.Calls)
		require.Len(t, mock.UpdateCalls, 1)
		assert.Equal(t, "conn-remote-1", mock.UpdateCalls[0].ID)
		assert.False(t, mock.UpdateCalls[0].IsEnabled)
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-remote-1",
			DestinationIDKey: "dst-remote-1",
		}, output)
	})

	t.Run("no-op when nothing changed", func(t *testing.T) {
		mock := &MockConnectionClient{}
		h := NewHandler(mock, "event-stream")

		output, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-1", "dst-remote-1", true),
			stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		require.NoError(t, err)
		assert.Empty(t, mock.Calls)
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-remote-1",
			DestinationIDKey: "dst-remote-1",
		}, output)
	})

	t.Run("destination change replaces the connection", func(t *testing.T) {
		mock := &MockConnectionClient{
			CreateFunc: func(conn *client.Connection) (*client.Connection, error) {
				created := *conn
				created.ID = "conn-remote-2"
				return &created, nil
			},
		}
		h := NewHandler(mock, "event-stream")

		output, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-1", "dst-remote-9", true),
			stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		require.NoError(t, err)
		assert.Equal(t, []string{"DeleteConnection", "CreateConnection"}, mock.Calls)
		assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls)
		require.Len(t, mock.CreateCalls, 1)
		assert.Equal(t, client.Connection{
			ExternalID:    "android-to-s3",
			SourceID:      "src-remote-1",
			DestinationID: "dst-remote-9",
			IsEnabled:     true,
		}, mock.CreateCalls[0])
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-2",
			SourceIDKey:      "src-remote-1",
			DestinationIDKey: "dst-remote-9",
		}, output)
	})

	t.Run("source change replaces the connection", func(t *testing.T) {
		mock := &MockConnectionClient{}
		h := NewHandler(mock, "event-stream")

		_, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-9", "dst-remote-1", true),
			stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		require.NoError(t, err)
		assert.Equal(t, []string{"DeleteConnection", "CreateConnection"}, mock.Calls)
	})

	t.Run("replacement keeps the revived row's remote id", func(t *testing.T) {
		// The backend revives the soft-deleted row when the same
		// source–destination pair is recreated: the create response carries the
		// OLD remote id, and that id — not a presumed-fresh one — must land in
		// the output.
		mock := &MockConnectionClient{
			CreateFunc: func(conn *client.Connection) (*client.Connection, error) {
				created := *conn
				created.ID = "conn-remote-1"
				return &created, nil
			},
		}
		h := NewHandler(mock, "event-stream")

		output, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-1", "dst-remote-1", true),
			stateData("conn-remote-1", "src-remote-1", "dst-remote-9", true))

		require.NoError(t, err)
		assert.Equal(t, []string{"DeleteConnection", "CreateConnection"}, mock.Calls)
		assert.Equal(t, "conn-remote-1", (*output)[IDKey])
	})

	t.Run("replacement stops when delete fails", func(t *testing.T) {
		mock := &MockConnectionClient{
			DeleteFunc: func(_ string) error {
				return errors.New("forbidden")
			},
		}
		h := NewHandler(mock, "event-stream")

		_, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-1", "dst-remote-9", true),
			stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		require.Error(t, err)
		assert.ErrorContains(t, err, "forbidden")
		assert.Equal(t, []string{"DeleteConnection"}, mock.Calls)
	})

	t.Run("says the old connection is gone when the recreate fails", func(t *testing.T) {
		mock := &MockConnectionClient{
			CreateFunc: func(_ *client.Connection) (*client.Connection, error) {
				return nil, errors.New("plan gate")
			},
		}
		h := NewHandler(mock, "event-stream")

		_, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-1", "dst-remote-9", true),
			stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		require.Error(t, err)
		assert.Equal(t, []string{"DeleteConnection", "CreateConnection"}, mock.Calls)
		assert.ErrorContains(t, err, "the previous connection was deleted")
		assert.ErrorContains(t, err, "plan gate")
	})

	t.Run("errors when state lacks the remote id", func(t *testing.T) {
		h := NewHandler(&MockConnectionClient{}, "event-stream")

		_, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-1", "dst-remote-1", true),
			resources.ResourceData{})

		require.Error(t, err)
		assert.ErrorContains(t, err, "missing id in state")
	})

	t.Run("surfaces server errors with externalId", func(t *testing.T) {
		mock := &MockConnectionClient{
			UpdateFunc: func(_ *client.Connection) (*client.Connection, error) {
				return nil, errors.New("plan gate")
			},
		}
		h := NewHandler(mock, "event-stream")

		_, err := h.Update(context.Background(), "android-to-s3",
			derefData("src-remote-1", "dst-remote-1", false),
			stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		assert.EqualError(t, err, `updating event stream connection "android-to-s3": plan gate`)
	})
}

func TestDelete(t *testing.T) {
	t.Run("deletes only the connection", func(t *testing.T) {
		mock := &MockConnectionClient{}
		h := NewHandler(mock, "event-stream")

		err := h.Delete(context.Background(), "android-to-s3", stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		require.NoError(t, err)
		assert.Equal(t, []string{"DeleteConnection"}, mock.Calls)
		assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls)
	})

	t.Run("errors when state lacks the remote id", func(t *testing.T) {
		h := NewHandler(&MockConnectionClient{}, "event-stream")

		err := h.Delete(context.Background(), "android-to-s3", resources.ResourceData{})

		require.Error(t, err)
		assert.ErrorContains(t, err, "missing id in state")
	})

	t.Run("surfaces server errors with externalId", func(t *testing.T) {
		mock := &MockConnectionClient{
			DeleteFunc: func(_ string) error {
				return errors.New("forbidden")
			},
		}
		h := NewHandler(mock, "event-stream")

		err := h.Delete(context.Background(), "android-to-s3", stateData("conn-remote-1", "src-remote-1", "dst-remote-1", true))

		assert.EqualError(t, err, `deleting event stream connection "android-to-s3": forbidden`)
	})
}
