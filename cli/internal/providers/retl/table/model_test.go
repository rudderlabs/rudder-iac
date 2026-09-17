package table_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sourcekeys"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
)

// The oneof tag is written by hand — struct tags cannot interpolate a constant
// — so it can drift from the definition set it mirrors. Table sources accept
// the warehouses plus s3, and this is what keeps the tag saying so.
func TestSourceDefinitionTagMatchesTableSet(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeOf(table.TableSpec{}).FieldByName("SourceDefinition")
	require.True(t, ok)

	want := "required,oneof=" + sourcekeys.OneOf(sourcekeys.TableDefinitions)
	assert.Equal(t, want, field.Tag.Get("validate"),
		"the oneof tag must list exactly sourcekeys.TableDefinitions")
}
