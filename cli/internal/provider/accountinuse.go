package provider

import "github.com/rudderlabs/rudder-iac/api/client"

// ExplainBlockingAccountUsage annotates the control plane's refusal to delete an
// account that something still uses. The user cannot always see the dependent
// in the plan, so the note names what to do with it. The backend's reason stays
// first and still unwraps, as in ExplainBlockingConnections.
func ExplainBlockingAccountUsage(err error) error {
	return explainBlocked(err, (*client.APIError).BlockedByAccountUsage,
		"other resources still use this account",
		"Point them at another account or remove them from the project")
}
