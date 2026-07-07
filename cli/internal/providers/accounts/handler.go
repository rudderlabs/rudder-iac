package accounts

import (
	"context"
	"fmt"
	"regexp"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

// bigQueryDefinition is the only account definition supported this milestone
// (BigQuery-only scope). Other warehouses land in the "Post Bigquery" work.
const bigQueryDefinition = "SOURCE_BIGQUERY"

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
	List(ctx context.Context, opts ...client.ListAccountsOption) (*client.AccountsPage, error)
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
type HandlerImpl struct {
	store AccountStore
}

// NewHandler builds the account BaseHandler around the given store.
func NewHandler(store AccountStore) *AccountHandler {
	return handler.NewHandler(&HandlerImpl{store: store})
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

// --- Remote lifecycle: stubbed here; implemented in DEX-459 (CRUD + load + map)
//     and DEX-460 (import + export). ---

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*RemoteAccount, error) {
	return nil, fmt.Errorf("accounts: LoadRemoteResources not implemented (DEX-459)")
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*RemoteAccount, error) {
	return nil, fmt.Errorf("accounts: LoadImportableResources not implemented (DEX-459)")
}

func (h *HandlerImpl) MapRemoteToState(remote *RemoteAccount, urnResolver handler.URNResolver) (*AccountResource, *AccountState, error) {
	return nil, nil, fmt.Errorf("accounts: MapRemoteToState not implemented (DEX-459)")
}

func (h *HandlerImpl) Create(ctx context.Context, data *AccountResource) (*AccountState, error) {
	return nil, fmt.Errorf("accounts: Create not implemented (DEX-459)")
}

func (h *HandlerImpl) Update(ctx context.Context, newData *AccountResource, oldData *AccountResource, oldState *AccountState) (*AccountState, error) {
	return nil, fmt.Errorf("accounts: Update not implemented (DEX-459)")
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *AccountResource, oldState *AccountState) error {
	return fmt.Errorf("accounts: Delete not implemented (DEX-459)")
}

func (h *HandlerImpl) Import(ctx context.Context, data *AccountResource, remoteId string) (*AccountState, error) {
	return nil, fmt.Errorf("accounts: Import not implemented (DEX-460)")
}

func (h *HandlerImpl) FormatForExport(
	collection map[string]*RemoteAccount,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	return nil, nil, fmt.Errorf("accounts: FormatForExport not implemented (DEX-460)")
}
