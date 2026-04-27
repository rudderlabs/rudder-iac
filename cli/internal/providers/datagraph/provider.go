package datagraph

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	dgRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
)

// Provider wraps the base provider to provide a concrete type for dependency injection
// The provider owns all spec parsing logic for data-graph specs
type Provider struct {
	*provider.BaseProvider
	client              dgClient.DataGraphClient
	dataGraphHandler    *datagraph.DataGraphHandler
	modelHandler        *model.ModelHandler
	relationshipHandler *relationship.RelationshipHandler
}

// NewProvider creates a new data graph provider instance.
// The accountGetter is used internally to build the account name resolver.
func NewProvider(client dgClient.DataGraphClient, accountGetter datagraph.AccountGetter) *Provider {
	var accountResolver datagraph.AccountNameResolver
	if accountGetter != nil {
		accountResolver = datagraph.NewAccountNameResolver(accountGetter)
	}
	dgHandler := datagraph.NewHandler(client, accountResolver)
	modelHandler := model.NewHandler(client)
	relationshipHandler := relationship.NewHandler(client)

	handlers := []provider.Handler{
		dgHandler,
		modelHandler,
		relationshipHandler,
	}

	return &Provider{
		BaseProvider:        provider.NewBaseProvider(handlers),
		client:              client,
		dataGraphHandler:    dgHandler,
		modelHandler:        modelHandler,
		relationshipHandler: relationshipHandler,
	}
}

// Client returns the underlying DataGraphClient for direct API access
func (p *Provider) Client() dgClient.DataGraphClient {
	return p.client
}

// ParseSpec overrides BaseProvider.ParseSpec to extract IDs from data-graph specs with inline models
func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	// For data-graph specs, extract both data graph ID and inline model IDs
	if s.Kind == "data-graph" {
		return p.parseDataGraphWithInlineModels(s)
	}

	// For other specs, use base implementation
	return p.BaseProvider.ParseSpec(path, s)
}

// LoadSpec overrides BaseProvider.LoadSpec to handle inline models in data-graph specs
func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	// For data-graph specs, check for inline models
	if s.Kind == "data-graph" {
		return p.loadDataGraphWithInlineModels(s)
	}

	// For other specs, use base implementation
	return p.BaseProvider.LoadSpec(path, s)
}

// SupportedMatchPatterns declares the (kind, version) pairs this provider fully handles.
// Data-graph specs only support the V1 version; no legacy version support.
func (p *Provider) SupportedMatchPatterns() []rules.MatchPattern {
	return prules.V1VersionPatterns("data-graph")
}

func (p *Provider) SyntacticRules() []rules.Rule {
	return []rules.Rule{
		dgRules.NewDataGraphSpecSyntaxValidRule(),
	}
}

func (p *Provider) SemanticRules() []rules.Rule {
	return []rules.Rule{
		dgRules.NewRelationshipCardinalityValidRule(),
		dgRules.NewRelationshipRefsValidRule(),
		dgRules.NewRelationshipUniquePairRule(),
		dgRules.NewUniqueNamesValidRule(),
	}
}

func (p *Provider) parseDataGraphWithInlineModels(s *specs.Spec) (*specs.ParsedSpec, error) {
	// Parse the data-graph spec to extract IDs
	var dgSpec dgModel.DataGraphSpec
	if err := mapstructure.Decode(s.Spec, &dgSpec); err != nil {
		return nil, fmt.Errorf("decoding data graph spec: %w", err)
	}

	urnEntries := []specs.URNEntry{{
		URN:             resources.URN(dgSpec.ID, datagraph.HandlerMetadata.ResourceType),
		JSONPointerPath: "/spec/id",
	}}

	for i, modelSpec := range dgSpec.Models {
		if modelSpec.ID != "" {
			urnEntries = append(urnEntries, specs.URNEntry{
				URN:             resources.URN(modelSpec.ID, model.HandlerMetadata.ResourceType),
				JSONPointerPath: fmt.Sprintf("/spec/models/%d/id", i),
			})
		}
		for j, relSpec := range modelSpec.Relationships {
			if relSpec.ID != "" {
				urnEntries = append(urnEntries, specs.URNEntry{
					URN:             resources.URN(relSpec.ID, relationship.HandlerMetadata.ResourceType),
					JSONPointerPath: fmt.Sprintf("/spec/models/%d/relationships/%d/id", i, j),
				})
			}
		}
	}

	return &specs.ParsedSpec{URNs: urnEntries}, nil
}

