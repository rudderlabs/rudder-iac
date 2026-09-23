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

// resolveAccountRef replaces an account reference in a source's graph data with
// the account's remote id, returning data the preview API can be called with.
//
// preview and validate read the project graph without remote state, so a source
// written as `account: "#account:<id>"` still carries the PropertyRef the graph
// uses to order an apply. The remote id that ref would resolve to is the id of
// the managed account whose externalId is the referenced local id — which is
// how the state itself is built (accounts.MapRemoteToState reads the local id
// straight back out of ExternalID), so this is the same answer by the same
// rule rather than a second source of truth.
//
// Data that already carries a plain account_id is returned untouched, and so is
// data for any other kind.
func resolveAccountRef(ctx context.Context, lister accountLister, data resources.ResourceData) (resources.ResourceData, error) {
	// Both shapes are read: retl and event-stream store *resources.PropertyRef
	// while datacatalog stores it by value.
	var urn string
	switch ref := data[sqlmodel.AccountIDKey].(type) {
	case *resources.PropertyRef:
		urn = ref.URN
	case resources.PropertyRef:
		urn = ref.URN
	default:
		return data, nil
	}
	localID, ok := strings.CutPrefix(urn, accounts.AccountResourceType+":")
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
