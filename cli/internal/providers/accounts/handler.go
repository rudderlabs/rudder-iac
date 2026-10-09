package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
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
	// RETL sources reference accounts as "#account:<id>".
	ReferencedByKind: true,
}

// accountDefinition is what the CLI knows about one account definition.
//
// Type is the account's role in the control plane, which for a source account
// is the name of the source definition it backs ("type" in
// integrations-config's sources/<type>/accounts/<name>/db-config.json).
//
// SecretKeys is its secret field set — the account-side analogue of a
// destination definition's SecretKeys(). All other config keys are treated as
// (non-secret) options by splitConfig. For Snowflake, only the auth mode in
// play supplies one of the secrets; the others are simply absent from the
// user's config and the split handles that generically.
type accountDefinition struct {
	Type       string
	SecretKeys []string
	// RequiredOptions are the config keys the account schema marks required, so
	// validate can flag a missing one before apply reaches the API (DEX-994).
	// Required secrets are derived from SecretKeys, see requiredSecrets.
	RequiredOptions []string
}

// registeredAccounts is every account definition the CLI can manage. One entry
// per definition, so a definition cannot have a type without a secret set or
// the reverse.
//
// ponytail: hardcoded. The real registry fetches secretFields from the control-plane
// account-definitions API (unversioned, name-keyed) — see DEX-467. The split
// logic below is driven by these definitions, so a new warehouse is a new entry
// here plus, when it has several auth modes, an entry in authModeRequirements.
var registeredAccounts = map[string]accountDefinition{
	"SOURCE_BIGQUERY": {
		Type: "bigquery", SecretKeys: []string{"credentials"},
		RequiredOptions: []string{"project"},
	},
	"SOURCE_POSTGRES": {
		Type: "postgres", SecretKeys: []string{"password"},
		RequiredOptions: []string{"host", "dbname", "user", "port", "sslMode"},
	},
	"SOURCE_SNOWFLAKE": {
		Type: "snowflake", SecretKeys: []string{"password", "privateKey", "privateKeyPassphrase"},
		RequiredOptions: []string{"account", "dbname", "warehouse", "user", "authenticationType"},
	},
}

// optionalSecrets are secret keys an account schema does not require.
var optionalSecrets = []string{"privateKeyPassphrase"}

// requiredSecrets is keys without the optional ones.
func requiredSecrets(keys []string) []string {
	return slices.DeleteFunc(slices.Clone(keys), func(key string) bool {
		return slices.Contains(optionalSecrets, key)
	})
}

// authModeRequirement is how a definition with several auth modes picks the
// config keys it requires on top of RequiredOptions.
type authModeRequirement struct {
	// Key is the config key that selects the mode.
	Key string
	// Required lists the keys each mode needs.
	Required map[string][]string
}

// authModeRequirements covers the definitions whose required keys depend on the
// auth mode. A definition absent here requires all of its non-optional secrets.
//
// A new definition with several auth modes also needs an entry in
// authModeSecrets and absentAuthModes, or every secret is sent again on each
// apply (DEX-958).
var authModeRequirements = map[string]authModeRequirement{
	"SOURCE_BIGQUERY": {
		Key: "authMethod",
		Required: map[string][]string{
			"serviceAccountKey": requiredSecrets(registeredAccounts["SOURCE_BIGQUERY"].SecretKeys),
			// Federation has no key file; the schema requires these three
			// options instead and wants credentials empty.
			"workloadIdentityFederation": {"workloadIdentityProjectNumber", "workloadIdentityPoolId", "workloadIdentityProviderId"},
		},
	},
	"SOURCE_SNOWFLAKE": {
		Key: "authenticationType",
		Required: map[string][]string{
			"keyPair":  requiredSecrets(authModeSecrets["SOURCE_SNOWFLAKE"]["keyPair"]),
			"password": requiredSecrets(authModeSecrets["SOURCE_SNOWFLAKE"]["password"]),
		},
	},
}

// given reports whether config sets key to something. A null value, as in
// "account:" with nothing after it, counts as not set.
func given(config map[string]any, key string) bool {
	return config[key] != nil
}

// requiredConfigKeys lists the keys an account of this definition must set,
// options first.
func requiredConfigKeys(definitionName string, def accountDefinition, config map[string]any) []string {
	required := slices.Clone(def.RequiredOptions)
	modes, discriminated := authModeRequirements[definitionName]
	if !discriminated {
		return append(required, requiredSecrets(def.SecretKeys)...)
	}
	// A missing required discriminator and an unknown mode are each reported on
	// their own, since there is no telling which keys are required.
	if _, _, unknown := unknownAuthMode(definitionName, config); unknown {
		return required
	}
	if slices.Contains(def.RequiredOptions, modes.Key) && !given(config, modes.Key) {
		return required
	}
	return append(required, modes.Required[authMode(definitionName, config)]...)
}

