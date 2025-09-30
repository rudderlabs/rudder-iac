package source

import "github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"

const (
	IDKey               = "id" // remote id
	NameKey             = "name"
	EnabledKey          = "enabled"
	SourceDefinitionKey = "type"

	TrackKey              = "track"
	IdentifyKey           = "identify"
	GroupKey              = "group"
	PageKey               = "page"
	ScreenKey             = "screen"
	TrackingPlanKey       = "tracking_plan"
	TrackingPlanConfigKey = "tracking_plan_config"
	TrackingPlanIDKey     = "tracking_plan_id"

	PropagateViolationsKey = "propagate_violations"
	DropUnplannedPropertiesKey = "drop_unplanned_properties"
	DropOtherViolationsKey = "drop_other_violations"
	DropUnplannedEventsKey = "drop_unplanned_events"

	ResourceType = "event-stream-source"
)

var sourceDefinitions = []string{
	"java",
	"dotnet",
	"php",
	"flutter",
	"cordova",
	"rust",
	"react_native",
	"python",
	"ios",
	"android",
	"javascript",
	"go",
	"node",
	"ruby",
	"unity",
}

// YAML structs
type sourceSpec struct {
	LocalId          string                `mapstructure:"id"`
	Name             string                `mapstructure:"name"`
	SourceDefinition string                `mapstructure:"type"`
	Enabled          bool                  `mapstructure:"enabled"`
	Governance       *sourceGovernanceSpec `mapstructure:"governance"`
}

type sourceGovernanceSpec struct {
	TrackingPlan *trackingPlanSpec `mapstructure:"validations"`
}

type trackingPlanSpec struct {
	Ref    string                  `mapstructure:"tracking_plan"`
	Config *trackingPlanConfigSpec `mapstructure:"config"`
}

type trackingPlanConfigSpec struct {
	Track    *trackConfigSpec `mapstructure:"track"`
	Identify *eventConfigSpec `mapstructure:"identify"`
	Group    *eventConfigSpec `mapstructure:"group"`
	Page     *eventConfigSpec `mapstructure:"page"`
	Screen   *eventConfigSpec `mapstructure:"screen"`
}

type eventConfigSpec struct {
	PropagateViolations     *bool `mapstructure:"propagate_violations"`
	DropUnplannedProperties *bool `mapstructure:"drop_unplanned_properties"`
	DropOtherViolations     *bool `mapstructure:"drop_other_violations"`
}

type trackConfigSpec struct {
	eventConfigSpec `mapstructure:",squash"`
	DropUnplannedEvents *bool `json:"drop_unplanned_events" mapstructure:"drop_unplanned_events"`
}

// Resource structs
type sourceResource struct {
	LocalId          string
	Name             string
	SourceDefinition string
	Enabled          bool
	Governance       *governanceResource
}

type governanceResource struct {
	TrackingPlan *trackingPlanResource
}

type trackingPlanResource struct {
	Ref    *resources.PropertyRef
	Config *trackingPlanConfigResource
}

type trackingPlanConfigResource struct {
	Track    *TrackConfigResource
	Identify *EventConfigResource
	Group    *EventConfigResource
	Page     *EventConfigResource
	Screen   *EventConfigResource
}

type EventConfigResource struct {
	PropagateViolations     *bool
	DropUnplannedProperties *bool
	DropOtherViolations     *bool
}

type TrackConfigResource struct {
	*EventConfigResource
	DropUnplannedEvents *bool
}
