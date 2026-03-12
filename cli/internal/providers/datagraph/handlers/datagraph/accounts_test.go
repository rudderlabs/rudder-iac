package datagraph

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAccountGetter struct {
	account *client.Account
	err     error
}

func (m *mockAccountGetter) Get(_ context.Context, _ string) (*client.Account, error) {
	return m.account, m.err
}

func TestAccountNameResolver_GetAccountName(t *testing.T) {
	t.Run("returns account name when present", func(t *testing.T) {
		resolver := NewAccountNameResolver(&mockAccountGetter{
			account: &client.Account{
				Name:       "My Warehouse",
				Definition: struct {
					Type     string `json:"type"`
					Category string `json:"category"`
				}{Type: "SNOWFLAKE"},
			},
		})

		name, err := resolver.GetAccountName(context.Background(), "account-1")
		require.NoError(t, err)
		assert.Equal(t, "My Warehouse", name)
	})

	t.Run("falls back to definition type when name is empty", func(t *testing.T) {
		resolver := NewAccountNameResolver(&mockAccountGetter{
			account: &client.Account{
				Definition: struct {
					Type     string `json:"type"`
					Category string `json:"category"`
				}{Type: "SNOWFLAKE"},
			},
		})

		name, err := resolver.GetAccountName(context.Background(), "account-1")
		require.NoError(t, err)
		assert.Equal(t, "SNOWFLAKE", name)
	})

	t.Run("returns error when both name and definition type are empty", func(t *testing.T) {
		resolver := NewAccountNameResolver(&mockAccountGetter{
			account: &client.Account{},
		})

		_, err := resolver.GetAccountName(context.Background(), "account-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no name or definition type")
	})

	t.Run("returns error when API call fails", func(t *testing.T) {
		resolver := NewAccountNameResolver(&mockAccountGetter{
			err: assert.AnError,
		})

		_, err := resolver.GetAccountName(context.Background(), "account-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetching account")
	})

	t.Run("caches results across calls", func(t *testing.T) {
		callCount := 0
		getter := &countingAccountGetter{
			account:   &client.Account{Name: "Cached Warehouse"},
			callCount: &callCount,
		}
		resolver := NewAccountNameResolver(getter)

		name1, err := resolver.GetAccountName(context.Background(), "account-1")
		require.NoError(t, err)
		assert.Equal(t, "Cached Warehouse", name1)

		name2, err := resolver.GetAccountName(context.Background(), "account-1")
		require.NoError(t, err)
		assert.Equal(t, "Cached Warehouse", name2)

		// Different account ID should trigger a new call
		name3, err := resolver.GetAccountName(context.Background(), "account-2")
		require.NoError(t, err)
		assert.Equal(t, "Cached Warehouse", name3)

		assert.Equal(t, 2, callCount, "API should be called once per unique account ID")
	})
}

type countingAccountGetter struct {
	account   *client.Account
	callCount *int
}

func (m *countingAccountGetter) Get(_ context.Context, _ string) (*client.Account, error) {
	*m.callCount++
	return m.account, nil
}
