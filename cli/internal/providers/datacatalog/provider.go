package datacatalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	dcdocs "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	catalogrules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	categoryRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/category"
	customtypeRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/customtype"
	eventRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/event"
	propertyRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/property"
	trackingplanRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/trackingplan"
	pstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	rstate "github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

var log = logger.New("datacatalogprovider")

const importDir = "data-catalog"

type Provider struct {
	provider.EmptyProvider
	concurrency   int
	client        catalog.DataCatalog
	dc            *localcatalog.DataCatalog
	providerStore map[string]entityProvider
	workspaceID   string
	// Offline validation can suppress imports when the project declares more
	// than one workspace, even if an individual spec has a unique inline ID.
	validationWorkspaceID string
	validationScoped      bool
}

func New(client catalog.DataCatalog) *Provider {
	return &Provider{
		concurrency: config.GetConfig().Concurrency.CatalogProvider,
		client:      client,
		dc:          localcatalog.New(),
		providerStore: map[string]entityProvider{
			types.PropertyResourceType:     NewPropertyProvider(client, importDir),
			types.EventResourceType:        NewEventProvider(client, importDir),
			types.CustomTypeResourceType:   NewCustomTypeProvider(client, importDir),
			types.TrackingPlanResourceType: NewTrackingPlanProvider(client, importDir),
			types.CategoryResourceType:     NewCategoryProvider(client, importDir),
		},
	}
}

func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	return p.dc.ParseSpec(path, s)
}

func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	return p.dc.LoadSpec(path, s)
}

func (p *Provider) LoadLegacySpec(path string, s *specs.Spec) error {
	return p.dc.LoadLegacySpec(path, s)
}

// SetWorkspaceID scopes inline and central import identities to the active
// workspace. Project.Load calls this before loading resource specs. An unresolved
// workspace cannot grandfather a legacy tracking-plan name.
func (p *Provider) SetWorkspaceID(id string) {
	p.workspaceID = id
	p.validationScoped = false
	p.dc.SetWorkspaceID(id)
	if tp, ok := p.providerStore[types.TrackingPlanResourceType].(*TrackingPlanEntityProvider); ok {
		tp.TrackingPlanProvider.workspaceID = id
	}
}

// SetValidationWorkspaceID narrows graph metadata for offline validation only.
// It never changes the active workspace used by CheckPlan or writes.
func (p *Provider) SetValidationWorkspaceID(id string) {
	p.validationWorkspaceID = id
	p.validationScoped = true
}

// LoadImportManifest delegates the active workspace's manifest to the local
// catalog so resources whose URNs match a manifest entry get ImportMetadata
// attached at ResourceGraph() time, mirroring the inline metadata.import path.
func (p *Provider) LoadImportManifest(m *specs.WorkspaceImportMetadata) error {
	return p.dc.LoadImportManifest(m)
}

func (p *Provider) MigrateSpec(s *specs.Spec) (*specs.Spec, error) {
	return p.dc.MigrateSpec(s)
}

func (p *Provider) SupportedKinds() []string {
	return []string{
		localcatalog.KindProperties,
		localcatalog.KindEvents,
		localcatalog.KindTrackingPlans,
		localcatalog.KindTrackingPlansV1,
		localcatalog.KindCustomTypes,
		localcatalog.KindCategories,
	}
}

func (p *Provider) SupportedMatchPatterns() []rules.MatchPattern {
	var patterns []rules.MatchPattern
	// properties, events, categories, custom-types support both legacy and v1 versions
	for _, kind := range []string{
		localcatalog.KindProperties,
		localcatalog.KindEvents,
		localcatalog.KindCategories,
		localcatalog.KindCustomTypes,
	} {
		patterns = append(patterns, prules.LegacyVersionPatterns(kind)...)
		patterns = append(patterns, prules.V1VersionPatterns(kind)...)
	}
	// tp (KindTrackingPlans) is legacy-only
	patterns = append(patterns, prules.LegacyVersionPatterns(localcatalog.KindTrackingPlans)...)
	// tracking-plan (KindTrackingPlansV1) is v1-only
	patterns = append(patterns, prules.V1VersionPatterns(localcatalog.KindTrackingPlansV1)...)
	return patterns
}

func (p *Provider) SupportedTypes() []string {
	return []string{
		types.PropertyResourceType,
		types.EventResourceType,
		types.TrackingPlanResourceType,
		types.CustomTypeResourceType,
		types.CategoryResourceType,
	}
}

