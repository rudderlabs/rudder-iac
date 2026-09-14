package connection

import (
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sqlModelKind = SourceKind{
	Kind:         "retl-source-sql-model",
	ResourceType: "retl-source-sql-model",
	SourceType:   retlClient.ModelSourceType,
}

// TestSourceKinds pins the whole table, not just the sql model row: registering
// a kind is a contract change — the kind's handler has to publish the shared
// source keys and the warehouse table has to cover its source definitions — so
// a new row must not slip in on a one-line edit.
func TestSourceKinds(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []SourceKind{sqlModelKind}, SourceKinds)
}

func TestSourceKindLookups(t *testing.T) {
	t.Parallel()

	t.Run("by kind", func(t *testing.T) {
		t.Parallel()
		found, ok := SourceKindByKind("retl-source-sql-model")
		require.True(t, ok)
		assert.Equal(t, sqlModelKind, found)

		_, ok = SourceKindByKind("retl-source-table")
		assert.False(t, ok)
	})

	t.Run("by resource type", func(t *testing.T) {
		t.Parallel()
		found, ok := SourceKindByResourceType("retl-source-sql-model")
		require.True(t, ok)
		assert.Equal(t, sqlModelKind, found)

		_, ok = SourceKindByResourceType("event-stream-source")
		assert.False(t, ok)
	})

	t.Run("by source type", func(t *testing.T) {
		t.Parallel()
		found, ok := SourceKindBySourceType(retlClient.ModelSourceType)
		require.True(t, ok)
		assert.Equal(t, sqlModelKind, found)

		_, ok = SourceKindBySourceType(retlClient.TableSourceType)
		assert.False(t, ok)
	})
}

// TestSharedSourceKeysMatchSQLModel guards the contract every rETL source
// handler honours: connection validation reads these keys off a source's graph
// data without knowing which kind it is.
func TestSharedSourceKeysMatchSQLModel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, sqlmodel.SourceDefinitionKey, SourceDefinitionKey)
	assert.Equal(t, sqlmodel.PrimaryKeyKey, SourcePrimaryKeyKey)
	assert.Equal(t, sqlmodel.EnabledKey, SourceEnabledKey)
}

func TestParseSourceRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     string
		want    *resources.PropertyRef
		wantErr string
	}{
		{
			name: "sql model",
			ref:  "#retl-source-sql-model:users",
			want: &resources.PropertyRef{URN: "retl-source-sql-model:users", Property: "id"},
		},
		{
			name: "surrounding whitespace",
			ref:  "  #retl-source-sql-model:users  ",
			want: &resources.PropertyRef{URN: "retl-source-sql-model:users", Property: "id"},
		},
		{
			name:    "wrong family",
			ref:     "#event-stream-source:users",
			wantErr: `source reference "#event-stream-source:users" is not a rETL source: expected #retl-source-sql-model:<id>`,
		},
		{
			name:    "not a reference",
			ref:     "users",
			wantErr: `invalid source reference "users": expected #retl-source-sql-model:<id>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ref, err := parseSourceRef(test.ref)
			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, ref)
		})
	}
}
