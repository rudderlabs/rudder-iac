package sqlmodel_test

import (
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatForExportSkipsMatched(t *testing.T) {
	t.Parallel()

	h := sqlmodel.NewHandler(nil, "retl")

	matched := &resources.RemoteResource{
		ID:          "src1",
		ExternalID:  "orders",
		Data:        &retlClient.RETLSource{ID: "src1", Name: "Orders", AccountID: "acc_1", WorkspaceID: "ws1"},
		MatchedWith: resources.NewResource("orders", sqlmodel.ResourceType, resources.ResourceData{}, []string{}),
	}
	unmatched := &resources.RemoteResource{
		ID:         "src2",
		ExternalID: "customers",
		Data:       &retlClient.RETLSource{ID: "src2", Name: "Customers", AccountID: "acc_1", WorkspaceID: "ws1"},
	}

	collection := resources.NewRemoteResources()
	collection.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{"src1": matched, "src2": unmatched})

	entities, entries, err := h.FormatForExport(collection, nil, nil)
	require.NoError(t, err)

	// One spec file per unmatched model; the matched model writes nothing.
	require.Len(t, entities, 1)
	assert.Contains(t, entities[0].RelativePath, "customers.yaml")
	assert.NotContains(t, entities[0].RelativePath, "orders")

	// Manifest entries for both, the matched one under its adopted local URN.
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws1", URN: "retl-source-sql-model:orders", RemoteID: "src1"},
		{WorkspaceID: "ws1", URN: "retl-source-sql-model:customers", RemoteID: "src2"},
	}, entries)
}
