package provider

import (
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

// retlSourceFlags are the experimental flags that put the rETL source kinds in
// scope for a run. Rendered through config.GetEnvironmentVariableName so they
// cannot drift from the flags they name. Both the umbrella switch and the kind's
// own flag are needed, because GetConfig clears every experimental flag unless
// the umbrella is also on.
func retlSourceFlags() string {
	return fmt.Sprintf(
		"RUDDERSTACK_CLI_EXPERIMENTAL=true %s=true %s=true",
		config.GetEnvironmentVariableName("retlTableSupport"),
		config.GetEnvironmentVariableName("retlConnectionSupport"),
	)
}

// ExplainBlockingAccountUsage annotates the control plane's refusal to delete an
// account that something still uses.
//
// The dependent is frequently invisible to the run that tripped it. A rETL
// source whose kind sits behind an experimental flag is never loaded, so it
// appears in no plan and the refusal names nothing the user can see; a source
// created in the webapp is outside the project entirely. Either way the CLI is
// better placed than the API to say what to do next.
//
// Sibling of ExplainBlockingConnections, which does the same for a source or
// destination delete blocked by connections. The backend's own reason is
// preserved and still unwraps — the annotation is added, not substituted.
func ExplainBlockingAccountUsage(err error) error {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || !apiErr.BlockedByAccountUsage() {
		return err
	}

	return fmt.Errorf(
		"%w: resources this run is not managing still use this account. If they are rETL sources the CLI created, "+
			"re-run with %s; otherwise delete them in the workspace first",
		err, retlSourceFlags())
}
