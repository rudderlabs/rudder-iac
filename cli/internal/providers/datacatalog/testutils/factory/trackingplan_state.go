package factory

import (
	"github.com/google/uuid"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
)

type TrackingPlanArgsFactory struct {
	trackingplanArgs state.TrackingPlanArgs
}

func NewTrackingPlanArgsFactory() *TrackingPlanArgsFactory {

	args := state.TrackingPlanArgs{
		Name:        "tracking-plan-name",
		Description: "tracking-plan-description",
		LocalID:     uuid.New().String(),
		Events:      nil,
	}

	return &TrackingPlanArgsFactory{
		trackingplanArgs: args,
	}
}

func (f *TrackingPlanArgsFactory) WithName(name string) *TrackingPlanArgsFactory {
	f.trackingplanArgs.Name = name
	return f
}

func (f *TrackingPlanArgsFactory) WithDescription(description string) *TrackingPlanArgsFactory {
	f.trackingplanArgs.Description = description
	return f
}

func (f *TrackingPlanArgsFactory) WithLocalID(localID string) *TrackingPlanArgsFactory {
	f.trackingplanArgs.LocalID = localID
	return f
}

func (f *TrackingPlanArgsFactory) WithEvent(event *state.TrackingPlanEventArgs) *TrackingPlanArgsFactory {
	if f.trackingplanArgs.Events == nil {
		f.trackingplanArgs.Events = make([]*state.TrackingPlanEventArgs, 0)
	}

	f.trackingplanArgs.Events = append(f.trackingplanArgs.Events, event)
	return f
}

func (f *TrackingPlanArgsFactory) Build() state.TrackingPlanArgs {
	return f.trackingplanArgs
}

type TrackingPlanStateFactory struct {
	trackingplanState state.TrackingPlanState
}

func NewTrackingPlanStateFactory() *TrackingPlanStateFactory {

	state := state.TrackingPlanState{
		ID:           "tracking-plan-id",
		Name:         "tracking-plan-name",
		Description:  "tracking-plan-description",
		WorkspaceID:  "workspace-id",
		Version:      1,
		CreationType: "backend",
		CreatedAt:    "2021-09-01T00:00:00Z",
		UpdatedAt:    "2021-09-02T00:00:00Z",
		Events:       nil,
	}

	return &TrackingPlanStateFactory{
		trackingplanState: state,
	}
}

func (f *TrackingPlanStateFactory) WithID(id string) *TrackingPlanStateFactory {
	f.trackingplanState.ID = id
	return f
}

func (f *TrackingPlanStateFactory) WithName(name string) *TrackingPlanStateFactory {
	f.trackingplanState.Name = name
	return f
}

func (f *TrackingPlanStateFactory) WithDescription(description string) *TrackingPlanStateFactory {
	f.trackingplanState.Description = description
	return f
}

func (f *TrackingPlanStateFactory) WithWorkspaceID(workspaceID string) *TrackingPlanStateFactory {
	f.trackingplanState.WorkspaceID = workspaceID
	return f
}

func (f *TrackingPlanStateFactory) WithCreatedAt(createdAt string) *TrackingPlanStateFactory {
	f.trackingplanState.CreatedAt = createdAt
	return f
}

func (f *TrackingPlanStateFactory) WithUpdatedAt(updatedAt string) *TrackingPlanStateFactory {
	f.trackingplanState.UpdatedAt = updatedAt
	return f
}

func (f *TrackingPlanStateFactory) WithTrackingPlanArgs(args state.TrackingPlanArgs) *TrackingPlanStateFactory {
	f.trackingplanState.TrackingPlanArgs = args
	return f
}

func (f *TrackingPlanStateFactory) WithEvent(event *state.TrackingPlanEventState) *TrackingPlanStateFactory {
	if f.trackingplanState.Events == nil {
		f.trackingplanState.Events = make([]*state.TrackingPlanEventState, 0)
	}

	f.trackingplanState.Events = append(f.trackingplanState.Events, event)
	return f
}

func (f *TrackingPlanStateFactory) Build() state.TrackingPlanState {
	return f.trackingplanState
}
