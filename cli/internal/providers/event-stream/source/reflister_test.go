package source_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	dctypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

func TestRefLister(t *testing.T) {
	t.Parallel()

	lister := source.RefLister()

	t.Run("targets the source resource type", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, source.ResourceType, lister.ResourceType)
	})

	t.Run("source references its tracking plan", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &sourceClient.EventStreamSource{
			TrackingPlan: &sourceClient.TrackingPlan{ID: "tp_1"},
		}}

		refs := lister.Refs(r)

		assert.Equal(t, []importmatcher.Ref{{EntityType: dctypes.TrackingPlanResourceType, RemoteID: "tp_1"}}, refs)
	})

	t.Run("source without a tracking plan references nothing", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &sourceClient.EventStreamSource{TrackingPlan: nil}}

		assert.Empty(t, lister.Refs(r))
	})
}
