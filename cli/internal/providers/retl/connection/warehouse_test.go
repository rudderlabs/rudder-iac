package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceWarehouseMetadata(t *testing.T) {
	t.Parallel()

	sql := WarehouseMetadata{
		SyncBehaviours:       []string{"upsert", "mirror", "full"},
		SupportsSQLModel:     true,
		SupportsSyncSettings: true,
	}

	tests := []struct {
		sourceDefinition string
		want             WarehouseMetadata
	}{
		{sourceDefinition: "bigquery", want: sql},
		{sourceDefinition: "databricks", want: sql},
		{sourceDefinition: "mysql", want: sql},
		{sourceDefinition: "postgres", want: sql},
		{sourceDefinition: "redshift", want: sql},
		{sourceDefinition: "snowflake", want: sql},
		{sourceDefinition: "trino", want: sql},
		{sourceDefinition: "s3", want: WarehouseMetadata{SyncBehaviours: []string{"upsert"}}},
		{sourceDefinition: "sftp", want: WarehouseMetadata{SyncBehaviours: []string{"upsert"}}},
		{sourceDefinition: "clickhouse", want: WarehouseMetadata{SyncBehaviours: []string{"upsert", "mirror", "full"}}},
	}

	for _, test := range tests {
		t.Run(test.sourceDefinition, func(t *testing.T) {
			t.Parallel()
			metadata, ok := SourceWarehouseMetadata(test.sourceDefinition)
			require.True(t, ok)
			assert.Equal(t, test.want, metadata)
		})
	}
}

// TestSourceWarehouseMetadataCopiesSyncBehaviours pins the isolation between
// calls: the SQL warehouses share one backing array, so a caller that sorts or
// appends in place would otherwise rewrite the table for every warehouse at
// once.
func TestSourceWarehouseMetadataCopiesSyncBehaviours(t *testing.T) {
	t.Parallel()

	metadata, ok := SourceWarehouseMetadata("postgres")
	require.True(t, ok)
	metadata.SyncBehaviours[0] = "clobbered"

	fresh, ok := SourceWarehouseMetadata("snowflake")
	require.True(t, ok)
	assert.Equal(t, []string{"upsert", "mirror", "full"}, fresh.SyncBehaviours)
}

// TestSourceWarehouseMetadataUnknown pins the unknown result: an absent
// definition must not read as a warehouse that supports nothing, or callers
// would report violations the backend would have allowed.
func TestSourceWarehouseMetadataUnknown(t *testing.T) {
	t.Parallel()

	metadata, ok := SourceWarehouseMetadata("oracle")
	assert.False(t, ok)
	assert.Equal(t, WarehouseMetadata{}, metadata)
}
