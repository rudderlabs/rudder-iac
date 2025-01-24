package testutils

import "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"

type DataCatalogFactory struct {
	dc *localcatalog.DataCatalog
}

func NewDataCatalogFactory() *DataCatalogFactory {
	return &DataCatalogFactory{
		dc: &localcatalog.DataCatalog{},
	}
}

func (f *DataCatalogFactory) WithEvent(group string, event *localcatalog.Event) *DataCatalogFactory {
	if f.dc.Events == nil {
		f.dc.Events = make(map[localcatalog.EntityGroup][]*localcatalog.Event)
	}
	f.dc.Events[localcatalog.EntityGroup(group)] = append(f.dc.Events["default"], event)
	return f
}

func (f *DataCatalogFactory) WithProperty(group string, prop *localcatalog.Property) *DataCatalogFactory {
	if f.dc.Properties == nil {
		f.dc.Properties = make(map[localcatalog.EntityGroup][]*localcatalog.Property)
	}
	f.dc.Properties[localcatalog.EntityGroup(group)] = append(f.dc.Properties["default"], prop)
	return f
}

func (f *DataCatalogFactory) WithTrackingPlan(group string, tp *localcatalog.TrackingPlan) *DataCatalogFactory {
	if f.dc.TrackingPlans == nil {
		f.dc.TrackingPlans = make(map[localcatalog.EntityGroup]*localcatalog.TrackingPlan)
	}
	f.dc.TrackingPlans[localcatalog.EntityGroup(group)] = tp
	return f
}

func (f *DataCatalogFactory) Build() *localcatalog.DataCatalog {
	return f.dc
}
