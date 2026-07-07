package accounts

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bqItem(id string) AccountItem {
	return AccountItem{
		ID:                    id,
		AccountDefinitionName: "SOURCE_BIGQUERY",
		Config: AccountConfig{
			Credentials: secret.New("{{ .BQ_CREDENTIALS }}"),
			Options:     map[string]any{"project": "lovable-prod"},
		},
	}
}

func TestExtractResourcesFromSpecParsesCollection(t *testing.T) {
	h := &HandlerImpl{}
	spec := &AccountSpec{Accounts: []AccountItem{bqItem("lovable-prod-bq"), bqItem("lovable-staging-bq")}}

	res, err := h.ExtractResourcesFromSpec("accounts.yaml", spec)
	require.NoError(t, err)
	require.Len(t, res, 2)

	a := res["lovable-prod-bq"]
	require.NotNil(t, a)
	assert.Equal(t, "lovable-prod-bq", a.ID)
	assert.Equal(t, "SOURCE_BIGQUERY", a.AccountDefinitionName)
	assert.Equal(t, "lovable-prod", a.Options["project"])
	require.NotNil(t, a.Credentials, "credentials must be carried as *secret.String")
}

func TestExtractResourcesFromSpecRejectsDuplicateID(t *testing.T) {
	h := &HandlerImpl{}
	spec := &AccountSpec{Accounts: []AccountItem{bqItem("dup"), bqItem("dup")}}

	_, err := h.ExtractResourcesFromSpec("accounts.yaml", spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate account id")
}

func TestExtractResourcesFromSpecValidation(t *testing.T) {
	cases := []struct {
		name string
		item AccountItem
		want string
	}{
		{"missing id", AccountItem{AccountDefinitionName: "SOURCE_BIGQUERY"}, "id is required"},
		{"invalid id token", AccountItem{ID: "bad id!", AccountDefinitionName: "SOURCE_BIGQUERY"}, "not a valid external-id token"},
		{"missing definition", AccountItem{ID: "a"}, "accountDefinitionName is required"},
		{"not screaming snake", AccountItem{ID: "a", AccountDefinitionName: "source_bigquery"}, "SCREAMING_SNAKE_CASE"},
		{"unsupported definition", AccountItem{ID: "a", AccountDefinitionName: "SOURCE_SNOWFLAKE"}, "not supported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &HandlerImpl{}
			_, err := h.ExtractResourcesFromSpec("accounts.yaml", &AccountSpec{Accounts: []AccountItem{tc.item}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestHandlerMetadata(t *testing.T) {
	h := &HandlerImpl{}
	assert.Equal(t, "accounts", h.Metadata().SpecKind)
	assert.Equal(t, "account", h.Metadata().ResourceType)
}
