package provider

import (
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
)

const workspaceRemedy = "Delete them in the workspace first"

// ExplainBlockingConnections annotates the control plane's refusal to delete an
// endpoint an rETL connection can join — a destination, or an rETL source.
//
// The CLI plans the deletion of connections it manages, so what still blocks the
// delete is a connection outside this project.
func ExplainBlockingConnections(err error) error {
	return explainBlockingConnections(err, workspaceRemedy)
}

// ExplainBlockingEventStreamConnections is the same for an event-stream source,
// which no rETL connection can join: retl/connection.SourceKinds is sqlModel
// plus table.
func ExplainBlockingEventStreamConnections(err error) error {
	return explainBlockingConnections(err, workspaceRemedy)
}

func explainBlockingConnections(err error, remedy string) error {
	return explainBlocked(err, (*client.APIError).BlockedByConnections,
		"connections this run is not managing are never removed with it", remedy)
}

// explainBlocked appends the reason and remedy to an API refusal that blocked
// recognizes. The backend's own message stays first and still unwraps.
func explainBlocked(err error, blocked func(*client.APIError) bool, reason, remedy string) error {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || !blocked(apiErr) {
		return err
	}

	return fmt.Errorf("%w: %s. %s", err, reason, remedy)
}
