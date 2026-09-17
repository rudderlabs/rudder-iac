package sqlmodel_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sourcekeys"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
)

// The oneof tag on SQLModelSpec.SourceDefinition is written by hand — struct
// tags cannot interpolate a constant — so it can drift from the definition set
// it is supposed to mirror. This is the check that catches that, and the reason
// sourcekeys.OneOf exists.
func TestSourceDefinitionTagMatchesWarehouseSet(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeOf(sqlmodel.SQLModelSpec{}).FieldByName("SourceDefinition")
	require.True(t, ok)

	want := "required,oneof=" + sourcekeys.OneOf(sourcekeys.WarehouseDefinitions)
	assert.Equal(t, want, field.Tag.Get("validate"),
		"the oneof tag must list exactly sourcekeys.WarehouseDefinitions")
}