func (p *Provider) GetLocalCatalog() *localcatalog.DataCatalog {
	return p.dc
}

func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	if err := inflateRefs(p.dc); err != nil {
		return nil, fmt.Errorf("inflating refs: %w", err)
	}

	graph, err := p.createResourceGraph()
	if err != nil {
		return nil, err
	}
	// Capture only declarations actually attached to the active workspace.
	if tp, ok := p.providerStore[types.TrackingPlanResourceType].(*TrackingPlanEntityProvider); ok {
		ids := make(map[string]string)
		for _, resource := range graph.ResourcesByType(types.TrackingPlanResourceType) {
			if meta := resource.ImportMetadata(); meta != nil && p.workspaceID != "" && meta.WorkspaceId == p.workspaceID && meta.RemoteId != "" {
				ids[resource.ID()] = meta.RemoteId
			}
		}
		tp.TrackingPlanProvider.importedIDs = ids
	}
	return graph, nil
}

// CheckPlan verifies legacy tracking plan names against the declared import
// identity and the live catalog before the first apply mutation. Managed
// updates additionally require unchanged name and description in prior state.
func (p *Provider) CheckPlan(ctx context.Context, plan *planner.Plan, st *rstate.State) error {
	if plan == nil {
		return nil
	}
	for _, op := range plan.Operations {
		if op == nil || op.Resource == nil || op.Resource.Type() != types.TrackingPlanResourceType || op.Type == planner.Delete {
			continue
		}
		name, _ := op.Resource.Data()["name"].(string)
		if catalogrules.ValidTrackingPlanName(name) {
			continue
		}
		if !catalogrules.LegacyTrackingPlanName(name) {
			return fmt.Errorf("tracking plan %s: invalid display_name %q (the catalog API rejects it)", op.Resource.URN(), name)
		}
		meta := op.Resource.ImportMetadata()
		declared := p.dc.ImportMetadata[op.Resource.URN()]
		if (op.Type != planner.Import && op.Type != planner.Update) || meta == nil || meta.RemoteId == "" || meta.WorkspaceId == "" || p.workspaceID == "" || meta.WorkspaceId != p.workspaceID || declared == nil || declared.WorkspaceID != meta.WorkspaceId || declared.RemoteID != meta.RemoteId {
			return fmt.Errorf("tracking plan %s: display_name %q contains '&' and requires a declared import for this workspace", op.Resource.URN(), name)
		}
		if op.Type == planner.Update {
			var previous *rstate.ResourceState
			if st != nil {
				previous = st.GetResource(op.Resource.URN())
			}
			if previous == nil || previous.ID != op.Resource.ID() || previous.Type != types.TrackingPlanResourceType {
				return fmt.Errorf("tracking plan %s: legacy display_name %q requires prior managed state", op.Resource.URN(), name)
			}
			prior := previous.Data()
			priorArgs, ok := prior["trackingPlanArgs"].(map[string]any)
			if !ok {
				return fmt.Errorf("tracking plan %s: legacy display_name %q requires prior managed tracking plan state", op.Resource.URN(), name)
			}
			prevID, _ := prior["id"].(string)
			prevWorkspace, _ := prior["workspaceId"].(string)
			prevName, _ := prior["name"].(string)
			prevDescription, _ := prior["description"].(string)
			prevArgsName, _ := priorArgs["name"].(string)
			prevArgsDescription, _ := priorArgs["description"].(string)
			description, _ := op.Resource.Data()["description"].(string)
			if prevID != meta.RemoteId || prevWorkspace != p.workspaceID || prevName != name || prevArgsName != name || prevArgsDescription != description || prevDescription != description {
				return fmt.Errorf("tracking plan %s: legacy display_name %q cannot be renamed or have its description changed", op.Resource.URN(), name)
			}
		}
		tp, ok := p.providerStore[types.TrackingPlanResourceType].(*TrackingPlanEntityProvider)
		if !ok {
			return fmt.Errorf("tracking plan %s: tracking plan provider is missing", op.Resource.URN())
		}
		if err := tp.TrackingPlanProvider.verifyLegacyIdentity(ctx, op.Resource.ID(), meta.RemoteId, name, op.Type == planner.Update, nil); err != nil {
			return fmt.Errorf("verifying tracking plan %s: %w", op.Resource.URN(), err)
		}
	}
	return nil
}

