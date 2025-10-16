package state

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
)

const (
	PropertiesIdentity       = "properties"
	TraitsIdentity           = "traits"
	ContextTraitsIdentity    = "context.traits"
	TrackingPlanResourceType = "tracking-plan"
)

type TrackingPlanState struct {
	TrackingPlanArgs
	ID           string
	Name         string
	Description  string
	Version      int
	CreationType string
	WorkspaceID  string
	CreatedAt    string
	UpdatedAt    string
	Events       []*TrackingPlanEventState
}

func (t *TrackingPlanState) LocalIDForCatalogEventID(eventID string) string {
	for _, event := range t.Events {
		if event.EventID == eventID {
			return event.LocalID
		}
	}
	return ""
}

func (t *TrackingPlanState) CatalogEventIDForLocalID(localID string) string {
	for _, event := range t.Events {
		if event.LocalID == localID {
			return event.EventID
		}
	}
	return ""
}

type TrackingPlanEventState struct {
	ID      string
	LocalID string
	EventID string
}

func (t *TrackingPlanEventState) GetLocalID() string {
	return t.LocalID
}

func (t *TrackingPlanState) EventByLocalID(localID string) *TrackingPlanEventState {
	for _, event := range t.Events {
		if event.LocalID == localID {
			return event
		}
	}
	return nil
}

type TrackingPlanPropertyState struct {
	Name        string
	LocalID     string
	Description string
	Type        string
	Config      map[string]interface{}
	Required    bool
}

type TrackingPlanArgsDiff struct {
	Added   []*TrackingPlanEventArgs
	Updated []*TrackingPlanEventArgs
	Deleted []*TrackingPlanEventArgs
}

func (t *TrackingPlanState) ToResourceData() resources.ResourceData {

	var (
		events []map[string]interface{}
	)

	for _, event := range t.Events {

		events = append(events, map[string]interface{}{
			"id":      event.ID,
			"eventId": event.EventID,
			"localId": event.LocalID,
		})
	}

	return resources.ResourceData{
		"id":               t.ID,
		"name":             t.Name,
		"description":      t.Description,
		"version":          t.Version,
		"creationType":     t.CreationType,
		"workspaceId":      t.WorkspaceID,
		"createdAt":        t.CreatedAt,
		"updatedAt":        t.UpdatedAt,
		"events":           events,
		"trackingPlanArgs": map[string]interface{}(t.TrackingPlanArgs.ToResourceData()),
	}
}

func (t *TrackingPlanState) FromResourceData(from resources.ResourceData) {

	t.ID = MustString(from, "id")
	t.Name = MustString(from, "name")
	t.Description = MustString(from, "description")
	// version can be either an int or a float64
	// in our old stateful approach, we used to get the version as a float64 as we used json.Unmarshall to decode the state api's response into a map[string]interface{}
	// in the stateless approach, we derive the state from the remote TrackingPlan which is a strongly typed struct where the version field is of type int
	t.Version = Int(from, "version", 0)
	if t.Version == 0 {
		t.Version = int(Float64(from, "version", 0))
	}

	t.CreationType = MustString(from, "creationType")
	t.WorkspaceID = MustString(from, "workspaceId")
	t.CreatedAt = MustString(from, "createdAt")
	t.UpdatedAt = MustString(from, "updatedAt")
	t.TrackingPlanArgs.FromResourceData(
		MustMapStringInterface(from, "trackingPlanArgs"),
	)

	events := InterfaceSlice(from, "events", nil)
	if len(events) == 0 {
		return
	}

	tpEvents := make([]*TrackingPlanEventState, len(events))
	for idx, event := range events {
		event := event.(map[string]interface{})

		tpEvents[idx] = &TrackingPlanEventState{
			ID:      MustString(event, "id"),
			EventID: MustString(event, "eventId"),
			LocalID: MustString(event, "localId"),
		}
	}

	t.Events = tpEvents
}

