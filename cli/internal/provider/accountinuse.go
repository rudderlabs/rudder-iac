package provider

import "github.com/rudderlabs/rudder-iac/api/client"

// ExplainBlockingAccountUsage annotates the control plane's refusal to delete an
// account that something still uses.
//
// The dependent is frequently invisible to the run that tripped it: a source
// created in the webapp is outside the project entirely, and the refusal names
// nothing the user can see in the plan. No flag helps with that, so the remedy
// is the two routes that do.
//
// Sibling of ExplainBlockingConnections, which does the same for a source or
// destination delete blocked by connections. The backend's own reason is
// preserved and still unwraps; the annotation is added, not substituted.
func ExplainBlockingAccountUsage(err error) error {
	return explainBlocked(err, (*client.APIError).BlockedByAccountUsage,
		"resources this run is not managing still use this account",
		"If they are data graphs the CLI manages, remove them from the project in the same run; otherwise delete them in the workspace first")
}
