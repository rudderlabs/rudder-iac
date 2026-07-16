package accounts

import (
	"encoding/json"

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
	// secret:"true" opts credentials into the BaseHandler secret-field support
	// (DEX-457): it is scrubbed from remote state (always-diff) and exported as a
	// {{ .VAR }} token rather than a literal.
	Credentials *secret.String `json:"credentials" secret:"true"`
}

// AccountState is the computed output — the remote account id.
type AccountState struct {
	ID string
}

// --- Export spec shapes ---
//
// Export uses a struct (not the decode-side AccountSpec/AccountConfig, which
// carry only mapstructure tags) so the framework's secret reflection
// (AttachSecretVariables) can discover the credentials field by its `json`
// name and type `secret.String`, replacing it with a "{{ .VAR }}" token. Item
// ids carry a `json:"id"` tag too, so each account gets its own variable name.

// accountsExportSpec is the `kind: accounts` spec produced on export — the
// collection of accounts, mirroring AccountSpec but json-tagged for
// serialization and secret discovery.
type accountsExportSpec struct {
	Accounts []accountExportItem `json:"accounts"`
}

// accountExportItem is one exported account. `id` is json-tagged so the secret
// reflection derives a per-account variable name from it.
type accountExportItem struct {
	ID                    string              `json:"id"`
	AccountDefinitionName string              `json:"accountDefinitionName"`
	Config                accountExportConfig `json:"config"`
}

// accountExportConfig serializes back to the flat `config` block a user writes:
// the secret `credentials` field alongside every non-secret option key. It
// keeps Credentials as a named `secret.String` field (not a map value) so the
// framework can find and tokenize it; Options is merged in by MarshalJSON since
// encoding/json has no inline-map support.
type accountExportConfig struct {
	Credentials secret.String  `json:"credentials"`
	Options     map[string]any `json:"-"`
}

// MarshalJSON flattens Options and Credentials into a single object so the
// exported `config` round-trips to the flat form the decoder expects. Credentials
// serializes through secret.String.MarshalJSON, emitting the "{{ .VAR }}" token
// (or a masked literal when the substitution gate is off) — never the real value.
func (c accountExportConfig) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, len(c.Options)+1)
	for k, v := range c.Options {
		out[k] = v
	}
	out["credentials"] = c.Credentials
	return json.Marshal(out)
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