// FromRemoteTrackingPlan converts from catalog.TrackingPlan to TrackingPlanState
func (t *TrackingPlanState) FromRemoteTrackingPlan(trackingPlan *catalog.TrackingPlanWithIdentifiers, collection *resources.ResourceCollection) error {
	t.ID = trackingPlan.ID
	t.Name = trackingPlan.Name
	t.WorkspaceID = trackingPlan.WorkspaceID
	t.Version = trackingPlan.Version
	t.CreationType = trackingPlan.CreationType
	t.CreatedAt = trackingPlan.CreatedAt.String()
	t.UpdatedAt = trackingPlan.UpdatedAt.String()
	if trackingPlan.Description != nil {
		t.Description = *trackingPlan.Description
	}

	events := make([]*TrackingPlanEventState, 0, len(trackingPlan.Events))
	for _, event := range trackingPlan.Events {
		events = append(events, &TrackingPlanEventState{
			// we dont set the tracking plan event ID in the stateless approach
			ID:      "",
			EventID: event.ID,
			LocalID: event.ExternalID,
		})
	}
	t.Events = events

	tpArgs := TrackingPlanArgs{}
	tpArgs.Name = trackingPlan.Name
	tpArgs.LocalID = trackingPlan.ExternalID
	if trackingPlan.Description != nil {
		tpArgs.Description = *trackingPlan.Description
	}

	for _, remoteTPEvent := range trackingPlan.Events {
		eventArgs := &TrackingPlanEventArgs{
			ID:              remoteTPEvent.ID,
			LocalID:         remoteTPEvent.ExternalID,
			AllowUnplanned:  remoteTPEvent.AdditionalProperties,
			IdentitySection: remoteTPEvent.IdentitySection,
		}

		properties := make([]*TrackingPlanPropertyArgs, 0, len(remoteTPEvent.Properties))
		for _, remoteProp := range remoteTPEvent.Properties {
			propArgs := &TrackingPlanPropertyArgs{}
			propArgs.FromRemoteTrackingPlanProperty(remoteProp, collection, false)
			properties = append(properties, propArgs)
		}
		eventArgs.Properties = properties

		variants := make([]Variant, len(remoteTPEvent.Variants))
		for idx, variant := range remoteTPEvent.Variants {
			v := Variant{}
			v.FromRemoteVariant(variant, collection.GetURNByID, false)
			variants[idx] = v
		}
		eventArgs.Variants = variants

		tpArgs.Events = append(tpArgs.Events, eventArgs)
	}
	t.TrackingPlanArgs = tpArgs

	return nil
}

// Encapsulates the catalog argument which is added as a resource
// when registering the tracking plan
type TrackingPlanArgs struct {
	Name        string
	LocalID     string
	Description string
	Events      []*TrackingPlanEventArgs
}

func (args TrackingPlanArgs) Diff(other TrackingPlanArgs) *TrackingPlanArgsDiff {

	diffed := &TrackingPlanArgsDiff{
		Added:   make([]*TrackingPlanEventArgs, 0),
		Updated: make([]*TrackingPlanEventArgs, 0),
		Deleted: make([]*TrackingPlanEventArgs, 0),
	}

	for _, otherEvent := range other.Events {
		if args.EventByLocalID(otherEvent.LocalID) == nil {
			diffed.Added = append(diffed.Added, otherEvent)
		}
	}

	for _, event := range args.Events {

		otherEvent := other.EventByLocalID(event.LocalID)

		if otherEvent == nil {
			diffed.Deleted = append(diffed.Deleted, event)
			continue
		}

		if event.Diff(otherEvent) {
			diffed.Updated = append(diffed.Updated, otherEvent)
		}

	}

	return diffed
}

type TrackingPlanEventArgs struct {
	ID              any
	LocalID         string
	AllowUnplanned  bool
	IdentitySection string
	Properties      []*TrackingPlanPropertyArgs
	Variants        Variants
}

func (args *TrackingPlanEventArgs) GetLocalID() string {
	return args.LocalID
}

func (args *TrackingPlanEventArgs) Diff(other *TrackingPlanEventArgs) bool {
	if args.LocalID != other.LocalID {
		return true
	}

	if args.AllowUnplanned != other.AllowUnplanned {
		return true
	}

	if args.IdentitySection != other.IdentitySection {
		return true
	}

	if len(args.Properties) != len(other.Properties) {
		return true
	}

	for _, prop := range args.Properties {

		otherProp := other.PropertyByLocalID(prop.LocalID)
		if otherProp == nil {
			return true
		}

		if prop.Diff(otherProp) {
			return true
		}
	}

	if args.Variants.Diff(other.Variants) {
		return true
	}

	return false
}

func (args *TrackingPlanEventArgs) PropertyByLocalID(id string) *TrackingPlanPropertyArgs {
	for _, prop := range args.Properties {
		if prop.LocalID == id {
			return prop
		}
	}
	return nil
}

