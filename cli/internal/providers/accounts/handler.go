package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

// bigQueryDefinition is the only account definition supported this milestone
// (BigQuery-only scope). Other warehouses land in the "Post Bigquery" work.
const bigQueryDefinition = "SOURCE_BIGQUERY"

// bigQuerySecretField is the sole secret field in a BigQuery account's config.
// Sourcing this dynamically from the account definition is a separate follow-up.
const bigQuerySecretField = "credentials"

var (
	screamingSnakeCase = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	validExternalID    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
)

// AccountStore is the subset of the accounts API client the handler needs. It is
// declared at the point of use so tests can supply a mock. The real client's
// `*client.Client.Accounts` satisfies it.
type AccountStore interface {
	Create(ctx context.Context, req *client.CreateAccountRequest) (*client.Account, error)
	Update(ctx context.Context, id string, req *client.UpdateAccountRequest) (*client.Account, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*client.Account, error)
	ListAll(ctx context.Context, opts ...client.ListAccountsOption) ([]client.Account, error)
	SetExternalID(ctx context.Context, id string, externalID string) error
}

// AccountHandler is the concrete BaseHandler for warehouse accounts.
type AccountHandler = handler.BaseHandler[AccountSpec, AccountResource, AccountState, RemoteAccount]

// HandlerMetadata identifies the account resource type and its spec kind.
var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "account",
	SpecKind:         "accounts",
	SpecMetadataName: "accounts",
}

// HandlerImpl provides the account-specific pieces the BaseHandler composes.
// It embeds the single-spec export strategy so export (one `kind: accounts`
// file) and generic secret tokenization come from the framework; the strategy
// also carries the declared secret fields BaseHandler injects via SetSecretFields.
type HandlerImpl struct {
	*export.SingleSpecExportStrategy[accountsExportSpec, RemoteAccount]
	store AccountStore
}

// NewHandler builds the account BaseHandler around the given store.
func NewHandler(store AccountStore) *AccountHandler {
	h := &HandlerImpl{store: store}
	h.SingleSpecExportStrategy = &export.SingleSpecExportStrategy[accountsExportSpec, RemoteAccount]{Handler: h}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *AccountSpec {
	return &AccountSpec{}
}

// ExtractResourcesFromSpec parses the `spec.accounts[]` collection into one
// AccountResource per item keyed by its `id`, validating each item and rejecting
// duplicate ids within the spec.
func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *AccountSpec) (map[string]*AccountResource, error) {
	res := make(map[string]*AccountResource, len(spec.Accounts))
	for i := range spec.Accounts {
		item := spec.Accounts[i]
		if err := validateAccountItem(item); err != nil {
			return nil, fmt.Errorf("account %q in %s: %w", item.ID, path, err)
		}
		if _, dup := res[item.ID]; dup {
			return nil, fmt.Errorf("duplicate account id %q in %s", item.ID, path)
		}

		credentials := item.Config.Credentials
		res[item.ID] = &AccountResource{
			ID:                    item.ID,
			AccountDefinitionName: item.AccountDefinitionName,
			Options:               item.Config.Options,
			Credentials:           &credentials,
		}
	}
	return res, nil
}

// validateAccountItem enforces the CLI-side rules (CONTRACT-ACCT-STATE-V1 §6.1):
// a valid stable id, and an accountDefinitionName that is SCREAMING_SNAKE_CASE
// and resolves to a supported (BigQuery) definition.
func validateAccountItem(item AccountItem) error {
	if item.ID == "" {
		return fmt.Errorf("id is required")
	}
	if !validExternalID.MatchString(item.ID) {
		return fmt.Errorf("id %q is not a valid external-id token", item.ID)
	}
	if item.AccountDefinitionName == "" {
		return fmt.Errorf("accountDefinitionName is required")
	}
	if !screamingSnakeCase.MatchString(item.AccountDefinitionName) {
		return fmt.Errorf("accountDefinitionName %q must be SCREAMING_SNAKE_CASE", item.AccountDefinitionName)
	}
	if item.AccountDefinitionName != bigQueryDefinition {
		return fmt.Errorf("accountDefinitionName %q is not supported (only %s this milestone)", item.AccountDefinitionName, bigQueryDefinition)
	}
	return nil
}

// Create provisions the account, then claims the spec id as its externalId so the
// next apply reconciles it instead of recreating it.
func (h *HandlerImpl) Create(ctx context.Context, data *AccountResource) (*AccountState, error) {
	options, secretPayload, err := splitConfig(data)
	if err != nil {
		return nil, err
	}

	created, err := h.store.Create(ctx, &client.CreateAccountRequest{
		AccountDefinitionName: data.AccountDefinitionName,
		Name:                  data.ID,
		Options:               options,
		Secret:                secretPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("creating account %q: %w", data.ID, err)
	}

	if err := h.store.SetExternalID(ctx, created.ID, data.ID); err != nil {
		return nil, fmt.Errorf("claiming external id for account %q: %w", data.ID, err)
	}

	return &AccountState{ID: created.ID}, nil
}

// Update full-replaces the account. accountDefinitionName is immutable (identity
// and resource are separate concerns), so a change is rejected up front.
func (h *HandlerImpl) Update(ctx context.Context, newData *AccountResource, oldData *AccountResource, oldState *AccountState) (*AccountState, error) {
	if newData.AccountDefinitionName != oldData.AccountDefinitionName {
		return nil, fmt.Errorf(
			"accountDefinitionName is immutable for account %q (was %q, got %q)",
			newData.ID, oldData.AccountDefinitionName, newData.AccountDefinitionName,
		)
	}

	options, secretPayload, err := splitConfig(newData)
	if err != nil {
		return nil, err
	}

	updated, err := h.store.Update(ctx, oldState.ID, &client.UpdateAccountRequest{
		Name:    newData.ID,
		Options: options,
		Secret:  secretPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("updating account %q: %w", newData.ID, err)
	}

	return &AccountState{ID: updated.ID}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *AccountResource, oldState *AccountState) error {
	if err := h.store.Delete(ctx, oldState.ID); err != nil {
		return fmt.Errorf("deleting account %q: %w", id, err)
	}
	return nil
}

// LoadRemoteResources returns the accounts this project manages: those with an
// externalId, scoped to the supported (BigQuery) definition.
func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*RemoteAccount, error) {
	accounts, err := h.store.ListAll(ctx, client.WithHasExternalID(true))
	if err != nil {
		return nil, fmt.Errorf("listing managed accounts: %w", err)
	}
	return supportedRemoteAccounts(accounts), nil
}

// LoadImportableResources returns unmanaged accounts (no externalId) that are
// candidates for import, scoped to the supported (BigQuery) definition. Filtering
// keys on the account definition, not category (category is in flux — §9.2).
func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*RemoteAccount, error) {
	accounts, err := h.store.ListAll(ctx, client.WithHasExternalID(false))
	if err != nil {
		return nil, fmt.Errorf("listing importable accounts: %w", err)
	}
	return supportedRemoteAccounts(accounts), nil
}

