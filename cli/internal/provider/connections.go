package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// ExplainBlockingConnections annotates the backend's refusal to delete a
// source or destination that still has connections. With the rETL connection
// flag off, the CLI never loads its own connections, so it cannot plan their
// deletion first and the refusal names nothing in the plan. Both env vars are
// named because GetConfig ignores every experimental flag unless the umbrella
// switch is also on.
func ExplainBlockingConnections(err error) error {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) ||
		apiErr.HTTPStatusCode != http.StatusBadRequest ||
		!strings.Contains(strings.ToLower(apiErr.Msg()), "active connections") {
		return err
	}

	return fmt.Errorf(
		"%w: connections this run is not managing are never removed with it. If they are rETL connections the CLI created, "+
			"re-run with RUDDERSTACK_CLI_EXPERIMENTAL=true RUDDERSTACK_X_RETL_CONNECTION_SUPPORT=true; otherwise delete them in the workspace first",
		err)
}