type TrackingPlanPropertyArgs struct {
	ID                   any
	LocalID              string
	Required             bool
	Properties           []*TrackingPlanPropertyArgs `json:"properties,omitempty"`
	AdditionalProperties bool                        `json:"additionalProperties"`
}

func (args *TrackingPlanPropertyArgs) GetLocalID() string {
	return args.LocalID
}

func (args *TrackingPlanPropertyArgs) Diff(other *TrackingPlanPropertyArgs) bool {
	if args.LocalID != other.LocalID {
		return true
	}

	if args.Required != other.Required {
		return true
	}

	// Compare nested properties
	if len(args.Properties) != len(other.Properties) {
		return true
	}

	for _, prop := range args.Properties {
		otherProp := other.PropertyByLocalID(prop.LocalID)
		if otherProp == nil {
			return true
		}

		if prop.Diff(otherProp) {
			return true
		}
	}

	return false
}

// Helper method to find nested property by LocalID
func (args *TrackingPlanPropertyArgs) PropertyByLocalID(id string) *TrackingPlanPropertyArgs {
	for _, prop := range args.Properties {
		if prop.LocalID == id {
			return prop
		}
	}
	return nil
}

func (args *TrackingPlanPropertyArgs) FromCatalogTrackingPlanEventProperty(prop *localcatalog.TPEventProperty, urnFromRef func(string) string) error {

	urn := urnFromRef(prop.Ref)
	if urn == "" {
		return fmt.Errorf("unable to resolve ref to the property urn: %s", prop.Ref)
	}

	args.ID = resources.PropertyRef{
		URN:      urn,
		Property: "id",
	}
	args.LocalID = prop.LocalID
	args.Required = prop.Required

	// Handle nested properties recursively
	if len(prop.Properties) > 0 {
		nestedProperties := make([]*TrackingPlanPropertyArgs, 0, len(prop.Properties))
		for _, nestedProp := range prop.Properties {
			nestedArgs := &TrackingPlanPropertyArgs{}
			if err := nestedArgs.FromCatalogTrackingPlanEventProperty(nestedProp, urnFromRef); err != nil {
				return fmt.Errorf("processing nested property %s: %w", nestedProp.LocalID, err)
			}
			nestedProperties = append(nestedProperties, nestedArgs)
		}
		// sort the nested properties array by the localID
		utils.SortByLocalID(nestedProperties)
		args.Properties = nestedProperties
		// set additionalProperties to true if there are nested properties
		args.AdditionalProperties = true
	}

	return nil
}

// FromRemoteTrackingPlanProperty converts a remote tracking plan property into an TrackingPlanPropertyArgs struct
// usePropertyRefsForDependencies is used to determine if the property ID should be converted to a propertyRef or not
// for TrackingPlanArgs(which becomes the state's input field later) we need to convert propertyIDs into propertyRefs
// for TrackingPlanState(which becomes the state's output field later), we use the propertyID as is
func (args *TrackingPlanPropertyArgs) FromRemoteTrackingPlanProperty(remoteProp *catalog.TrackingPlanEventProperty, collection *resources.ResourceCollection, usePropertyRefsForDependencies bool) error {
	if usePropertyRefsForDependencies {
		urn, err := collection.GetURNByID(PropertyResourceType, remoteProp.ID)
		if err != nil {
			return fmt.Errorf("getting URN for property %s: %w", remoteProp.ID, err)
		}

		args.ID = resources.PropertyRef{
			URN:      urn,
			Property: "id",
		}
	} else {
		args.ID = remoteProp.ID
	}

	prop, ok := collection.GetByID(PropertyResourceType, remoteProp.ID)
	if !ok {
		return fmt.Errorf("getting property %s from resourceCollection: %w", remoteProp.ID, resources.ErrRemoteResourceNotFound)
	}
	args.LocalID = prop.ExternalID
	args.Required = remoteProp.Required

	// Handle nested properties recursively
	if len(remoteProp.Properties) > 0 {
		nestedProperties := make([]*TrackingPlanPropertyArgs, 0, len(remoteProp.Properties))
		for _, nestedProp := range remoteProp.Properties {
			nestedArgs := &TrackingPlanPropertyArgs{}
			if err := nestedArgs.FromRemoteTrackingPlanProperty(nestedProp, collection, usePropertyRefsForDependencies); err != nil {
				return fmt.Errorf("processing nested property %s: %w", nestedProp.ID, err)
			}
			nestedProperties = append(nestedProperties, nestedArgs)
		}
		// sort the nested properties array by the localID
		utils.SortByLocalID(nestedProperties)
		args.Properties = nestedProperties
		// set additionalProperties to true if there are nested properties
		args.AdditionalProperties = true
	}

	return nil
}