func (p *Provider) loadDataGraphWithInlineModels(s *specs.Spec) error {
	// Parse the data-graph spec
	var dgSpec dgModel.DataGraphSpec
	if err := mapstructure.Decode(s.Spec, &dgSpec); err != nil {
		return fmt.Errorf("decoding data graph spec: %w", err)
	}

	// Extract the data graph resource
	dgResource, err := p.extractDataGraphResource(&dgSpec)
	if err != nil {
		return fmt.Errorf("extracting data graph resource: %w", err)
	}

	// Register the data graph resource with its handler
	if err := p.dataGraphHandler.AddResource(dgSpec.ID, dgResource); err != nil {
		return fmt.Errorf("adding data graph resource: %w", err)
	}

	// Process inline models if any
	for _, modelSpec := range dgSpec.Models {
		if err := p.processInlineModel(dgSpec.ID, modelSpec); err != nil {
			return fmt.Errorf("processing inline model %s: %w", modelSpec.ID, err)
		}
	}

	// Load import metadata into all handlers so the apply cycle
	// recognizes these resources as imports (not creates)
	commonMetadata, err := s.CommonMetadata()
	if err != nil {
		return fmt.Errorf("getting common metadata: %w", err)
	}
	if commonMetadata.Import != nil {
		if err := p.dataGraphHandler.LoadImportMetadata(commonMetadata.Import); err != nil {
			return fmt.Errorf("loading import metadata: %w", err)
		}
		if err := p.modelHandler.LoadImportMetadata(commonMetadata.Import); err != nil {
			return fmt.Errorf("loading import metadata: %w", err)
		}
		if err := p.relationshipHandler.LoadImportMetadata(commonMetadata.Import); err != nil {
			return fmt.Errorf("loading import metadata: %w", err)
		}
	}

	return nil
}

func (p *Provider) processInlineModel(dataGraphID string, modelSpec dgModel.ModelSpec) error {
	// Extract model resource from spec
	resource, err := p.extractModelResource(dataGraphID, &modelSpec)
	if err != nil {
		return fmt.Errorf("extracting model resource: %w", err)
	}

	// Register resource with handler
	if err := p.modelHandler.AddResource(modelSpec.ID, resource); err != nil {
		return fmt.Errorf("adding model resource: %w", err)
	}

	// Process inline relationships if any
	for _, relSpec := range modelSpec.Relationships {
		if err := p.processInlineRelationship(dataGraphID, modelSpec.ID, &relSpec); err != nil {
			return fmt.Errorf("processing inline relationship %s: %w", relSpec.ID, err)
		}
	}

	return nil
}


// extractModelResource creates a ModelResource from an inline model spec
func (p *Provider) extractModelResource(dataGraphID string, spec *dgModel.ModelSpec) (*dgModel.ModelResource, error) {
	// Create URN for the parent data graph
	dataGraphURN := resources.URN(dataGraphID, datagraph.HandlerMetadata.ResourceType)

	// Create PropertyRef to the data graph
	dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

	resource := &dgModel.ModelResource{
		ID:           spec.ID,
		DisplayName:  spec.DisplayName,
		Type:         spec.Type,
		Table:        spec.Table,
		Description:  spec.Description,
		DataGraphRef: dataGraphRef,
		PrimaryID:    spec.PrimaryID,
		Root:         spec.Root,
		Timestamp:    spec.Timestamp,
	}

	return resource, nil
}


// extractDataGraphResource creates a DataGraphResource from a spec
func (p *Provider) extractDataGraphResource(spec *dgModel.DataGraphSpec) (*dgModel.DataGraphResource, error) {
	resource := &dgModel.DataGraphResource{
		ID:        spec.ID,
		AccountID: spec.AccountID,
	}
	return resource, nil
}

func (p *Provider) processInlineRelationship(dataGraphID, sourceModelID string, relSpec *dgModel.RelationshipSpec) error {
	// Extract relationship resource
	resource, err := p.extractRelationshipResource(dataGraphID, sourceModelID, relSpec)
	if err != nil {
		return fmt.Errorf("extracting relationship: %w", err)
	}

	// Register with relationship handler
	if err := p.relationshipHandler.AddResource(relSpec.ID, resource); err != nil {
		return fmt.Errorf("adding relationship: %w", err)
	}

	return nil
}


func (p *Provider) extractRelationshipResource(dataGraphID, sourceModelID string, spec *dgModel.RelationshipSpec) (*dgModel.RelationshipResource, error) {
	// Create URN for data graph (parent)
	dataGraphURN := resources.URN(dataGraphID, datagraph.HandlerMetadata.ResourceType)
	dataGraphRef := relationship.CreateDataGraphReference(dataGraphURN)

	// Create URN for source model (from)
	sourceModelURN := resources.URN(sourceModelID, model.HandlerMetadata.ResourceType)
	sourceModelRef := relationship.CreateModelReference(sourceModelURN)

	// Parse target model reference from spec
	targetModelURN, err := parseModelReference(spec.Target)
	if err != nil {
		return nil, fmt.Errorf("parsing target model reference: %w", err)
	}
	targetModelRef := relationship.CreateModelReference(targetModelURN)

	return &dgModel.RelationshipResource{
		ID:             spec.ID,
		DisplayName:    spec.DisplayName,
		DataGraphRef:   dataGraphRef,
		SourceModelRef: sourceModelRef,
		TargetModelRef: targetModelRef,
		SourceJoinKey:  spec.SourceJoinKey,
		TargetJoinKey:  spec.TargetJoinKey,
		Cardinality:    spec.Cardinality,
	}, nil
}

