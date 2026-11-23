package datacatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Import writes V1 specs, and the import resolver copies MetadataRef verbatim
// into them when a spec references a resource the project already has, so it
// must be the V1 reference form.
func TestResourceGraph_MetadataRefIsV1Reference(t *testing.T) {
	graph, err := createResourceGraph(&localcatalog.DataCatalog{
		Properties:    []localcatalog.PropertyV1{{LocalID: "user-id", Name: "user_id", Type: "string"}},
		Events:        []localcatalog.EventV1{{LocalID: "signed-in", Name: "signed_in", Type: "track"}},
		CustomTypes:   []localcatalog.CustomTypeV1{{LocalID: "email", Name: "Email", Type: "string"}},
		Categories:    []localcatalog.CategoryV1{{LocalID: "auth", Name: "Auth"}},
		TrackingPlans: []*localcatalog.TrackingPlanV1{{LocalID: "web", Name: "Web"}},
	})
	require.NoError(t, err)

	for urn, want := range map[string]string{
		resources.URN("user-id", types.PropertyResourceType): "#property:user-id",
		resources.URN("signed-in", types.EventResourceType):  "#event:signed-in",
		resources.URN("email", types.CustomTypeResourceType): "#custom-type:email",
		resources.URN("auth", types.CategoryResourceType):    "#category:auth",
		resources.URN("web", types.TrackingPlanResourceType): "#tracking-plan:web",
	} {
		r, ok := graph.GetResource(urn)
		require.True(t, ok, urn)
		assert.Equal(t, want, r.FileMetadata().MetadataRef, urn)
	}
}
