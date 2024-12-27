package state

import "github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"

const (
	EntityTypeProperty     = "property"
	EntityTypeEvent        = "event"
	EntityTypeTrackingPlan = "tracking_plan"
)

type StateMapper interface {
	ToResourceData() *resources.ResourceData
	FromResourceData(*resources.ResourceData)
}
