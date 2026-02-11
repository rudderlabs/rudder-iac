package localcatalog

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/samber/lo"
)

var (
	log = logger.New("localcatalog")
)

const (
	KindProperties      = "properties"
	KindEvents          = "events"
	KindCategories      = "categories"
	KindTrackingPlans   = "tp"
	KindTrackingPlansV1 = "tracking-plan"
	KindCustomTypes     = "custom-types"
)

// entity group is logical grouping of entities defined
// as metadata->name in respective yaml file
type EntityGroup string

type WorkspaceRemoteIDMapping struct {
	WorkspaceID string
	RemoteID    string
}

// Create a reverse lookup based on the groupName and identifier per entity
type DataCatalog struct {
	Properties     []PropertyV1                         `json:"properties"`
	Events         []EventV1                            `json:"events"`
	TrackingPlans  []*TrackingPlanV1                    `json:"trackingPlans"`
	CustomTypes    []CustomTypeV1                       `json:"customTypes"`
	Categories     []CategoryV1                         `json:"categories"`
	ImportMetadata map[string]*WorkspaceRemoteIDMapping `json:"importMetadata"`
	ReferenceMap   map[string]string                    `json:"-"` // Maps URN references to original path-based references
}

func (dc *DataCatalog) Property(id string) *PropertyV1 {
	for i := range dc.Properties {
		if dc.Properties[i].LocalID == id {
			return &dc.Properties[i]
		}
	}
	return nil
}

func (dc *DataCatalog) Event(id string) *EventV1 {
	for i := range dc.Events {
		if dc.Events[i].LocalID == id {
			return &dc.Events[i]
		}
	}
	return nil
}

// Category returns a category by ID
func (dc *DataCatalog) Category(id string) *CategoryV1 {
	for i := range dc.Categories {
		if dc.Categories[i].LocalID == id {
			return &dc.Categories[i]
		}
	}
	return nil
}

// CustomType returns a custom type by ID
func (dc *DataCatalog) CustomType(id string) *CustomTypeV1 {
	for i := range dc.CustomTypes {
		if dc.CustomTypes[i].LocalID == id {
			return &dc.CustomTypes[i]
		}
	}
	return nil
}

func (dc *DataCatalog) TPEventRule(tpID, ruleID string) *TPRuleV1 {
	var tp *TrackingPlanV1
	for i := range dc.TrackingPlans {
		if dc.TrackingPlans[i].LocalID == tpID {
			tp = dc.TrackingPlans[i]
			break
		}
	}
	if tp == nil {
		return nil
	}

	for _, rule := range tp.Rules {
		if rule.LocalID == ruleID && rule.Type == "event_rule" {
			return rule
		}
	}

	return nil
}

func (dc *DataCatalog) TPEventRules(tpID string) ([]*TPRuleV1, bool) {
	var tp *TrackingPlanV1
	for i := range dc.TrackingPlans {
		if dc.TrackingPlans[i].LocalID == tpID {
			tp = dc.TrackingPlans[i]
			break
		}
	}
	if tp == nil {
		return nil, false
	}

	var toReturn []*TPRuleV1
	for _, rule := range tp.Rules {
		if rule.Type != "event_rule" {
			continue
		}
		toReturn = append(toReturn, rule)
	}

	return toReturn, true
}

func New() *DataCatalog {
	return &DataCatalog{
		Properties:     []PropertyV1{},
		Events:         []EventV1{},
		TrackingPlans:  []*TrackingPlanV1{},
		CustomTypes:    []CustomTypeV1{},
		Categories:     []CategoryV1{},
		ImportMetadata: map[string]*WorkspaceRemoteIDMapping{},
		ReferenceMap:   map[string]string{},
	}
}

// convertPathToURN converts a path-based reference to URN format
// Example: #/custom-types/common/Address -> #custom-type:Address
func convertPathToURN(pathRef string) (string, error) {
	pathRef = strings.TrimPrefix(pathRef, "#/")
	parts := strings.Split(pathRef, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid path reference: %s", pathRef)
	}

	entityType := parts[0]
	localId := parts[2]
	URN := ""
	switch entityType {
	case KindProperties:
		URN = resources.URN(localId, "property")
	case KindEvents:
		URN = resources.URN(localId, "event")
	case KindCustomTypes:
		URN = resources.URN(localId, "custom-type")
	case KindCategories:
		URN = resources.URN(localId, "category")
	default:
		return "", fmt.Errorf("invalid entity type: %s", entityType)
	}

	return fmt.Sprintf("#%s", URN), nil
}