// missingRequiredConfig lists the required config keys an account of this
// definition leaves out, options first. A definition the CLI does not register
// has no known requirements, so it reports nothing.
func missingRequiredConfig(definitionName string, config map[string]any) []string {
	def, ok := registeredAccounts[definitionName]
	if !ok {
		return nil
	}

	var missing []string
	for _, key := range requiredConfigKeys(definitionName, def, config) {
		if !given(config, key) {
			missing = append(missing, key)
		}
	}
	return missing
}

// unknownAuthMode reports the config key and the allowed values when the
// account names an auth mode its definition does not have. Without this check
// an unknown mode requires nothing, so a typo such as "serviceAccount" passes
// validate and fails at the API (DEX-994). An unset mode is not unknown.
func unknownAuthMode(definitionName string, config map[string]any) (key string, allowed []string, unknown bool) {
	modes, discriminated := authModeRequirements[definitionName]
	if !discriminated || !given(config, modes.Key) {
		return "", nil, false
	}
	if mode, isString := config[modes.Key].(string); isString {
		if _, known := modes.Required[mode]; known || mode == "" {
			return "", nil, false
		}
	}
	return modes.Key, slices.Sorted(maps.Keys(modes.Required)), true
}

// authModeSecrets maps each auth mode of a discriminated definition to the
// secrets it uses. A definition absent here has a single mode.
//
// ponytail: hardcoded alongside registeredAccounts and goes away with the same
// DEX-467 move to the control-plane account-definitions API, whose db-config
// already carries the discriminator ("authenticationType": "key_pair_or_password").
var authModeSecrets = map[string]map[string][]string{
	"SOURCE_SNOWFLAKE": {
		"keyPair":  {"privateKey", "privateKeyPassphrase"},
		"password": {"password"},
	},
}

// absentAuthModes maps a discriminated definition to the mode an account
// without the discriminator runs in. The Snowflake connector enables key-pair auth only on an explicit "keyPair"
// (rudder-sources snowflake.NewClient); an absent value predates key-pair
// support, so it is a password account whatever the form's default says.
var absentAuthModes = map[string]string{
	"SOURCE_BIGQUERY":  "serviceAccountKey",
	"SOURCE_SNOWFLAKE": "password",
}

// authMode is the account's auth mode, or "" for a definition with a single mode.
func authMode(definitionName string, config map[string]any) string {
	if mode, _ := config[authModeRequirements[definitionName].Key].(string); mode != "" {
		return mode
	}
	return absentAuthModes[definitionName]
}

// authModeSecretKeys is the subset of a definition's secret keys the account's
// own config can actually use. The account schema puts each mode's secrets
// behind an authenticationType branch with additionalProperties false, so the
// other mode's secret is not merely unused — it is rejected (DEX-958).
//
// A mode outside the enum keeps the full set: under-exporting would drop a
// secret the account needs, and a value the schema does not know is a shape this
// code should not be guessing at.
func authModeSecretKeys(definitionName string, config map[string]any, keys []string) []string {
	if modeKeys, ok := authModeSecrets[definitionName][authMode(definitionName, config)]; ok {
		return modeKeys
	}
	return keys
}

// DefinitionType returns the type of a registered account definition, e.g.
// "postgres" for SOURCE_POSTGRES. ok is false for an unregistered definition.
func DefinitionType(accountDefinitionName string) (string, bool) {
	d, ok := registeredAccounts[accountDefinitionName]
	return d.Type, ok
}

// secretKeys returns the secret field set of a registered account definition.
// ok is false for an unregistered definition.
func secretKeys(accountDefinitionName string) ([]string, bool) {
	d, ok := registeredAccounts[accountDefinitionName]
	return d.SecretKeys, ok
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
	keys, ok := secretKeys(spec.AccountDefinitionName)
	if !ok {
		return nil, fmt.Errorf("unsupported account definition %q", spec.AccountDefinitionName)
	}
	resource := &AccountResource{
		ID:                    spec.ID,
		Name:                  spec.Name,
		AccountDefinitionName: spec.AccountDefinitionName,
		Config:                secret.WrapKnownSecrets(spec.Config, keys),
	}
	return map[string]*AccountResource{spec.ID: resource}, nil
}

