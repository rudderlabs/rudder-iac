package accounts

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

// AccountConfig is the single flat `config` block a user writes. The handler
// routes each field to the API's `options` (non-secret) vs `secret` per the
// account definition's secret-field set. For BigQuery the only secret field is
// `credentials`; every other key is carried in Options via mapstructure's
// ",remain".
type AccountConfig struct {
	Credentials secret.String  `mapstructure:"credentials"`
	Options     map[string]any `mapstructure:",remain"`
}

// AccountItem is one account in the `kind: accounts` collection.
type AccountItem struct {
	ID                    string        `mapstructure:"id"`
	AccountDefinitionName string        `mapstructure:"accountDefinitionName"`
	Config                AccountConfig `mapstructure:"config"`
}

// AccountSpec is the decoded `kind: accounts` spec — one file declares one or
// more accounts.
type AccountSpec struct {
	Accounts []AccountItem `mapstructure:"accounts"`
}

// AccountResource is the RawData (differ input) for a single account. The secret
// is carried as *secret.String so the value survives the differ's struct→map
// decode (mirrors the framework's example provider).
type AccountResource struct {
	ID                    string         `json:"id"`
	AccountDefinitionName string         `json:"accountDefinitionName"`
	Options               map[string]any `json:"options"`
	Credentials           *secret.String `json:"credentials"`
}

// AccountState is the computed output — the remote account id.
type AccountState struct {
	ID string
}

// RemoteAccount wraps the API account to implement the RemoteResource interface.
type RemoteAccount struct {
	*client.Account
}

// Metadata implements the RemoteResource interface.
func (r RemoteAccount) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: r.WorkspaceID,
		Name:        r.Name,
	}
}
