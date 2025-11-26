package factory

import (
	"time"

	"github.com/google/uuid"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
)

type TrackingPlanCatalogFactory struct {
	trackingplan catalog.TrackingPlan
}

func NewTrackingPlanCatalogFactory() *TrackingPlanCatalogFactory {

	tp := catalog.TrackingPlan{
		ID:           uuid.New().String(),
		Name:         "default-tracking-plan",
		Version:      1,
		CreationType: "backend",
		WorkspaceID:  "workspace-id",
		CreatedAt:    time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2021, 9, 2, 0, 0, 0, 0, time.UTC),
		Description:  strptr("default-tracking-plan-description"),
	}
	return &TrackingPlanCatalogFactory{
		trackingplan: tp,
	}
}

func (f *TrackingPlanCatalogFactory) WithID(id string) *TrackingPlanCatalogFactory {
	f.trackingplan.ID = id
	return f
}

func (f *TrackingPlanCatalogFactory) WithName(name string) *TrackingPlanCatalogFactory {
	f.trackingplan.Name = name
	return f
}

func (f *TrackingPlanCatalogFactory) WithDescription(description string) *TrackingPlanCatalogFactory {
	f.trackingplan.Description = strptr(description)
	return f
}

func (f *TrackingPlanCatalogFactory) WithWorkspaceID(workspaceID string) *TrackingPlanCatalogFactory {
	f.trackingplan.WorkspaceID = workspaceID
	return f
}

func (f *TrackingPlanCatalogFactory) WithCreationType(creationType string) *TrackingPlanCatalogFactory {
	f.trackingplan.CreationType = creationType
	return f
}

func (f *TrackingPlanCatalogFactory) WithCreatedAt(createdAt time.Time) *TrackingPlanCatalogFactory {
	f.trackingplan.CreatedAt = createdAt
	return f
}

func (f *TrackingPlanCatalogFactory) WithUpdatedAt(updatedAt time.Time) *TrackingPlanCatalogFactory {
	f.trackingplan.UpdatedAt = updatedAt
	return f
}

func (f *TrackingPlanCatalogFactory) WithVersion(version int) *TrackingPlanCatalogFactory {
	f.trackingplan.Version = version
	return f
}

func (f *TrackingPlanCatalogFactory) Build() catalog.TrackingPlan {
	return f.trackingplan
}

func strptr(s string) *string {
	return &s
}
