package sqlmodel

import (
	"fmt"
	"regexp"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// AccountKey is the spec field that references a project account as
// "#account:<id>", the alternative to a raw account_id: a spec sets exactly one
// of the two. Both RETL source kinds accept it. In graph data the reference
// sits under AccountIDKey, so every consumer of a dereferenced source reads the
// account's remote id from the same key whichever form the spec used.
const AccountKey = "account"

// accountRefPattern names the validate:"pattern=..." check for AccountKey.
const accountRefPattern = "account_ref"

// accountRefRegex matches "#account:<id>" and captures the id.
var accountRefRegex = regexp.MustCompile(`^#` + accounts.AccountSpecKind + `:(.+)$`)

func init() {
	funcs.NewPattern(accountRefPattern, accountRefRegex.String(), "must be of the form #account:<id>")
}

// ParseAccountRef returns the local id of the account an "#account:<id>"
// reference names.
func ParseAccountRef(ref string) (string, error) {
	matches := accountRefRegex.FindStringSubmatch(ref)
	if matches == nil {
		return "", fmt.Errorf("invalid account reference %q: expected format #%s:<id>", ref, accounts.AccountSpecKind)
	}
	return matches[1], nil
}

// AccountRef returns the graph value of a referenced account: a PropertyRef
// that resolves to the account's remote id once the account is created or
// imported, and that gives the source a dependency edge on it.
func AccountRef(id string) *resources.PropertyRef {
	ref := handler.CreatePropertyRef(
		resources.URN(id, accounts.AccountResourceType),
		func(state *accounts.AccountState) (string, error) {
			if state.ID == "" {
				return "", fmt.Errorf("account state has empty ID")
			}
			return state.ID, nil
		},
	)
	// Stamped so the differ compares it field for field with the state-side
	// ref AccountInput builds, as event stream connections do for destinations.
	ref.Property = "id"
	return ref
}

// AccountInput returns a remote source's account_id state input: a reference to
// the account when the CLI manages it, else the id. keepID is set when the
// source's local spec names its account by id; the input then stays the id, so
// that spec does not show a change on every plan once the CLI manages the
// account. A source with no local spec is about to be deleted, and its
// reference orders that delete before the account's.
//
// A local reference to an account the CLI does not manage compares against the
// id and shows as a change, which apply resolves by pointing the source at the
// referenced account.
func AccountInput(accountID string, keepID bool, collection *resources.RemoteResources) any {
	if keepID {
		return accountID
	}
	urn, err := collection.GetURNByID(accounts.AccountResourceType, accountID)
	if err != nil {
		return accountID
	}
	return &resources.PropertyRef{URN: urn, Property: "id"}
}
