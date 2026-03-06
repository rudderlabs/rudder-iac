package source

import (
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
)

// SourceSpecV1 mirrors SourceSpec but with V1-specific validation tags.
// The TrackingPlanSpecV1 nested struct uses the V1 ref pattern for validation.
type SourceSpecV1 struct {
	LocalID          string                  `json:"id"         mapstructure:"id"         validate:"required"`
	Name             string                  `json:"name"       mapstructure:"name"       validate:"required"`
	SourceDefinition string                  `json:"type"       mapstructure:"type"       validate:"required,oneof=java dotnet php flutter cordova rust react_native python ios android javascript go node ruby unity"`
	Enabled          *bool                   `json:"enabled"    mapstructure:"enabled"`
	Governance       *SourceGovernanceSpecV1 `json:"governance" mapstructure:"governance"`
}

type SourceGovernanceSpecV1 struct {
	TrackingPlan *TrackingPlanSpecV1 `json:"validations" mapstructure:"validations"`
}

type TrackingPlanSpecV1 struct {
	Ref    string                           `json:"tracking_plan" mapstructure:"tracking_plan" validate:"required,pattern=tracking_plan_ref"`
	Config *esSource.TrackingPlanConfigSpec `json:"config"        mapstructure:"config"        validate:"required"`
}