// ToResourceData converts TrackingPlanPropertyArgs to resource data map
func (args *TrackingPlanPropertyArgs) ToResourceData() map[string]interface{} {
	propMap := map[string]interface{}{
		"id":                   args.ID,
		"localId":              args.LocalID,
		"required":             args.Required,
		"additionalProperties": args.AdditionalProperties,
	}

	// Handle nested properties recursively
	if len(args.Properties) > 0 {
		nestedProps := make([]map[string]interface{}, 0, len(args.Properties))
		for _, nestedProp := range args.Properties {
			nestedProps = append(nestedProps, nestedProp.ToResourceData())
		}
		propMap["properties"] = nestedProps
	}

	return propMap
}

// FromResourceData populates TrackingPlanPropertyArgs from resource data map
func (args *TrackingPlanPropertyArgs) FromResourceData(propMap map[string]interface{}) {
	args.LocalID = MustString(propMap, "localId")
	args.Required = MustBool(propMap, "required")
	args.ID = String(propMap, "id", "")
	args.AdditionalProperties = Bool(propMap, "additionalProperties", false)

	// Handle nested properties recursively
	nestedProps := NormalizeToSliceMap(propMap, "properties")
	if len(nestedProps) > 0 {
		nestedProperties := make([]*TrackingPlanPropertyArgs, len(nestedProps))
		for nestedIdx, nestedProp := range nestedProps {
			nestedProperty := nestedProp
			nestedArg := &TrackingPlanPropertyArgs{}
			nestedArg.FromResourceData(nestedProperty)
			nestedProperties[nestedIdx] = nestedArg
		}
		args.Properties = nestedProperties
	}
}

func (args *TrackingPlanArgs) FromCatalogTrackingPlan(from *localcatalog.TrackingPlan, urnFromRef func(string) string) error {
	args.Name = from.Name
	args.LocalID = from.LocalID
	args.Description = from.Description

	events := make([]*TrackingPlanEventArgs, 0, len(from.EventProps))
	for _, event := range from.EventProps {
		properties := make([]*TrackingPlanPropertyArgs, 0, len(event.Properties))

		for _, prop := range event.Properties {
			tpProperty := &TrackingPlanPropertyArgs{}

			if err := tpProperty.FromCatalogTrackingPlanEventProperty(
				prop,
				urnFromRef,
			); err != nil {
				return fmt.Errorf("processing property %s: %w", prop.LocalID, err)
			}

			properties = append(properties, tpProperty)
		}
		// sort the properties array by the localID
		utils.SortByLocalID(properties)

		var variants Variants
		for _, localVariant := range event.Variants {
			variant := &Variant{}

			if err := variant.FromLocalCatalogVariant(
				localVariant,
				urnFromRef,
			); err != nil {
				return fmt.Errorf("converting variant for event %s: %w", event.LocalID, err)
			}
			variants = append(variants, *variant)
		}

		// set the identity section to its default value 'properties' if it is not set
		identitySection := event.IdentitySection
		if identitySection == "" {
			identitySection = PropertiesIdentity
		}

		events = append(events, &TrackingPlanEventArgs{
			ID: resources.PropertyRef{
				URN:      urnFromRef(event.Ref),
				Property: "id",
			},
			LocalID:         event.LocalID,
			AllowUnplanned:  event.AllowUnplanned,
			IdentitySection: identitySection,
			Properties:      properties,
			Variants:        variants,
		})
	}

	// sort the events array by the localID
	utils.SortByLocalID(events)
	args.Events = events
	return nil
}

func (args *TrackingPlanArgs) EventByLocalID(id string) *TrackingPlanEventArgs {

	for _, event := range args.Events {
		if event.LocalID == id {
			return event
		}
	}
	return nil
}

