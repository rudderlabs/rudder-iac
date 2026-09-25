package retlsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

func graphOf(rs ...*resources.Resource) *resources.Graph {
	g := resources.NewGraph()
	for _, r := range rs {
		g.AddResource(r)
	}
	return g
}

func TestFindSource(t *testing.T) {
	t.Parallel()

	var (
		model = resources.NewResource("orders", sqlmodel.ResourceType, resources.ResourceData{}, nil)
		tbl   = resources.NewResource("users", table.ResourceType, resources.ResourceData{}, nil)
	)

	t.Run("finds a SQL model", func(t *testing.T) {
		t.Parallel()
		got, err := findSource(graphOf(model, tbl), "orders")
		require.NoError(t, err)
		assert.Same(t, model, got)
	})

	t.Run("finds a table source", func(t *testing.T) {
		t.Parallel()
		got, err := findSource(graphOf(model, tbl), "users")
		require.NoError(t, err)
		assert.Same(t, tbl, got)
	})

	t.Run("reports a missing id as before", func(t *testing.T) {
		t.Parallel()
		_, err := findSource(graphOf(model), "users")
		require.EqualError(t, err, "resource with external id 'users' not found in project")
	})

	t.Run("rejects an id shared by both kinds", func(t *testing.T) {
		t.Parallel()
		shared := resources.NewResource("orders", table.ResourceType, resources.ResourceData{}, nil)
		_, err := findSource(graphOf(model, shared), "orders")
		require.EqualError(t, err, "external id 'orders' is used by both retl-source-sql-model and retl-source-table in the project")
	})
}