// exportLookups holds pre-computed indexes used when formatting data graph specs for export
type exportLookups struct {
	modelsByDG       map[string][]*resources.RemoteResource
	relsByKey        map[relKey][]*resources.RemoteResource
	modelExternalIDs map[string]string
}

type relKey struct {
	dataGraphID   string
	sourceModelID string
}

// FormatForExport overrides BaseProvider.FormatForExport to produce composite specs.
// Data graphs are composite resources: a single YAML spec contains a data graph with
// inline models and inline relationships, so export logic lives at the provider level.
func (p *Provider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	importableDataGraphs := collection.GetAll(datagraph.HandlerMetadata.ResourceType)
	if len(importableDataGraphs) == 0 {
		return nil, nil, nil
	}

	lookups := p.groupResourcesByDataGraph(collection, importableDataGraphs)

	// Sort data graphs by external ID for deterministic output
	sortedDGs := make([]*resources.RemoteResource, 0, len(importableDataGraphs))
	for _, dg := range importableDataGraphs {
		sortedDGs = append(sortedDGs, dg)
	}
	slices.SortFunc(sortedDGs, func(a, b *resources.RemoteResource) int {
		return cmp.Compare(a.ExternalID, b.ExternalID)
	})

	var (
		result  []writer.FormattableEntity
		entries []importmanifest.ImportEntry
	)
	for _, dgResource := range sortedDGs {
		entity, dgEntries, err := p.formatDataGraphSpec(dgResource, lookups)
		if err != nil {
			return nil, nil, err
		}
		result = append(result, entity)
		entries = append(entries, dgEntries...)
	}

	return result, entries, nil
}

// groupResourcesByDataGraph builds lookup indexes from the importable collection,
// filtering children to only those whose parent data graph is also importable.
func (p *Provider) groupResourcesByDataGraph(
	collection *resources.RemoteResources,
	importableDataGraphs map[string]*resources.RemoteResource,
) *exportLookups {
	importableModels := collection.GetAll(model.HandlerMetadata.ResourceType)
	importableRelationships := collection.GetAll(relationship.HandlerMetadata.ResourceType)

	importableDGIDs := make(map[string]bool, len(importableDataGraphs))
	for _, dg := range importableDataGraphs {
		importableDGIDs[dg.ID] = true
	}

	// Models grouped by DataGraphID, excluding models under managed (non-importable) data graphs
	modelsByDG := make(map[string][]*resources.RemoteResource)
	for _, m := range importableModels {
		remote := m.Data.(*dgModel.RemoteModel)
		if !importableDGIDs[remote.DataGraphID] {
			continue
		}
		modelsByDG[remote.DataGraphID] = append(modelsByDG[remote.DataGraphID], m)
	}

	// Relationships grouped by DataGraphID and SourceModelID
	relsByKey := make(map[relKey][]*resources.RemoteResource)
	for _, r := range importableRelationships {
		remote := r.Data.(*dgModel.RemoteRelationship)
		key := relKey{dataGraphID: remote.DataGraphID, sourceModelID: remote.SourceModelID}
		relsByKey[key] = append(relsByKey[key], r)
	}

	// Model remote ID -> externalID for resolving relationship target references
	modelExternalIDs := make(map[string]string)
	for _, m := range importableModels {
		remote := m.Data.(*dgModel.RemoteModel)
		modelExternalIDs[remote.ID] = m.ExternalID
	}

	return &exportLookups{
		modelsByDG:       modelsByDG,
		relsByKey:        relsByKey,
		modelExternalIDs: modelExternalIDs,
	}
}