func (args *TrackingPlanArgs) FromResourceData(from resources.ResourceData) {

	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.LocalID = MustString(from, "localId")

	var (
		events []interface{}
	)

	// When loading the args from the state []map[string]interface{} is treated as []interface{}
	// but when we have events from catalog being registered as a resource, it is []map[string]interface{}
	events = InterfaceSlice(from, "events", nil)
	if len(events) == 0 {
		eventsMap := MapStringInterfaceSlice(from, "events", nil)
		for _, event := range eventsMap {
			events = append(events, event)
		}
	}

	eventProps := make([]*TrackingPlanEventArgs, len(events))
	for idx, event := range events {
		event := event.(map[string]interface{})

		eventProps[idx] = &TrackingPlanEventArgs{
			ID:              String(event, "id", ""),
			LocalID:         MustString(event, "localId"),
			AllowUnplanned:  MustBool(event, "allowUnplanned"),
			IdentitySection: String(event, "identitySection", ""),
			Properties:      make([]*TrackingPlanPropertyArgs, 0),
		}

		variants := NormalizeToSliceMap(event, "variants")

		var variantsToAdd Variants
		variantsToAdd.FromResourceData(variants)
		eventProps[idx].Variants = variantsToAdd

		// Same situation as the events
		properties := InterfaceSlice(event, "properties", nil)
		if len(properties) == 0 {
			propertiesMap := MapStringInterfaceSlice(event, "properties", nil)
			for _, prop := range propertiesMap {
				properties = append(properties, prop)
			}
		}

		tpProperties := make([]*TrackingPlanPropertyArgs, len(properties))
		for idx, property := range properties {
			property := property.(map[string]interface{})
			tpProperty := &TrackingPlanPropertyArgs{}
			tpProperty.FromResourceData(property)
			tpProperties[idx] = tpProperty
		}
		eventProps[idx].Properties = tpProperties
	}
	args.Events = eventProps
}

func (args *TrackingPlanArgs) ToResourceData() resources.ResourceData {

	events := make([]map[string]interface{}, 0)
	for _, event := range args.Events {

		properties := make([]map[string]interface{}, 0)
		for _, property := range event.Properties {
			properties = append(properties, property.ToResourceData())
		}

		events = append(events, map[string]interface{}{
			"id":              event.ID,
			"localId":         event.LocalID,
			"allowUnplanned":  event.AllowUnplanned,
			"identitySection": event.IdentitySection,
			"properties":      properties,
			"variants":        event.Variants.ToResourceData(),
		})
	}

	return resources.ResourceData{
		"name":        args.Name,
		"description": args.Description,
		"localId":     args.LocalID,
		"events":      events,
	}
}

func (args *TrackingPlanArgs) FromRemoteTrackingPlan(trackingPlan *catalog.TrackingPlanWithIdentifiers, collection *resources.ResourceCollection) error {
	args.Name = trackingPlan.Name
	args.LocalID = trackingPlan.ExternalID
	if trackingPlan.Description != nil {
		args.Description = *trackingPlan.Description
	}

	events := make([]*TrackingPlanEventArgs, 0, len(trackingPlan.Events))
	for _, event := range trackingPlan.Events {
		eventURN, err := collection.GetURNByID(EventResourceType, event.ID)
		if err != nil {
			return fmt.Errorf("getting URN for event %s: %w", event.ID, err)
		}
		eventArgs := &TrackingPlanEventArgs{
			ID: resources.PropertyRef{
				URN:      eventURN,
				Property: "id",
			},
			LocalID:         event.ExternalID,
			AllowUnplanned:  event.AdditionalProperties,
			IdentitySection: event.IdentitySection,
		}

		properties := make([]*TrackingPlanPropertyArgs, 0, len(event.Properties))
		for _, prop := range event.Properties {
			tpProperty := &TrackingPlanPropertyArgs{}
			tpProperty.FromRemoteTrackingPlanProperty(prop, collection, true)
			properties = append(properties, tpProperty)
		}
		// sort the properties array by the localID
		utils.SortByLocalID(properties)
		eventArgs.Properties = properties

		variants := make([]Variant, 0, len(event.Variants))
		for _, remoteVariant := range event.Variants {
			variant := Variant{}
			variant.FromRemoteVariant(remoteVariant, collection.GetURNByID, true)
			variants = append(variants, variant)
		}
		eventArgs.Variants = variants
		events = append(events, eventArgs)
	}
	// sort the events array by the localID
	utils.SortByLocalID(events)
	args.Events = events

	return nil
}
