package datacatalog

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// PropertySpecFactory creates instances of PropertySpec for validation
type PropertySpecFactory struct{}

func (f *PropertySpecFactory) Kind() string {
	return "properties"
}

func (f *PropertySpecFactory) NewSpec() any {
	return &localcatalog.PropertySpec{}
}

func (f *PropertySpecFactory) SpecFieldName() string {
	return "properties"
}

func (f *PropertySpecFactory) Examples() rules.Examples {
	return rules.Examples{
		Valid: []string{
			`properties:
  - id: user_id
    name: User ID
    description: Unique identifier for the user
    type: string
  - id: email
    name: Email
    type: string`,
		},
		Invalid: []string{
			`properties:
  - name: Missing ID
    type: string`,
			`properties:
  - id: user_id
    # Missing required name field
    type: string`,
		},
	}
}

// Compile-time verification that PropertySpecFactory
// implements SpecFactory
var _ provider.SpecFactory = (*PropertySpecFactory)(nil)
