package source

import (
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

const (
	// IDKey               = "id" // remote id
	// NameKey             = "name"
	// EnabledKey          = "enabled"
	// SourceDefinitionKey = "type"

	// TrackKey              = "track"
	// IdentifyKey           = "identify"
	// GroupKey              = "group"
	// PageKey               = "page"
	// ScreenKey             = "screen"
	// TrackingPlanKey       = "tracking_plan"
	// TrackingPlanConfigKey = "tracking_plan_config"
	// TrackingPlanIDKey     = "tracking_plan_id"

	// PropagateViolationsKey     = "propagate_violations"
	// DropUnplannedPropertiesKey = "drop_unplanned_properties"
	// DropOtherViolationsKey     = "drop_other_violations"
	// DropUnplannedEventsKey     = "drop_unplanned_events"

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
	SpecKind     = "event-stream-source"
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

// YAML structs
type SourceSpec struct {
	LocalId    string                `mapstructure:"id"`
	Name       string                `mapstructure:"name"`
	Type       string                `mapstructure:"type"`
	Enabled    *bool                 `mapstructure:"enabled"`
	Governance *sourceGovernanceSpec `mapstructure:"governance"`
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
	eventConfigSpec     `mapstructure:",squash"`
	DropUnplannedEvents *bool `json:"drop_unplanned_events" mapstructure:"drop_unplanned_events"`
}

// Resource structs
type SourceResource struct {
	ID         string
	Name       string              `diff:"name"`
	Type       string              `diff:"type"`
	Enabled    bool                `diff:"enabled"`
	Governance *GovernanceResource `diff:"governance"`
}

// GetTrackingPlanID returns the tracking plan ID from the governance config, or empty string if not set
func (sr *SourceResource) GetTrackingPlanID() string {
	if sr.Governance == nil || sr.Governance.Validations == nil || sr.Governance.Validations.TrackingPlanRef == nil {
		return ""
	}
	return sr.Governance.Validations.TrackingPlanRef.Value
}

// GetTrackingPlanConfig returns the tracking plan config from the governance config, or nil if not set
func (sr *SourceResource) GetTrackingPlanConfig() *TrackingPlanConfigResource {
	if sr.Governance == nil || sr.Governance.Validations == nil {
		return nil
	}
	return sr.Governance.Validations.Config
}

type WorkspaceRemoteIDMapping struct {
	WorkspaceId string
	RemoteId    string
}

type GovernanceResource struct {
	Validations *ValidationsResource `diff:"validations"`
}

type ValidationsResource struct {
	TrackingPlanRef *resources.PropertyRef      `diff:"tracking_plan"`
	Config          *TrackingPlanConfigResource `diff:"config"`
}

type TrackingPlanConfigResource struct {
	Track    *TrackConfigResource `diff:"track"`
	Identify *EventConfigResource `diff:"identify"`
	Group    *EventConfigResource `diff:"group"`
	Page     *EventConfigResource `diff:"page"`
	Screen   *EventConfigResource `diff:"screen"`
}

type EventConfigResource struct {
	PropagateViolations     *bool `diff:"propagate_violations"`
	DropUnplannedProperties *bool `diff:"drop_unplanned_properties"`
	DropOtherViolations     *bool `diff:"drop_other_violations"`
}

type TrackConfigResource struct {
	*EventConfigResource
	DropUnplannedEvents *bool `diff:"drop_unplanned_events"`
}

type SourceState struct {
	ID             string
	TrackingPlanID string
}

type SourceRemote struct {
	*sourceClient.EventStreamSource
}

func (sr SourceRemote) GetResourceMetadata() *provider.RemoteResourceMetadata {
	return &provider.RemoteResourceMetadata{
		ID:          sr.ID,
		ExternalID:  sr.ExternalID,
		WorkspaceID: sr.WorkspaceID,
		Name:        sr.Name,
	}
}
