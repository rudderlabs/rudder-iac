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
		},
		"options":   a.Options,
		"createdAt": a.CreatedAt,
		"updatedAt": a.UpdatedAt,
	}
}
