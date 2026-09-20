package workspace

import (
	"context"
	"errors"
	"testing"

	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/workspace"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindByExternalID(t *testing.T) {
	t.Parallel()

	rows := map[string][]resources.ResourceData{
		"retl-source-sql-model": {
			{"id": "src-1", "externalId": "vip-customers", "name": "High Value Customers"},
			{"id": "src-2", "externalId": "orders", "name": "Orders"},
		},
	}

	t.Run("returns the single matching row", func(t *testing.T) {
		t.Parallel()

		got, err := findByExternalID(context.Background(), &fakeLister{rows: rows}, "retl-source-sql-model", "orders", "rETL source")

		require.NoError(t, err)
		assert.Equal(t, []resources.ResourceData{{"id": "src-2", "externalId": "orders", "name": "Orders"}}, got)
	})

	// The external id is what an author wrote in a spec, so the message names
	// it rather than reporting an empty result.
	t.Run("names the missing id and the kind", func(t *testing.T) {
		t.Parallel()

		_, err := findByExternalID(context.Background(), &fakeLister{rows: rows}, "retl-source-sql-model", "nope", "rETL source")

		assert.EqualError(t, err, `no rETL source with external id "nope" in this workspace`)
	})

	t.Run("a listing failure is surfaced, not reported as missing", func(t *testing.T) {
		t.Parallel()

		_, err := findByExternalID(context.Background(), &fakeLister{err: errors.New("forbidden")}, "retl-source-sql-model", "orders", "rETL source")

		assert.EqualError(t, err, "forbidden")
	})

	// A resource whose listing carries no externalId can never be addressed;
	// this was true of sources and accounts until their List started emitting
	// it, and the failure looked like the resource not existing.
	t.Run("rows without an externalId never match", func(t *testing.T) {
		t.Parallel()

		_, err := findByExternalID(context.Background(),
			&fakeLister{rows: map[string][]resources.ResourceData{"account": {{"id": "acc-1", "name": "warehouse"}}}},
			"account", "warehouse", "account")

		assert.ErrorContains(t, err, `no account with external id "warehouse"`)
	})
}

func TestViewFormat(t *testing.T) {
	t.Parallel()

	// The detailed form, not the table: the table is a bubbles TUI and fails
	// outright without a TTY, which a view read from a pipe must not do.
	assert.Equal(t, lister.DetailedFormat, viewFormat(false))
	assert.Equal(t, lister.JSONFormat, viewFormat(true))
}

func TestOneResourceIgnoresTheQuery(t *testing.T) {
	t.Parallel()

	row := resources.ResourceData{"id": "x"}
	got, err := oneResource{row}.List(context.Background(), "anything", lister.Filters{"a": "b"})

	require.NoError(t, err)
	assert.Equal(t, []resources.ResourceData{row}, got)
}

// An account's resource data is what `accounts view` prints, so the secret
// never being in it is the thing that keeps a password off the screen.
func TestAccountResourceDataCarriesNoSecret(t *testing.T) {
	t.Parallel()

	account := apiClient.Account{
		ID:         "acc-1",
		ExternalID: "analytics-pg",
		Name:       "analytics-pg",
		Options:    []byte(`{"host":"db.internal","port":5432}`),
	}
	data := (&workspace.Account{Account: &account}).ToResourceData()

	assert.Equal(t, "analytics-pg", data["externalId"], "a view addresses accounts by external id")
	for _, secret := range []string{"password", "secret", "token", "secrets", "secretConfig"} {
		assert.NotContains(t, data, secret)
	}
}
