package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/examples"
)

// Each WithCategory conflict says why the category cannot be sent. None
// reaches the client.
func TestWithCategoryErrors(t *testing.T) {
	var (
		ex = examples.New(&fakeClient{})
		id = examples.Identity{UserID: "user-123"}
	)

	tests := []struct {
		name    string
		call    func() error
		wantErr string
	}{
		{
			name: "a declared category left unset",
			call: func() error {
				return ex.PageCategoryVariant(id, "", examples.PageCategoryVariantPropertiesAbsent{}, examples.WithCategory("news"))
			},
			wantErr: `ruddertyper: category conflicts with the tracking plan: WithCategory("news") on a payload whose declared category is unset; set it in the payload`,
		},
		{
			name: "a closed schema without category",
			call: func() error {
				return ex.Screen(id, "", &examples.ScreenProperties{ProductID: "p1"}, examples.WithCategory("Products"))
			},
			wantErr: `ruddertyper: category conflicts with the tracking plan: WithCategory("Products") on a payload that does not accept an undeclared category`,
		},
		{
			name: "a different typed category",
			call: func() error {
				return ex.PageCategory(id, "", &examples.PageCategoryProperties{Category: examples.Ptr("news")}, examples.WithCategory("sports"))
			},
			wantErr: `ruddertyper: category conflicts with the tracking plan: WithCategory("sports") differs from properties.category "news"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.call(), tt.wantErr)
		})
	}
}