func (p *Provider) List(ctx context.Context, resourceType string, filters lister.Filters) ([]resources.ResourceData, error) {
	switch resourceType {
	case types.TrackingPlanResourceType:
		return p.listTrackingPlans(ctx)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func (p *Provider) listTrackingPlans(ctx context.Context) ([]resources.ResourceData, error) {
	trackingPlans, err := p.client.GetTrackingPlansWithIdentifiers(ctx, catalog.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tracking plans: %w", err)
	}

	var result []resources.ResourceData
	for _, tp := range trackingPlans {
		// Handle nil description
		description := ""
		if tp.Description != nil {
			description = *tp.Description
		}

		// Include all relevant fields for Lister display
		// "id" and "name" are shown in columns, other fields in details panel
		resourceData := resources.ResourceData{
			"name":        tp.Name,
			"id":          tp.ID,
			"version":     tp.Version,
			"description": description,
			"createdAt":   tp.CreatedAt.String(),
			"updatedAt":   tp.UpdatedAt.String(),
		}
		result = append(result, resourceData)
	}

	return result, nil
}

func (p *Provider) createResourceGraph() (*resources.Graph, error) {
	catalog := p.dc
	graph := resources.NewGraph()

	// First, pre-calculate all URNs to use for references
	propIDToURN := make(map[string]string)
	for _, prop := range catalog.Properties {
		propIDToURN[prop.LocalID] = resources.URN(prop.LocalID, types.PropertyResourceType)
	}

	eventIDToURN := make(map[string]string)
	for _, event := range catalog.Events {
		eventIDToURN[event.LocalID] = resources.URN(event.LocalID, types.EventResourceType)
	}

	customTypeIDToURN := make(map[string]string)
	for _, customType := range catalog.CustomTypes {
		customTypeIDToURN[customType.LocalID] = resources.URN(customType.LocalID, types.CustomTypeResourceType)
	}

	categoryIDToURN := make(map[string]string)
	for _, category := range catalog.Categories {
		categoryIDToURN[category.LocalID] = resources.URN(category.LocalID, types.CategoryResourceType)
	}

	getResourceImportMetadata := func(id, resourceType string) resources.ResourceOpts {
		metadata, ok := catalog.ImportMetadata[resources.URN(id, resourceType)]
		if !ok || metadata == nil || p.workspaceID != "" && metadata.WorkspaceID != p.workspaceID ||
			p.validationScoped && (p.validationWorkspaceID == "" || metadata.WorkspaceID != p.validationWorkspaceID) {
			return nil
		}
		return resources.WithResourceImportMetadata(metadata.RemoteID, metadata.WorkspaceID)
	}

	// Helper function to get URN from reference
	getURNFromRef := func(ref string) string {
		return strings.TrimPrefix(ref, "#")
	}

	// Add properties to the graph
	for _, prop := range catalog.Properties {
		log.Debug("adding property to graph", "id", prop.LocalID)

		args := &pstate.PropertyArgs{}
		if err := args.FromCatalogPropertyType(prop, getURNFromRef); err != nil {
			return nil, fmt.Errorf("creating property args from catalog property: %s, err:%w", prop.LocalID, err)
		}

		resource := resources.NewResource(
			prop.LocalID,
			types.PropertyResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(prop.LocalID, types.PropertyResourceType),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindProperties,
				prop.LocalID,
			)),
		)
		graph.AddResource(resource)

		propIDToURN[prop.LocalID] = resource.URN()
	}

	// Add events to the graph
	for _, event := range catalog.Events {
		log.Debug("adding event to graph", "event", event.LocalID)

		args := pstate.EventArgs{}
		args.FromCatalogEvent(&event, getURNFromRef)
		resource := resources.NewResource(
			event.LocalID,
			types.EventResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(event.LocalID, types.EventResourceType),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindEvents,
				event.LocalID,
			)),
		)
		graph.AddResource(resource)

		eventIDToURN[event.LocalID] = resource.URN()
	}

	// Add custom types to the graph with dependencies on properties or other custom types
	for _, customType := range catalog.CustomTypes {
		log.Debug("adding custom type to graph", "id", customType.LocalID)

		// Add CustomTypeArgs
		args := pstate.CustomTypeArgs{}
		if err := args.FromCatalogCustomType(&customType, getURNFromRef); err != nil {
			return nil, fmt.Errorf("creating custom type args from catalog custom type: %s, err:%w", customType.LocalID, err)
		}
		resource := resources.NewResource(
			customType.LocalID,
			types.CustomTypeResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(customType.LocalID, types.CustomTypeResourceType),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindCustomTypes,
				customType.LocalID,
			)),
		)
		graph.AddResource(resource)
	}

	// Add categories to the graph
	for _, category := range catalog.Categories {
		log.Debug("adding category to graph", "id", category.LocalID)

		args := pstate.CategoryArgs{}
		args.FromCatalogCategory(&category)
		resource := resources.NewResource(
			category.LocalID,
			types.CategoryResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(category.LocalID, types.CategoryResourceType),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindCategories,
				category.LocalID,
			)),
		)
		graph.AddResource(resource)
	}

	// Add tracking plans to the graph
	for _, tp := range catalog.TrackingPlans {
		log.Debug("adding tracking plan to graph", "tp", tp.LocalID)

		args := pstate.TrackingPlanArgs{}
		if err := args.FromCatalogTrackingPlan(tp, getURNFromRef); err != nil {
			return nil, fmt.Errorf("creating tracking plan args: %w", err)
		}

		resource := resources.NewResource(
			tp.LocalID,
			types.TrackingPlanResourceType,
			args.ToResourceData(),
			make([]string, 0),
			getResourceImportMetadata(tp.LocalID, types.TrackingPlanResourceType),
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s",
				localcatalog.KindTrackingPlans,
				tp.LocalID,
			)),
			resources.WithAdditionalMetadata(map[string]any{
				"ruleIdToEventId": lo.SliceToMap(tp.Rules, func(rule *localcatalog.TPRuleV1) (string, string) {
					return rule.LocalID, strings.TrimPrefix(rule.Event, "#event:")
				}),
			}),
		)
		graph.AddResource(resource)
		graph.AddDependencies(resource.URN(), getDependencies(tp, propIDToURN, eventIDToURN))
	}

	return graph, nil
}