// formatDataGraphSpec produces a single FormattableEntity for one data graph and its children,
// plus the set of ImportEntry rows that feed the aggregated import-manifest.
func (p *Provider) formatDataGraphSpec(
	dgResource *resources.RemoteResource,
	lookups *exportLookups,
) (writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	remoteDG := dgResource.Data.(*dgModel.RemoteDataGraph)

	entries := []importmanifest.ImportEntry{
		{
			WorkspaceID: remoteDG.WorkspaceID,
			URN:         resources.URN(dgResource.ExternalID, datagraph.HandlerMetadata.ResourceType),
			RemoteID:    remoteDG.ID,
		},
	}

	modelSpecs, modelEntries := p.buildInlineModelSpecs(remoteDG.ID, remoteDG.WorkspaceID, lookups)
	entries = append(entries, modelEntries...)

	metadata := specs.Metadata{
		Name: datagraph.HandlerMetadata.SpecMetadataName,
	}

	metadataMap, err := metadata.ToMap()
	if err != nil {
		return writer.FormattableEntity{}, nil, fmt.Errorf("converting metadata to map: %w", err)
	}

	specBody := map[string]any{
		"id":         dgResource.ExternalID,
		"account_id": remoteDG.AccountID,
	}
	if len(modelSpecs) > 0 {
		specBody["models"] = modelSpecs
	}

	spec := &specs.Spec{
		Version:  specs.SpecVersionV1,
		Kind:     datagraph.HandlerMetadata.SpecKind,
		Metadata: metadataMap,
		Spec:     specBody,
	}

	return writer.FormattableEntity{
		Content:      spec,
		RelativePath: filepath.Join("data-graphs", fmt.Sprintf("%s.yaml", dgResource.ExternalID)),
	}, entries, nil
}

// buildInlineModelSpecs builds model specs with inline relationships for a single data graph,
// returning both the specs and the ImportEntry rows for all included child resources.
func (p *Provider) buildInlineModelSpecs(
	dataGraphID string,
	workspaceID string,
	lookups *exportLookups,
) ([]dgModel.ModelSpec, []importmanifest.ImportEntry) {
	dgModels := lookups.modelsByDG[dataGraphID]
	slices.SortFunc(dgModels, func(a, b *resources.RemoteResource) int {
		return cmp.Compare(a.ExternalID, b.ExternalID)
	})

	var (
		modelSpecs []dgModel.ModelSpec
		entries    []importmanifest.ImportEntry
	)

	for _, modelResource := range dgModels {
		remoteModel := modelResource.Data.(*dgModel.RemoteModel)

		entries = append(entries, importmanifest.ImportEntry{
			WorkspaceID: workspaceID,
			URN:         resources.URN(modelResource.ExternalID, model.HandlerMetadata.ResourceType),
			RemoteID:    remoteModel.ID,
		})

		// Sort relationships for this model by external ID
		key := relKey{dataGraphID: dataGraphID, sourceModelID: remoteModel.ID}
		modelRels := lookups.relsByKey[key]
		slices.SortFunc(modelRels, func(a, b *resources.RemoteResource) int {
			return cmp.Compare(a.ExternalID, b.ExternalID)
		})

		var relSpecs []dgModel.RelationshipSpec
		for _, relResource := range modelRels {
			remoteRel := relResource.Data.(*dgModel.RemoteRelationship)

			// Skip relationships whose target model is not in the importable collection
			targetExternalID := lookups.modelExternalIDs[remoteRel.TargetModelID]
			if targetExternalID == "" {
				continue
			}

			entries = append(entries, importmanifest.ImportEntry{
				WorkspaceID: workspaceID,
				URN:         resources.URN(relResource.ExternalID, relationship.HandlerMetadata.ResourceType),
				RemoteID:    remoteRel.ID,
			})

			relSpecs = append(relSpecs, dgModel.RelationshipSpec{
				ID:            relResource.ExternalID,
				DisplayName:   remoteRel.Name,
				Cardinality:   remoteRel.Cardinality,
				Target:        fmt.Sprintf("#%s:%s", model.HandlerMetadata.ResourceType, targetExternalID),
				SourceJoinKey: remoteRel.SourceJoinKey,
				TargetJoinKey: remoteRel.TargetJoinKey,
			})
		}

		modelSpecs = append(modelSpecs, dgModel.ModelSpec{
			ID:            modelResource.ExternalID,
			DisplayName:   remoteModel.Name,
			Type:          remoteModel.Type,
			Table:         remoteModel.TableRef,
			Description:   remoteModel.Description,
			Relationships: relSpecs,
			PrimaryID:     remoteModel.PrimaryID,
			Root:          remoteModel.Root,
			Timestamp:     remoteModel.Timestamp,
		})
	}

	return modelSpecs, entries
}

// parseModelReference parses a model reference like '#data-graph-model:user' and returns the URN
func parseModelReference(ref string) (string, error) {
	if !strings.HasPrefix(ref, "#data-graph-model:") {
		return "", fmt.Errorf("invalid model reference format, expected '#data-graph-model:<id>', got %q", ref)
	}
	modelID := strings.TrimPrefix(ref, "#data-graph-model:")
	if modelID == "" {
		return "", fmt.Errorf("model ID cannot be empty in reference")
	}
	return resources.URN(modelID, model.HandlerMetadata.ResourceType), nil
}
