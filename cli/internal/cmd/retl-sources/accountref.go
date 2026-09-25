package retlsource

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// accountLister is the slice of the accounts API this package needs, so the
// resolver is testable without a live client.
type accountLister interface {
	ListAll(ctx context.Context, opts ...client.ListAccountsOption) ([]client.Account, error)
}

// resolveAccountRef swaps an unresolved `#account:` ref for the remote id of the
// managed account whose externalId matches, the same rule MapRemoteToState uses,
// because preview and validate read the graph without remote state.
func resolveAccountRef(ctx context.Context, lister accountLister, data resources.ResourceData) (resources.ResourceData, error) {
	ref, ok := data[sqlmodel.AccountIDKey].(*resources.PropertyRef)
	if !ok {
		return data, nil
	}
	localID, ok := strings.CutPrefix(ref.URN, accounts.AccountResourceType+":")
	if !ok {
		return data, nil
	}

	all, err := lister.ListAll(ctx, client.WithHasExternalID(true))
	if err != nil {
		return nil, fmt.Errorf("listing managed accounts: %w", err)
	}

	// config-backend enforces UNIQUE (workspaceId, externalId) on managed
	// accounts (UK__accounts__workspace_id__external_id), so the first match is
	// the only one.
	remoteID := ""
	for _, account := range all {
		if account.ExternalID == localID {
			remoteID = account.ID
			break
		}
	}
	if remoteID == "" {
		return nil, fmt.Errorf("account %q is referenced but does not exist in the workspace yet; run `rudder-cli apply` first, or set account_id to preview against an existing account", localID)
	}

	// Copied rather than mutated: the caller's data is the graph's, and the
	// graph is shared with anything else reading the project.
	resolved := maps.Clone(data)
	resolved[sqlmodel.AccountIDKey] = remoteID
	return resolved, nil
}
