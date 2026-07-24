package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

// AccountHandler is the BaseHandler instantiation for accounts.
type AccountHandler = handler.BaseHandler[AccountSpec, AccountResource, AccountState, RemoteAccount]

// HandlerMetadata describes the account handler for the framework.
var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     AccountResourceType,
	SpecKind:         AccountSpecKind,
	SpecMetadataName: AccountMetadataName,
}

// accountSecretKeys maps an account definition to its secret field set — the
// account-side analogue of a destination definition's SecretKeys().
//
// ponytail: BigQuery-only, hardcoded. The real registry fetches secretFields
// from the control-plane account-definitions API (unversioned, name-keyed).
// Add a lookup when a second definition lands; the split logic below is already
// definition-driven, so only this map changes.
var accountSecretKeys = map[string][]string{
	"SOURCE_BIGQUERY": {"credentials"},
}

// AccountStore is the subset of the accounts API client the handler needs;
// declared at the point of use so tests inject a mock. *client.Client.Accounts
// satisfies it.
type AccountStore interface {
	Create(ctx context.Context, req *client.CreateAccountRequest) (*client.Account, error)
	Update(ctx context.Context, id string, req *client.UpdateAccountRequest) (*client.Account, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*client.Account, error)
	ListAll(ctx context.Context, opts ...client.ListAccountsOption) ([]client.Account, error)
	SetExternalID(ctx context.Context, id, externalID string) error
}

// HandlerImpl owns account CRUD against the API client.
type HandlerImpl struct {
	store AccountStore
}

// NewHandler builds an *AccountHandler wired to the given store.
func NewHandler(store AccountStore) *AccountHandler {
	return handler.NewHandler(&HandlerImpl{store: store})
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata { return HandlerMetadata }

func (h *HandlerImpl) NewSpec() *AccountSpec { return &AccountSpec{} }

// ExtractResourcesFromSpec resolves the account definition's secret keys and
// wraps them in Config as *secret.String — mirrors the destination handler,
// minus the (type, version) registry lookup (account definitions are
// unversioned).
func (h *HandlerImpl) ExtractResourcesFromSpec(_ string, spec *AccountSpec) (map[string]*AccountResource, error) {
	keys, ok := accountSecretKeys[spec.AccountDefinitionName]
	if !ok {
		return nil, fmt.Errorf("unsupported account definition %q", spec.AccountDefinitionName)
	}
	resource := &AccountResource{
		ID:                    spec.ID,
		AccountDefinitionName: spec.AccountDefinitionName,
		Config:                secret.WrapKnownSecrets(spec.Config, keys),
	}
	return map[string]*AccountResource{spec.ID: resource}, nil
}

// Create provisions the account, then claims the spec ID as its externalId so
// the next apply reconciles instead of recreating. External ID is set last so a
// failed claim never leaves a partially-adopted resource behind (mirrors the
// destination handler's Import ordering).
func (h *HandlerImpl) Create(ctx context.Context, data *AccountResource) (*AccountState, error) {
	options, secretPayload, err := h.splitConfig(data)
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

// Update rejects an immutable definition change and full-replaces the account
// (PUT is REST-strict — a missing field means set-to-empty).
func (h *HandlerImpl) Update(ctx context.Context, newData *AccountResource, oldData *AccountResource, oldState *AccountState) (*AccountState, error) {
	if newData.AccountDefinitionName != oldData.AccountDefinitionName {
		return nil, fmt.Errorf("account definition change is not supported: old %q, new %q", oldData.AccountDefinitionName, newData.AccountDefinitionName)
	}

	options, secretPayload, err := h.splitConfig(newData)
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

func (h *HandlerImpl) Delete(ctx context.Context, _ string, _ *AccountResource, oldState *AccountState) error {
	if err := h.store.Delete(ctx, oldState.ID); err != nil {
		return fmt.Errorf("deleting account %q: %w", oldState.ID, err)
	}
	return nil
}

// MapRemoteToState rebuilds the flat config from the remote options and marks
// every secret key unknown (the API never returns secret values), so the differ
// flags them SecretOnly rather than phantom drift — same rule as destinations.
func (h *HandlerImpl) MapRemoteToState(remote *RemoteAccount, _ handler.URNResolver) (*AccountResource, *AccountState, error) {
	if remote.ExternalID == "" {
		return nil, nil, fmt.Errorf("managed account %s has empty external ID", remote.ID)
	}

	keys, ok := accountSecretKeys[remote.Definition.Name]
	if !ok {
		return nil, nil, fmt.Errorf("managed account %s has unsupported definition %q", remote.ID, remote.Definition.Name)
	}

	config, err := unmarshalOptions(remote.Options)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshalling options for account %s: %w", remote.ID, err)
	}
	// The API never returns the secret, so it is absent from remote options. Seed
	// each secret key so the presence-based WrapUnknownSecrets marks it unknown —
	// account secrets are unconditional (unlike a destination's optional secrets),
	// so they must always be present-and-unknown and therefore always re-applied.
	for _, key := range keys {
		if _, ok := config[key]; !ok {
			config[key] = ""
		}
	}
	config = secret.WrapUnknownSecrets(config, keys)

	resource := &AccountResource{
		ID:                    remote.ExternalID,
		AccountDefinitionName: remote.Definition.Name,
		Config:                config,
	}
	return resource, &AccountState{ID: remote.ID}, nil
}

// LoadRemoteResources returns managed accounts (ExternalID set) of a supported
// definition.
func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*RemoteAccount, error) {
	all, err := h.store.ListAll(ctx, client.WithHasExternalID(true))
	if err != nil {
		return nil, fmt.Errorf("listing managed accounts: %w", err)
	}
	return supportedRemoteAccounts(all), nil
}

// LoadImportableResources returns unmanaged accounts (no ExternalID) of a
// supported definition.
func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*RemoteAccount, error) {
	all, err := h.store.ListAll(ctx, client.WithHasExternalID(false))
	if err != nil {
		return nil, fmt.Errorf("listing importable accounts: %w", err)
	}
	return supportedRemoteAccounts(all), nil
}