// transformReferencesInSpec recursively walks the spec map and transforms
// all string values starting with #/ to URN format, tracking the mappings
func (dc *DataCatalog) transformReferencesInSpec(spec map[string]any) error {
	for key, value := range spec {
		switch v := value.(type) {
		case string:
			if strings.HasPrefix(v, "#/") {
				urnRef, err := convertPathToURN(v)
				if err != nil {
					return err
				}
				dc.ReferenceMap[urnRef] = v
				spec[key] = urnRef
			}
		case map[string]any:
			if err := dc.transformReferencesInSpec(v); err != nil {
				return err
			}
		case []any:
			for i, item := range v {
				switch itemVal := item.(type) {
				case string:
					if strings.HasPrefix(itemVal, "#/") {
						urnRef, err := convertPathToURN(itemVal)
						if err != nil {
							return err
						}
						dc.ReferenceMap[urnRef] = itemVal
						v[i] = urnRef
					}
				case map[string]any:
					if err := dc.transformReferencesInSpec(itemVal); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (dc *DataCatalog) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	var (
		parsedSpec   specs.ParsedSpec
		idArray      []any
		basePath     string
		resourceType string
	)

	switch s.Kind {
	case KindProperties:
		properties, ok := s.Spec["properties"].([]any)
		if !ok {
			return nil, fmt.Errorf("kind: %s, properties not found in spec", s.Kind)
		}
		idArray = properties
		basePath = "/spec/properties"
		resourceType = "property"

	case KindEvents:
		events, ok := s.Spec["events"].([]any)
		if !ok {
			return nil, fmt.Errorf("kind: %s, events not found in spec", s.Kind)
		}
		idArray = events
		basePath = "/spec/events"
		resourceType = "event"

	case KindTrackingPlans, KindTrackingPlansV1:
		tpID, ok := s.Spec["id"].(string)
		if !ok {
			return nil, fmt.Errorf("kind: %s, id not found in tracking plan spec", s.Kind)
		}
		parsedSpec.URNs = append(parsedSpec.URNs, resources.URN(tpID, "tracking-plan"))
		parsedSpec.LocalIDs = append(parsedSpec.LocalIDs, specs.LocalID{
			ID:              tpID,
			JSONPointerPath: "/spec/id",
		})
		return &parsedSpec, nil

	case KindCustomTypes:
		customTypes, ok := s.Spec["types"].([]any)
		if !ok {
			return nil, fmt.Errorf("kind: %s, custom types not found in spec", s.Kind)
		}
		idArray = customTypes
		basePath = "/spec/types"
		resourceType = "custom-type"

	case KindCategories:
		categories, ok := s.Spec["categories"].([]any)
		if !ok {
			return nil, fmt.Errorf("kind: %s, categories not found in spec", s.Kind)
		}
		idArray = categories
		basePath = "/spec/categories"
		resourceType = "category"
	}

	for i, item := range idArray {
		idMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("entity is not a map[string]any: %s", s.Kind)
		}
		id, ok := idMap["id"].(string)
		if !ok {
			return nil, fmt.Errorf("id not found in entity: %s", s.Kind)
		}
		parsedSpec.URNs = append(parsedSpec.URNs, resources.URN(id, resourceType))
		parsedSpec.LocalIDs = append(parsedSpec.LocalIDs, specs.LocalID{
			ID:              id,
			JSONPointerPath: fmt.Sprintf("%s/%d/id", basePath, i),
		})
	}

	parsedSpec.LegacyResourceType = resourceType
	return &parsedSpec, nil
}

func (dc *DataCatalog) LoadLegacySpec(path string, s *specs.Spec) error {
	// Transform path-based references to URN format
	err := dc.transformReferencesInSpec(s.Spec)
	if err != nil {
		return fmt.Errorf("processing references in spec: %w", err)
	}

	if err := extractEntities(s, dc); err != nil {
		return fmt.Errorf("extracting data catalog entity from file: %s : %w", path, err)
	}

	if err := addImportMetadata(s, dc); err != nil {
		return fmt.Errorf("adding import metadata: %w", err)
	}

	return nil
}

func (dc *DataCatalog) LoadSpec(path string, s *specs.Spec) error {
	if err := extractEntititesV1(s, dc); err != nil {
		return fmt.Errorf("extracting data catalog entity from file: %s : %w", path, err)
	}

	if err := addImportMetadata(s, dc); err != nil {
		return fmt.Errorf("adding import metadata: %w", err)
	}

	return nil
}

func (dc *DataCatalog) MigrateSpec(s *specs.Spec) (*specs.Spec, error) {
	var resourceSpec any
	switch s.Kind {
	case KindProperties:
		properties, err := ExtractProperties(s)
		if err != nil {
			return nil, fmt.Errorf("extracting properties: %w", err)
		}
		resourceSpec = PropertySpecV1{
			Properties: properties,
		}
	case KindCustomTypes:
		customTypes, err := ExtractCustomTypes(s)
		if err != nil {
			return nil, fmt.Errorf("extracting custom types: %w", err)
		}
		resourceSpec = CustomTypeSpecV1{
			Types: customTypes,
		}
	case KindEvents:
		events, err := ExtractEvents(s)
		if err != nil {
			return nil, fmt.Errorf("extracting events: %w", err)
		}
		resourceSpec = EventSpecV1{
			Events: events,
		}
	case KindCategories:
		categories, err := ExtractCategories(s)
		if err != nil {
			return nil, fmt.Errorf("extracting categories: %w", err)
		}
		resourceSpec = CategorySpecV1{
			Categories: categories,
		}
	case KindTrackingPlans:
		trackingPlan, err := ExtractTrackingPlan(s)
		if err != nil {
			return nil, fmt.Errorf("extracting tracking plans: %w", err)
		}
		resourceSpec = trackingPlan
		// change kind for tracking plans from "tp" to "tracking-plan"
		s.Kind = KindTrackingPlansV1
	default:
		return nil, fmt.Errorf("unknown kind: %s", s.Kind)
	}

	jsonByt, err := json.Marshal(resourceSpec)
	if err != nil {
		return nil, fmt.Errorf("marshalling properties: %w", err)
	}
	if err = json.Unmarshal(jsonByt, &s.Spec); err != nil {
		return nil, fmt.Errorf("unmarshalling properties: %w", err)
	}
	return s, nil
}

func addImportMetadata(s *specs.Spec, dc *DataCatalog) error {
	metadata, err := s.CommonMetadata()
	if err != nil {
		return err
	}

	if metadata.Import != nil {
		// Map spec kind to resource type
		var resourceType string
		switch s.Kind {
		case KindProperties:
			resourceType = "property"
		case KindEvents:
			resourceType = "event"
		case KindCustomTypes:
			resourceType = "custom-type"
		case KindTrackingPlans, KindTrackingPlansV1:
			resourceType = "tracking-plan"
		case KindCategories:
			resourceType = "category"
		default:
			return fmt.Errorf("unknown kind: %s", s.Kind)
		}

		lo.ForEach(metadata.Import.Workspaces, func(workspace specs.WorkspaceImportMetadata, _ int) {
			// For each resource within the workspace, load the import metadata
			// which will be used during the creation of resourceGraph
			lo.ForEach(workspace.Resources, func(resource specs.ImportIds, _ int) {
				// Support both URN field (new) and LocalID field (legacy)
				var urn string
				if resource.URN != "" {
					urn = resource.URN
				} else {
					urn = resources.URN(resource.LocalID, resourceType)
				}
				dc.ImportMetadata[urn] = &WorkspaceRemoteIDMapping{
					WorkspaceID: workspace.WorkspaceID,
					RemoteID:    resource.RemoteID,
				}
			})
		})
	}

	return nil
}

// extractEntities parses the entity from file bytes
// and updates the datacatalog struct with it.
func extractEntities(s *specs.Spec, dc *DataCatalog) error {
	// TODO: properly handle metadata - ensuring schema and types
	switch s.Kind {
	case KindProperties:
		properties, err := ExtractProperties(s)
		if err != nil {
			return fmt.Errorf("extracting properties: %w", err)
		}
		dc.Properties = append(dc.Properties, properties...)

	case KindEvents:
		events, err := ExtractEvents(s)
		if err != nil {
			return fmt.Errorf("extracting property entity: %w", err)
		}
		dc.Events = append(dc.Events, events...)

	case KindCategories:
		categories, err := ExtractCategories(s)
		if err != nil {
			return fmt.Errorf("extracting categories: %w", err)
		}
		dc.Categories = append(dc.Categories, categories...)

	case KindTrackingPlans:
		trackingPlan, err := ExtractTrackingPlan(s)
		if err != nil {
			return fmt.Errorf("extracting tracking plan: %w", err)
		}

		// Check for duplicates
		for i := range dc.TrackingPlans {
			if dc.TrackingPlans[i].LocalID == trackingPlan.LocalID {
				return fmt.Errorf("duplicate tracking plan with id '%s' found", trackingPlan.LocalID)
			}
		}
		dc.TrackingPlans = append(dc.TrackingPlans, &trackingPlan)

	case KindCustomTypes:
		customTypes, err := ExtractCustomTypes(s)
		if err != nil {
			return fmt.Errorf("extracting custom types: %w", err)
		}
		dc.CustomTypes = append(dc.CustomTypes, customTypes...)

	default:
		return fmt.Errorf("unknown kind: %s", s.Kind)
	}

	return nil
}
