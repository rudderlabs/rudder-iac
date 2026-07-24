package accounts

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
)

// AccountSpec is the user-authored YAML representation of an account. Config is a
// single flat map; the handler splits it into the API's options (non-secret) and
// secret payloads by the account definition's secret-key set — the same shape a
// destination spec uses (one config map, secrets identified by key).
type AccountSpec struct {
	ID                    string         `mapstructure:"id" validate:"required"`
	AccountDefinitionName string         `mapstructure:"account_definition_name" validate:"required"`
	Config                map[string]any `mapstructure:"config"`
}

// AccountResource is the resolved representation the differ compares. Registered
// secret keys inside Config are wrapped as *secret.String so the differ's
// secret-aware branch owns their comparison — identical to DestinationResource.
type AccountResource struct {
	ID                    string
	AccountDefinitionName string
	Config                map[string]any
}

// AccountState is the persisted apply-cycle state — the remote account ID.
type AccountState struct {
	ID string
}

// RemoteAccount wraps client.Account to satisfy handler.RemoteResource.
type RemoteAccount struct {
	*client.Account
}

// Metadata exposes the identifying fields BaseHandler keys the remote collection
// on.
func (r RemoteAccount) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: r.WorkspaceID,
		Name:        r.Name,
	}
}
