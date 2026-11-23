package workspace

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

const AccountResourceType = "account"

type Account struct {
	*client.Account
}

func (a *Account) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"name": a.Name,
		"id":   a.ID,
		"definition": map[string]string{
			"type":     a.Definition.Type,
			"category": a.Definition.Category,
			// Absent for an account the legacy OAuth flow created: those carry a
			// null accountDefinitionName, and the API reports the account's role
			// as the type instead. Such an account cannot be adopted.
			"name": a.Definition.Name,
		},
		// externalId is how a user tells an account already adopted into a
		// project from a loose one, which is the question the adopt path leaves
		// them with. "managed" is the same fact as a column the table renders.
		"externalId": a.ExternalID,
		"managed":    managedLabel(a.ExternalID),
		"options":    a.Options,
		"createdAt":  a.CreatedAt,
		"updatedAt":  a.UpdatedAt,
	}
}

// managedLabel is the table cell for an account's adoption state.
func managedLabel(externalID string) string {
	if externalID == "" {
		return "-"
	}
	return externalID
}
