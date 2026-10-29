package connection

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisabledConnectionBackendNoOpIsReported(t *testing.T) {
	for _, method := range []string{"create", "update"} {
		t.Run(method, func(t *testing.T) {
			mock := &MockConnectionClient{
				CreateFunc: func(conn *client.Connection) (*client.Connection, error) {
					created := *conn
					created.ID = "conn-1"
					created.IsEnabled = true // config backend ignores enabled=false
					return &created, nil
				},
				UpdateFunc: func(conn *client.Connection) (*client.Connection, error) {
					updated := *conn
					updated.IsEnabled = true
					return &updated, nil
				},
			}
			h := NewHandler(mock, "event-stream")
			var err error
			if method == "create" {
				_, err = h.Create(context.Background(), "local", derefData("src", "dst", false))
			} else {
				_, err = h.Update(context.Background(), "local", derefData("src", "dst", false), stateData("conn-1", "src", "dst", true))
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, "enabled")
		})
	}
}