// Create provisions the account and claims the spec ID as its externalId in the
// same call — the backend sets externalId atomically with creation, so there is
// no separate SetExternalID round trip that could leave a partially-adopted
// resource behind. Import still adopts an existing account via Update +
// SetExternalID (it cannot create).
func (h *HandlerImpl) Create(ctx context.Context, data *AccountResource) (*AccountState, error) {
	options, secretPayload, err := h.splitConfig(data)
	if err != nil {
		return nil, err
	}

	created, err := h.store.Create(ctx, &client.CreateAccountRequest{
		AccountDefinitionName: data.AccountDefinitionName,
		Name:                  data.Name,
		Options:               options,
		Secret:                secretPayload,
		ExternalID:            data.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating account %q: %w", data.ID, err)
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
		Name:    newData.Name,
		Options: options,
		Secret:  secretPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("updating account %q: %w", newData.ID, err)
	}

	return &AccountState{ID: updated.ID}, nil
}

// Delete annotates an in-use refusal with what to do about it (DEX-959). Every
// other failure passes through untouched.
func (h *HandlerImpl) Delete(ctx context.Context, _ string, _ *AccountResource, oldState *AccountState) error {
	if err := h.store.Delete(ctx, oldState.ID); err != nil {
		return fmt.Errorf("deleting account %q: %w", oldState.ID, provider.ExplainBlockingAccountUsage(err))
	}
	return nil
}

// MapRemoteToState rebuilds the flat config from the remote options and marks
// the auth mode's secret keys unknown (the API never returns secret values), so
// the differ flags them SecretOnly rather than phantom drift — same rule as
// destinations.
func (h *HandlerImpl) MapRemoteToState(remote *RemoteAccount, _ handler.URNResolver) (*AccountResource, *AccountState, error) {
	if remote.ExternalID == "" {
		return nil, nil, fmt.Errorf("managed account %s has empty external ID", remote.ID)
	}

	keys, ok := secretKeys(remote.Definition.Name)
	if !ok {
		return nil, nil, fmt.Errorf("managed account %s has unsupported definition %q", remote.ID, remote.Definition.Name)
	}

	config, err := unmarshalOptions(remote.Options)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshalling options for account %s: %w", remote.ID, err)
	}
	// The API never returns the secret, so it is absent from remote options. Seed
	// the auth mode's secret keys so the presence-based WrapUnknownSecrets marks
	// them unknown — within its mode an account secret is unconditional (unlike a
	// destination's optional secrets), so it must always be present-and-unknown
	// and therefore always re-applied. Wrapping still covers every key, so a
	// secret of another mode that the API echoed back is never held as plain text.
	for _, key := range authModeSecretKeys(remote.Definition.Name, config, keys) {
		if _, ok := config[key]; !ok {
			config[key] = ""
		}
	}
	config = secret.WrapUnknownSecrets(config, keys)

	resource := &AccountResource{
		ID:                    remote.ExternalID,
		Name:                  remote.Name,
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

		spec, err := specs.ToImportSpec(AccountSpecKind, AccountMetadataName, workspaceMetadata, specMap)
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
	keys, ok := secretKeys(remote.Definition.Name)
	if !ok {
		return nil, fmt.Errorf("account %s has unsupported definition %q", remote.ID, remote.Definition.Name)
	}

	config, err := unmarshalOptions(remote.Options)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling options for account %s: %w", remote.ID, err)
	}
	// The account schema requires the discriminator on every update, so an
	// account that predates it would be rejected on its first apply. Writing the
	// mode it already runs in makes that apply add it instead.
	if mode := authMode(remote.Definition.Name, config); mode != "" {
		config[authModeRequirements[remote.Definition.Name].Key] = mode
	}
	// The API omits secrets, so surface the auth mode's secret keys as
	// present-but-empty so MaskSecrets emits a "{{ .VAR }}" token the user fills
	// via a var file. Masking covers every key, so a secret of another mode that
	// the API echoed back is never written in plain text.
	for _, key := range authModeSecretKeys(remote.Definition.Name, config, keys) {
		if _, exists := config[key]; !exists {
			config[key] = ""
		}
	}
	if err := secret.MaskSecrets(config, externalID, keys); err != nil {
		return nil, fmt.Errorf("masking account %s secrets: %w", remote.ID, err)
	}

	return map[string]any{
		"id":                      externalID,
		"name":                    remote.Name,
		"account_definition_name": remote.Definition.Name,
		"config":                  config,
	}, nil
}

// splitConfig reveals the secrets and partitions the flat config into the API's
// options (non-secret) and secret payloads by the definition's secret-key set.
// This is the one account-specific twist over destinations, which keep secrets
// inside a single config blob.
func (h *HandlerImpl) splitConfig(data *AccountResource) (json.RawMessage, json.RawMessage, error) {
	keys, ok := secretKeys(data.AccountDefinitionName)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported account definition %q", data.AccountDefinitionName)
	}

	// The partition below matches top-level keys exactly, so a dotted key would
	// leave its whole container — carrying the revealed plaintext — in options,
	// the non-secret API field, while secretPayload stayed empty. The shared
	// secret helpers do understand dotted paths (DEX-531); this split does not
	// yet, and the registry turns control-plane-driven with DEX-467. Refuse
	// before revealing anything.
	for _, k := range keys {
		if strings.Contains(k, ".") {
			return nil, nil, fmt.Errorf(
				"account definition %q declares nested secret key %q, which the account config split does not support",
				data.AccountDefinitionName, k,
			)
		}
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
		if _, ok := registeredAccounts[a.Definition.Name]; !ok {
			continue
		}
		result = append(result, &RemoteAccount{Account: a})
	}
	return result
}
