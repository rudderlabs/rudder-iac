package accounts

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
)

func TestRemoteAccountMetadata(t *testing.T) {
	r := RemoteAccount{Account: &client.Account{
		ID:          "remote-id",
		ExternalID:  "lovable-prod-bq",
		WorkspaceID: "ws-1",
		Name:        "prod",
	}}

	m := r.Metadata()
	assert.Equal(t, "remote-id", m.ID)
	assert.Equal(t, "lovable-prod-bq", m.ExternalID)
	assert.Equal(t, "ws-1", m.WorkspaceID)
	assert.Equal(t, "prod", m.Name)
}