// Import adopts an existing remote account: it pushes the spec via Update (same
// reconciliation path as apply — DRY), then sets the external ID last. Mirrors
// the destination handler's Import.
func (h *HandlerImpl) Import(ctx context.Context, data *AccountResource, remoteId string) (*AccountState, error) {
	remote, err := h.store.Get(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting account during import: %w", err)
	}

	oldData := &AccountResource{AccountDefinitionName: remote.Definition.Name}
	oldState := &AccountState{ID: remoteId}

	newState, err := h.Update(ctx, data, oldData, oldState)
	if err != nil {
		return nil, fmt.Errorf("updating account during import: %w", err)
	}

	if err := h.store.SetExternalID(ctx, remoteId, data.ID); err != nil {
		return nil, fmt.Errorf("setting external id for account during import: %w", err)
	}

	return newState, nil
}

// FormatForExport converts unmanaged accounts into importable YAML specs: config
// is the remote options with each secret key masked to a per-resource
// "{{ .VAR }}" token. Mirrors the destination export.
func (h *HandlerImpl) FormatForExport(
	collection map[string]*RemoteAccount,
	_ namer.Namer,
	_ resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	if len(collection) == 0 {
		return nil, nil, nil
	}

	var (
		entities []writer.FormattableEntity
		entries  []importmanifest.ImportEntry
	)

	for externalID, remote := range collection {
		specMap, err := h.toExportSpecMap(externalID, remote)
		if err != nil {
			return nil, nil, err
		}

		workspaceMetadata := specs.WorkspaceImportMetadata{
			WorkspaceID: remote.WorkspaceID,
			Resources: []specs.ImportIds{
				{
					URN:      resources.URN(externalID, AccountResourceType),
					RemoteID: remote.ID,
				},
			},
		}
		entries = append(entries, handlers.ImportEntriesFromWorkspace(workspaceMetadata)...)

		spec, err := handlers.ToImportSpec(AccountSpecKind, AccountMetadataName, workspaceMetadata, specMap)
		if err != nil {
			return nil, nil, fmt.Errorf("creating spec for account %s: %w", remote.ID, err)
		}

		entities = append(entities, writer.FormattableEntity{
			Content:      spec,
			RelativePath: filepath.Join("accounts", fmt.Sprintf("%s.yaml", externalID)),
		})
	}

	return entities, entries, nil
}

func (h *HandlerImpl) toExportSpecMap(externalID string, remote *RemoteAccount) (map[string]any, error) {
	keys, ok := accountSecretKeys[remote.Definition.Name]
	if !ok {
		return nil, fmt.Errorf("account %s has unsupported definition %q", remote.ID, remote.Definition.Name)
	}

	config, err := unmarshalOptions(remote.Options)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling options for account %s: %w", remote.ID, err)
	}
	// The API omits secrets, so surface each secret key as present-but-empty so
	// MaskSecrets emits a "{{ .VAR }}" token the user fills via a var file.
	for _, key := range keys {
		if _, exists := config[key]; !exists {
			config[key] = ""
		}
	}
	if err := secret.MaskSecrets(config, externalID, keys); err != nil {
		return nil, fmt.Errorf("masking account %s secrets: %w", remote.ID, err)
	}

	return map[string]any{
		"id":                      externalID,
		"account_definition_name": remote.Definition.Name,
		"config":                  config,
	}, nil
}

// splitConfig reveals the secrets and partitions the flat config into the API's
// options (non-secret) and secret payloads by the definition's secret-key set.
// This is the one account-specific twist over destinations, which keep secrets
// inside a single config blob.
func (h *HandlerImpl) splitConfig(data *AccountResource) (json.RawMessage, json.RawMessage, error) {
	keys, ok := accountSecretKeys[data.AccountDefinitionName]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported account definition %q", data.AccountDefinitionName)
	}

	revealed := secret.RevealSecrets(data.Config, keys)
	secretSet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		secretSet[k] = struct{}{}
	}

	options := map[string]any{}
	secretPayload := map[string]any{}
	for k, v := range revealed {
		if _, isSecret := secretSet[k]; isSecret {
			secretPayload[k] = v
		} else {
			options[k] = v
		}
	}

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling options for account %q: %w", data.ID, err)
	}
	secretJSON, err := json.Marshal(secretPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling secret for account %q: %w", data.ID, err)
	}
	return optionsJSON, secretJSON, nil
}

func unmarshalOptions(raw json.RawMessage) (map[string]any, error) {
	config := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &config); err != nil {
			return nil, err
		}
	}
	return config, nil
}

// supportedRemoteAccounts wraps and filters remote accounts to supported
// definitions. Filtering keys on the account definition (accounts have no
// registry to look a type up in).
func supportedRemoteAccounts(accounts []client.Account) []*RemoteAccount {
	result := make([]*RemoteAccount, 0, len(accounts))
	for i := range accounts {
		a := &accounts[i]
		if _, ok := accountSecretKeys[a.Definition.Name]; !ok {
			continue
		}
		result = append(result, &RemoteAccount{Account: a})
	}
	return result
}
