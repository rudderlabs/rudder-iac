package provider

import (
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// retlRemedy names the experimental flags because with the rETL connection flag
// off the CLI never loads its own connections, so it cannot plan their deletion
// first and the refusal names nothing in the plan. Both env vars appear because
// GetConfig ignores every experimental flag unless the umbrella switch is on.
const retlRemedy = "If they are rETL connections the CLI created, re-run with " +
	"RUDDERSTACK_CLI_EXPERIMENTAL=true RUDDERSTACK_X_RETL_CONNECTION_SUPPORT=true; " +
	"otherwise delete them in the workspace first"

const workspaceRemedy = "Delete them in the workspace first"

// ExplainBlockingConnections annotates the control plane's refusal to delete an
// endpoint an rETL connection can join — a destination, or an rETL source.
//
// The remedy lives here rather than beside the predicate in api/client because
// it names one provider's experimental flag; api/client has no business knowing
// that flag exists.
func ExplainBlockingConnections(err error) error {
	return explainBlockingConnections(err, retlRemedy)
}

// ExplainBlockingEventStreamConnections is the same for an event-stream source,
// which no rETL connection can join — retl/connection.SourceKinds is sqlModel
// plus table. Offering the rETL flags there would send the reader after flags
// that cannot help, so only the workspace remedy applies.
func ExplainBlockingEventStreamConnections(err error) error {
	return explainBlockingConnections(err, workspaceRemedy)
}

func explainBlockingConnections(err error, remedy string) error {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || !apiErr.BlockedByConnections() {
		return err
	}

	return fmt.Errorf("%w: connections this run is not managing are never removed with it. %s", err, remedy)
}