// getDependencies simply fetch the dependencies on the trackingplan in form of the URN's
// of the properties and events that are used in the tracking plan
func getDependencies(tp *localcatalog.TrackingPlanV1, propIDToURN, eventIDToURN map[string]string) []string {
	dependencies := make([]string, 0)

	for _, event := range tp.EventProps {
		if urn, ok := eventIDToURN[event.LocalID]; ok {
			dependencies = append(dependencies, urn)
		}

		for _, prop := range event.Properties {
			if urn, ok := propIDToURN[prop.LocalID]; ok {
				dependencies = append(dependencies, urn)
			}
		}
	}

	return dependencies
}

func inflateRefs(catalog *localcatalog.DataCatalog) error {
	log.Debug("inflating all the references in the catalog")

	for _, tp := range catalog.TrackingPlans {
		if err := tp.ExpandRefs(catalog); err != nil {
			return fmt.Errorf("expanding refs on tp: %s err: %w", tp.LocalID, err)
		}
	}
	return nil
}

func (p *Provider) SyntacticRules() []rules.Rule {
	eventRuleIncludesEnabled := config.GetConfig().ExperimentalFlags.EventRuleIncludes

	syntactic := []rules.Rule{
		propertyRules.NewPropertySpecSyntaxValidRule(),
		propertyRules.NewPropertyConfigValidRule(),
		customtypeRules.NewCustomTypeSpecSyntaxValidRule(),
		customtypeRules.NewCustomTypeConfigValidRule(),
		eventRules.NewEventSpecSyntaxValidRule(),
		categoryRules.NewCategorySpecSyntaxValidRule(),
		trackingplanRules.NewTrackingPlanSpecSyntaxValidRule(eventRuleIncludesEnabled),
	}

	return syntactic
}

// RuleDocEntries returns the authored documentation fragments embedded with
// the datacatalog provider, joined to registered rules by the docs generator.
func (p *Provider) RuleDocEntries() []docs.RuleDocEntry {
	entries, _ := docs.LoadRuleDocEntries(dcdocs.FragmentsFS, ".")
	return entries
}

func (p *Provider) SemanticRules() []rules.Rule {
	eventRuleIncludesEnabled := config.GetConfig().ExperimentalFlags.EventRuleIncludes

	return []rules.Rule{
		propertyRules.NewPropertySemanticValidRule(),
		eventRules.NewEventSemanticValidRule(),
		categoryRules.NewCategorySemanticValidRule(),
		customtypeRules.NewCustomTypeSemanticValidRule(),
		trackingplanRules.NewTrackingPlanSemanticValidRule(eventRuleIncludesEnabled),
	}
}
