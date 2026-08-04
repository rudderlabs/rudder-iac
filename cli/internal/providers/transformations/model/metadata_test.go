package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
)

// Metadata must carry WorkspaceID: import --merge stamps matched-resource
// manifest entries with it (via BaseHandler), and an empty workspace ID would
// place the entry in a phantom "" group that apply cannot scope to a workspace.
func TestMetadataCarriesWorkspaceID(t *testing.T) {
	t.Parallel()

	t.Run("transformation", func(t *testing.T) {
		t.Parallel()

		r := RemoteTransformation{Transformation: &transformations.Transformation{
			ID:          "3A83iCuj",
			ExternalID:  "transformation1",
			Name:        "Transformation1",
			WorkspaceID: "ws-123",
		}}

		assert.Equal(t, handler.RemoteResourceMetadata{
			ID:          "3A83iCuj",
			ExternalID:  "transformation1",
			Name:        "Transformation1",
			WorkspaceID: "ws-123",
		}, r.Metadata())
	})

	t.Run("library", func(t *testing.T) {
		t.Parallel()

		r := RemoteLibrary{TransformationLibrary: &transformations.TransformationLibrary{
			ID:          "3A83Utg4",
			ExternalID:  "getuseraddress",
			Name:        "getUserAddress",
			WorkspaceID: "ws-123",
		}}

		assert.Equal(t, handler.RemoteResourceMetadata{
			ID:          "3A83Utg4",
			ExternalID:  "getuseraddress",
			Name:        "getUserAddress",
			WorkspaceID: "ws-123",
		}, r.Metadata())
	})
}
