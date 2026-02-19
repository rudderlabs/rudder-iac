package source

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

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

	PropagateViolationsKey     = "propagate_violations"
	DropUnplannedPropertiesKey = "drop_unplanned_properties"
	DropOtherViolationsKey     = "drop_other_violations"
	DropUnplannedEventsKey     = "drop_unplanned_events"

	// YAML keys for import spec
	TrackingPlanRefYAMLKey         = "tracking_plan"
	TrackingPlanConfigYAMLKey      = "config"
	GovernanceYAMLKey              = "governance"
	ValidationsYAMLKey             = "validations"
	TrackYAMLKey                   = "track"
	IdentifyYAMLKey                = "identify"
	GroupYAMLKey                   = "group"
	PageYAMLKey                    = "page"
	ScreenYAMLKey                  = "screen"
	PropagateViolationsYAMLKey     = "propagate_violations"
	DropUnplannedPropertiesYAMLKey = "drop_unplanned_properties"
	DropOtherViolationsYAMLKey     = "drop_other_violations"
	DropUnplannedEventsYAMLKey     = "drop_unplanned_events"

	ResourceType = "event-stream-source"
	ResourceKind = "event-stream-source"
	MetadataName = "event-stream-source"
	ImportPath   = "sources"
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

// SourceSpec mirrors the YAML spec structure. JSON tags enable the typed rule engine's
// json.Marshal/Unmarshal round-trip; validate tags drive go-playground/validator checks.
// Pointer fields let the validator skip inner required tags when the parent block is absent.
type SourceSpec struct {
	LocalID          string                `json:"id"         mapstructure:"id"         validate:"required"`
	Name             string                `json:"name"       mapstructure:"name"       validate:"required"`
	SourceDefinition string                `json:"type"       mapstructure:"type"       validate:"required,oneof=java dotnet php flutter cordova rust react_native python ios android javascript go node ruby unity"`
	Enabled          *bool                 `json:"enabled"    mapstructure:"enabled"`
	Governance       *SourceGovernanceSpec `json:"governance" mapstructure:"governance"`
}

type SourceGovernanceSpec struct {
	TrackingPlan *TrackingPlanSpec `json:"validations" mapstructure:"validations"`
}

type TrackingPlanSpec struct {
	Ref    string                  `json:"tracking_plan" mapstructure:"tracking_plan" validate:"required,pattern=legacy_tracking_plan_ref"`
	Config *TrackingPlanConfigSpec `json:"config"        mapstructure:"config"        validate:"required"`
}

type TrackingPlanConfigSpec struct {
	Track    *TrackConfigSpec `json:"track"    mapstructure:"track"`
	Identify *EventConfigSpec `json:"identify" mapstructure:"identify"`
	Group    *EventConfigSpec `json:"group"    mapstructure:"group"`
	Page     *EventConfigSpec `json:"page"     mapstructure:"page"`
	Screen   *EventConfigSpec `json:"screen"   mapstructure:"screen"`
}

type EventConfigSpec struct {
	PropagateViolations     *bool `json:"propagate_violations"      mapstructure:"propagate_violations"`
	DropUnplannedProperties *bool `json:"drop_unplanned_properties" mapstructure:"drop_unplanned_properties"`
	DropOtherViolations     *bool `json:"drop_other_violations"     mapstructure:"drop_other_violations"`
}

type TrackConfigSpec struct {
	EventConfigSpec         `mapstructure:",squash"`
	DropUnplannedEvents *bool `json:"drop_unplanned_events" mapstructure:"drop_unplanned_events"`
}

// Resource structs
type sourceResource struct {
	LocalID          string
	Name             string
	SourceDefinition string
	Enabled          bool
	Governance       *governanceResource
	ImportMetadata   map[string]*WorkspaceRemoteIDMapping
}

type WorkspaceRemoteIDMapping struct {
	WorkspaceId string
	RemoteId    string
}

type governanceResource struct {
	Validations *validationsResource
}

type validationsResource struct {
	TrackingPlanRef *resources.PropertyRef
	Config          *trackingPlanConfigResource
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
