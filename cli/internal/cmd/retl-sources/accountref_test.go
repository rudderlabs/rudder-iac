package retlsource

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type stubAccounts struct {
	accounts []client.Account
	err      error
	opts     []client.ListAccountsOption
}

func (s *stubAccounts) ListAll(_ context.Context, opts ...client.ListAccountsOption) ([]client.Account, error) {
	s.opts = opts
	return s.accounts, s.err
}

func TestResolveAccountRef(t *testing.T) {
	t.Parallel()

	t.Run("resolves a reference to the managed account's remote id", func(t *testing.T) {
		t.Parallel()

		lister := &stubAccounts{accounts: []client.Account{
			{ID: "acc-other", ExternalID: "staging-pg"},
			{ID: "acc-1", ExternalID: "prod-pg"},
		}}
		data := resources.ResourceData{sqlmodel.AccountIDKey: sqlmodel.AccountRef("prod-pg"), sqlmodel.SQLKey: "SELECT 1"}

		resolved, err := resolveAccountRef(context.Background(), lister, data)

		require.NoError(t, err)
		assert.Equal(t, resources.ResourceData{
			sqlmodel.AccountIDKey: "acc-1",
			sqlmodel.SQLKey:       "SELECT 1",
		}, resolved)
		assert.Len(t, lister.opts, 1, "only managed accounts carry an external id to match on")
	})

	// The graph is shared with everything else reading the project, so the
	// resolved copy must not write back into it.
	t.Run("leaves the caller's data untouched", func(t *testing.T) {
		t.Parallel()

		ref := sqlmodel.AccountRef("prod-pg")
		data := resources.ResourceData{sqlmodel.AccountIDKey: ref}
		lister := &stubAccounts{accounts: []client.Account{{ID: "acc-1", ExternalID: "prod-pg"}}}

		_, err := resolveAccountRef(context.Background(), lister, data)

		require.NoError(t, err)
		assert.Equal(t, ref, data[sqlmodel.AccountIDKey])
	})

	t.Run("passes a plain account id through without calling the API", func(t *testing.T) {
		t.Parallel()

		lister := &stubAccounts{err: errors.New("must not be called")}
		data := resources.ResourceData{sqlmodel.AccountIDKey: "acc-1"}

		resolved, err := resolveAccountRef(context.Background(), lister, data)

		require.NoError(t, err)
		assert.Equal(t, data, resolved)
		assert.Nil(t, lister.opts)
	})

	// The account is in the project but not yet in the workspace: apply is what
	// fixes it, and the message has to say so rather than failing downstream
	// with an empty account id.
	t.Run("names the account when it does not exist upstream", func(t *testing.T) {
		t.Parallel()

		lister := &stubAccounts{accounts: []client.Account{{ID: "acc-1", ExternalID: "staging-pg"}}}
		data := resources.ResourceData{sqlmodel.AccountIDKey: sqlmodel.AccountRef("prod-pg")}

		_, err := resolveAccountRef(context.Background(), lister, data)

		require.EqualError(t, err, "account \"prod-pg\" is referenced but does not exist in the workspace yet; run `rudder-cli apply` first, or set account_id to preview against an existing account")
	})

	t.Run("wraps a listing failure", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("boom")
		lister := &stubAccounts{err: sentinel}
		data := resources.ResourceData{sqlmodel.AccountIDKey: sqlmodel.AccountRef("prod-pg")}

		_, err := resolveAccountRef(context.Background(), lister, data)

		require.ErrorIs(t, err, sentinel)
		require.EqualError(t, err, "listing managed accounts: boom")
	})
}
