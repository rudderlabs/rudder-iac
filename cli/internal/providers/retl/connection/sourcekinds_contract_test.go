package connection

import (
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The contract SourceKinds' doc comment demands of every registered kind:
// the handler publishes the three shared source keys in its graph data, and an
// "id" in its state output for a PropertyRef to resolve. sqlmodel is covered by
// its own suite; these prove it for the table kind, which is the reason the row
// below it is safe to register.
//
// Driven through the table handler's exported surface rather than its spec
// struct, so a change to how the data is assembled — not just to the keys —
// fails here too.

func tableResourceData(t *testing.T, spec map[string]any) resources.ResourceData {
	t.Helper()

	handler := table.NewHandler(nil, "retl")
	require.NoError(t, handler.LoadSpec("", &specs.Spec{Spec: spec}))

	graphResources, err := handler.GetResources()
	require.NoError(t, err)
	require.Len(t, graphResources, 1)
	return graphResources[0].Data()
}

func TestTableSourcePublishesSharedSourceKeys(t *testing.T) {
	t.Parallel()

	data := tableResourceData(t, map[string]any{
		"id":                "users",
		"display_name":      "Users",
		"account_id":        "acc-1",
		"source_definition": "postgres",
		"primary_key":       "id",
		"schema":            "public",
		"table":             "users",
	})

	assert.Equal(t, "postgres", data[SourceDefinitionKey])
	assert.Equal(t, "id", data[SourcePrimaryKeyKey])
	assert.Equal(t, true, data[SourceEnabledKey], "a spec with no enabled key creates an enabled source")
}

// s3 is the shape that makes the shared keys worth pinning: it has no primary
// key, and the contract is that the key is still *present* — empty rather than
// absent — so a connection reads both source definitions through one shape
// instead of probing for the key first.
func TestTableSourcePublishesSharedSourceKeysForS3(t *testing.T) {
	t.Parallel()

	data := tableResourceData(t, map[string]any{
		"id":                "events",
		"display_name":      "Events",
		"account_id":        "acc-1",
		"source_definition": "s3",
		"bucket_name":       "acme-events",
		"object_prefix":     "retl/",
		"enabled":           false,
	})

	assert.Equal(t, "s3", data[SourceDefinitionKey])
	require.Contains(t, data, SourcePrimaryKeyKey, "primary_key must be present even where it is not used")
	assert.Equal(t, "", data[SourcePrimaryKeyKey])
	assert.Equal(t, false, data[SourceEnabledKey])
}

// A connection resolves its source through a PropertyRef on "id", so the kind's
// state output has to carry the remote id under that key.
func TestTableSourceOutputCarriesID(t *testing.T) {
	t.Parallel()

	collection := resources.NewRemoteResources()
	collection.Set(table.ResourceType, map[string]*resources.RemoteResource{
		"src-1": {
			ID:         "src-1",
			ExternalID: "users",
			Data: retlClient.RETLSource{
				ID:                   "src-1",
				Name:                 "Users",
				ExternalID:           "users",
				AccountID:            "acc-1",
				SourceType:           retlClient.TableSourceType,
				SourceDefinitionName: "postgres",
				IsEnabled:            true,
				Config:               retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "public", Table: "users"},
			},
		},
	})

	remoteState, err := table.NewHandler(nil, "retl").MapRemoteToState(collection)
	require.NoError(t, err)

	resourceState := remoteState.GetResource(resources.URN("users", table.ResourceType))
	require.NotNil(t, resourceState)

	// The property parseSourceRef builds every source reference on.
	assert.Equal(t, "src-1", resourceState.Output["id"])
}

// The warehouse table is keyed by source definition, not by source kind, so
// registering the table kind needs no additions to it — but that is only true
// while every definition a table source accepts is already covered. These pin
// the two facts a connection to a table source depends on.
func TestWarehouseMetadataCoversTableSourceDefinitions(t *testing.T) {
	t.Parallel()

	t.Run("s3 is upsert-only and carries no sync settings", func(t *testing.T) {
		t.Parallel()

		metadata, ok := SourceWarehouseMetadata("s3")
		require.True(t, ok)
		assert.Equal(t, WarehouseMetadata{SyncBehaviours: []string{"upsert"}}, metadata)
	})

	t.Run("every warehouse a table source accepts is known", func(t *testing.T) {
		t.Parallel()

		// The source_definition values the table spec kind validates against.
		for _, definition := range []string{
			"postgres", "redshift", "snowflake", "bigquery", "mysql", "databricks", "trino", "s3",
		} {
			_, ok := SourceWarehouseMetadata(definition)
			assert.True(t, ok, "no warehouse metadata for %q, which a table source may name", definition)
		}
	})
}
