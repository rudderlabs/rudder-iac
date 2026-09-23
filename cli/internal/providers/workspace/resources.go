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
		// externalId is how an author addresses this account in a spec, and it
		// marks which rows the CLI manages. Without it a listing cannot be
		// looked up by the id the spec used.
		"externalId": a.ExternalID,
		"definition": map[string]string{
			"type":     a.Definition.Type,
			"category": a.Definition.Category,
		},
		"options":   a.Options,
		"createdAt": a.CreatedAt,
		"updatedAt": a.UpdatedAt,
	}
}
