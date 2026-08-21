package source

import (
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// RefLister returns the import --merge reference lister for event stream
// sources. A source references its tracking plan, so the precheck can flag a
// source importable whose tracking plan is deleted locally but not yet applied.
func RefLister() importmatcher.RefLister {
	return importmatcher.RefLister{
		ResourceType: ResourceType,
		Refs:         sourceRefs,
	}
}

func sourceRefs(r *resources.RemoteResource) []importmatcher.Ref {
	source := r.Data.(*sourceClient.EventStreamSource)
	if source.TrackingPlan == nil || source.TrackingPlan.ID == "" {
		return nil
	}
	return []importmatcher.Ref{{EntityType: types.TrackingPlanResourceType, RemoteID: source.TrackingPlan.ID}}
}