// MapRemoteToState keys the resource by externalId and marks the secret unknown —
// the API never returns secret values, so credentials always diff and force a
// re-send. BaseHandler additionally scrubs the declared credentials field.
func (h *HandlerImpl) MapRemoteToState(remote *RemoteAccount, urnResolver handler.URNResolver) (*AccountResource, *AccountState, error) {
	if remote.ExternalID == "" {
		return nil, nil, nil // unmanaged — skip (framework convention)
	}

	var options map[string]any
	if len(remote.Options) > 0 {
		if err := json.Unmarshal(remote.Options, &options); err != nil {
			return nil, nil, fmt.Errorf("unmarshalling options for account %s: %w", remote.ID, err)
		}
	}

	unknown := secret.NewUnknown()
	res := &AccountResource{
		ID:                    remote.ExternalID,
		AccountDefinitionName: remote.Definition.Name,
		Options:               options,
		Credentials:           &unknown,
	}
	state := &AccountState{ID: remote.ID}
	return res, state, nil
}

// Import claims an existing remote account for this project: it links the remote
// to the spec id via externalId, then full-replaces the account with the spec's
// config so the imported resource immediately matches what the user declared. The
// next apply then reconciles it like any managed account.
func (h *HandlerImpl) Import(ctx context.Context, data *AccountResource, remoteId string) (*AccountState, error) {
	remote, err := h.store.Get(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting account %s: %w", remoteId, err)
	}

	if err := h.store.SetExternalID(ctx, remote.ID, data.ID); err != nil {
		return nil, fmt.Errorf("claiming external id for account %q: %w", data.ID, err)
	}

	options, secretPayload, err := splitConfig(data)
	if err != nil {
		return nil, err
	}

	updated, err := h.store.Update(ctx, remote.ID, &client.UpdateAccountRequest{
		Name:    data.ID,
		Options: options,
		Secret:  secretPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("updating account %q: %w", data.ID, err)
	}

	return &AccountState{ID: updated.ID}, nil
}

// MapRemoteToSpec builds the single `kind: accounts` spec from the managed
// remotes, one item per account keyed by externalId. Credentials is a placeholder
// (a remote never returns the real value); the framework's AttachSecretVariables
// replaces it with a per-account "{{ .VAR }}" token before the spec is written.
func (h *HandlerImpl) MapRemoteToSpec(
	remotes map[string]*RemoteAccount,
	_ resolver.ReferenceResolver,
) (*export.SpecExportData[accountsExportSpec], error) {
	items := make([]accountExportItem, 0, len(remotes))
	for externalID, remote := range remotes {
		var options map[string]any
		if len(remote.Options) > 0 {
			if err := json.Unmarshal(remote.Options, &options); err != nil {
				return nil, fmt.Errorf("unmarshalling options for account %s: %w", remote.ID, err)
			}
		}

		items = append(items, accountExportItem{
			ID:                    externalID,
			AccountDefinitionName: remote.Definition.Name,
			Config: accountExportConfig{
				Credentials: secret.NewUnknown(),
				Options:     options,
			},
		})
	}

	return &export.SpecExportData[accountsExportSpec]{
		RelativePath: "accounts/accounts.yaml",
		Data:         &accountsExportSpec{Accounts: items},
	}, nil
}

// splitConfig routes the flat config into the API's options (non-secret) and
// secret payloads. For BigQuery the only secret field is `credentials`.
func splitConfig(data *AccountResource) (json.RawMessage, json.RawMessage, error) {
	options := data.Options
	if options == nil {
		options = map[string]any{}
	}
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling options for account %q: %w", data.ID, err)
	}

	secretJSON, err := json.Marshal(map[string]string{bigQuerySecretField: revealCredentials(data)})
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling secret for account %q: %w", data.ID, err)
	}
	return optionsJSON, secretJSON, nil
}

func revealCredentials(data *AccountResource) string {
	if data.Credentials == nil {
		return ""
	}
	return data.Credentials.Reveal()
}

// supportedRemoteAccounts wraps and filters remote accounts to the supported
// (BigQuery) definition.
func supportedRemoteAccounts(accounts []client.Account) []*RemoteAccount {
	result := make([]*RemoteAccount, 0, len(accounts))
	for i := range accounts {
		account := accounts[i]
		if account.Definition.Name != bigQueryDefinition {
			continue
		}
		result = append(result, &RemoteAccount{Account: &account})
	}
	return result
}
