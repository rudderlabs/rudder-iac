package workspace

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
)

func account(name, externalID, definitionName string) *Account {
	a := &client.Account{
		ID:         "acct-1",
		Name:       name,
		ExternalID: externalID,
	}
	a.Definition.Name = definitionName
	a.Definition.Type = "google_adwords_offline_conversions"
	a.Definition.Category = "destination"
	return &Account{Account: a}
}

// An adopted account has to be distinguishable from a loose one, which is the
// question `workspace accounts list` leaves a user with once adopt exists.
func TestAccountListShowsAdoptionState(t *testing.T) {
	t.Parallel()

	adopted := account("Google Ads (prod)", "google-ads-prod",
		"DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH").ToResourceData()
	assert.Equal(t, "google-ads-prod", adopted["externalId"])
	assert.Equal(t, "google-ads-prod", adopted["managed"])

	loose := account("Google Ads (staging)", "",
		"DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH").ToResourceData()
	assert.Equal(t, "", loose["externalId"])
	assert.Equal(t, "-", loose["managed"])
}

// An account the legacy OAuth flow created carries a null accountDefinitionName,
// so the API reports no definition name for it. Surfacing that is how a user
// sees why such an account cannot be adopted.
func TestAccountListSurfacesTheDefinitionName(t *testing.T) {
	t.Parallel()

	withDefinition := account("Google Ads", "", "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH")
	assert.Equal(t, map[string]string{
		"type":     "google_adwords_offline_conversions",
		"category": "destination",
		"name":     "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH",
	}, withDefinition.ToResourceData()["definition"])

	legacy := account("LinkedIn (prod)", "", "")
	assert.Equal(t, map[string]string{
		"type":     "google_adwords_offline_conversions",
		"category": "destination",
		"name":     "",
	}, legacy.ToResourceData()["definition"])
}

// Every lister shares one table printer, so the column must be driven by the
// data and not by the resource type.
func TestManagedLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "-", managedLabel(""))
	assert.Equal(t, "google-ads-prod", managedLabel("google-ads-prod"))
}
