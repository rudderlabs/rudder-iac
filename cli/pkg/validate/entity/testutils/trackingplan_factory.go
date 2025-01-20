package testutils

import "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"

type LocalCatalogTrackingPlanFactory struct {
	tp *localcatalog.TrackingPlan
}

func NewLocalCatalogTrackingPlanFactory() *LocalCatalogTrackingPlanFactory {
	return &LocalCatalogTrackingPlanFactory{
		tp: &localcatalog.TrackingPlan{},
	}
}

func (f *LocalCatalogTrackingPlanFactory) WithLocalID(localID string) *LocalCatalogTrackingPlanFactory {
	f.tp.LocalID = localID
	return f
}

func (f *LocalCatalogTrackingPlanFactory) WithName(name string) *LocalCatalogTrackingPlanFactory {
	f.tp.Name = name
	return f
}

func (f *LocalCatalogTrackingPlanFactory) WithDescription(description string) *LocalCatalogTrackingPlanFactory {
	f.tp.Description = description
	return f
}

func (f *LocalCatalogTrackingPlanFactory) WithRule(rule *localcatalog.TPRule) *LocalCatalogTrackingPlanFactory {
	f.tp.Rules = append(f.tp.Rules, rule)
	return f
}

func (f *LocalCatalogTrackingPlanFactory) WithExpandedEvent(eventProp *localcatalog.TPEvent) *LocalCatalogTrackingPlanFactory {
	f.tp.EventProps = append(f.tp.EventProps, eventProp)
	return f
}

func (f *LocalCatalogTrackingPlanFactory) Build() *localcatalog.TrackingPlan {
	return f.tp
}
