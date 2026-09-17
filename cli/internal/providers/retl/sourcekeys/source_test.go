package sourcekeys_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sourcekeys"
)

// The keys are the wire contract of the resource graph: a remote source
// mapped to state and a local spec must land on byte-identical keys or the
// differ reports a phantom change on every plan. Pinned as a whole map so a
// rename has to be deliberate.
func TestKeysAreStable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, map[string]string{
		"id":                sourcekeys.IDKey,
		"local_id":          sourcekeys.LocalIDKey,
		"display_name":      sourcekeys.DisplayNameKey,
		"account_id":        sourcekeys.AccountIDKey,
		"source_definition": sourcekeys.SourceDefinitionKey,
		"primary_key":       sourcekeys.PrimaryKeyKey,
		"enabled":           sourcekeys.EnabledKey,
		"source_type":       sourcekeys.SourceTypeKey,
		"createdAt":         sourcekeys.CreatedAtKey,
		"updatedAt":         sourcekeys.UpdatedAtKey,
	}, map[string]string{
		sourcekeys.IDKey:               sourcekeys.IDKey,
		sourcekeys.LocalIDKey:          sourcekeys.LocalIDKey,
		sourcekeys.DisplayNameKey:      sourcekeys.DisplayNameKey,
		sourcekeys.AccountIDKey:        sourcekeys.AccountIDKey,
		sourcekeys.SourceDefinitionKey: sourcekeys.SourceDefinitionKey,
		sourcekeys.PrimaryKeyKey:       sourcekeys.PrimaryKeyKey,
		sourcekeys.EnabledKey:          sourcekeys.EnabledKey,
		sourcekeys.SourceTypeKey:       sourcekeys.SourceTypeKey,
		sourcekeys.CreatedAtKey:        sourcekeys.CreatedAtKey,
		sourcekeys.UpdatedAtKey:        sourcekeys.UpdatedAtKey,
	})
}

// s3 is file-backed, so it is a table definition but never a warehouse one.
// A kind that runs SQL must not offer it.
func TestDefinitionSets(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []sourcekeys.Definition{
		sourcekeys.DefinitionPostgres,
		sourcekeys.DefinitionRedshift,
		sourcekeys.DefinitionSnowflake,
		sourcekeys.DefinitionBigQuery,
		sourcekeys.DefinitionMySQL,
		sourcekeys.DefinitionDatabricks,
		sourcekeys.DefinitionTrino,
	}, sourcekeys.WarehouseDefinitions)

	assert.Equal(t, append(append([]sourcekeys.Definition{}, sourcekeys.WarehouseDefinitions...), sourcekeys.DefinitionS3),
		sourcekeys.TableDefinitions, "table definitions are the warehouses plus s3")
	assert.NotContains(t, sourcekeys.WarehouseDefinitions, sourcekeys.DefinitionS3)
}

// OneOf feeds the validate:"oneof=..." tags each kind writes by hand. Its
// output is what a kind's own test compares its tag against, so the rendering
// has to be exactly the space-separated form the validator expects.
func TestOneOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "postgres redshift snowflake bigquery mysql databricks trino", sourcekeys.OneOf(sourcekeys.WarehouseDefinitions))
	assert.Equal(t, "postgres redshift snowflake bigquery mysql databricks trino s3", sourcekeys.OneOf(sourcekeys.TableDefinitions))
	assert.Equal(t, "", sourcekeys.OneOf(nil))
}
