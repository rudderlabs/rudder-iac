package transformations

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/display"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/parser"
	lrules "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/rules/library"
	trules "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/rules/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	vdocs "github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
	"github.com/samber/lo"

	tfdocs "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/docs"
)

var log = logger.New("transformationsprovider")

// pendingDeletes collects delete intents for deferred execution.
// Deletes are deferred because the backend enforces referential integrity
// against published state — a library cannot be deleted while a published
// transformation still imports it. By deferring deletes until after batch
// publish in ConsolidateSync, the published dependency graph is updated first.
type pendingDeletes struct {
	mu              sync.RWMutex
	transformations []string // remote IDs
	libraries       []string // remote IDs
}

// plannedPublish stages and publishes, as one batch, every transformation and
// library change in the plan the provider last checked. The backend links a
// destination only to a published transformation, and the destination's
// operation waits only on the operation of the transformation it links. So
// whichever planned change the syncer applies first does the whole batch, and
// the others return its outcome.
type plannedPublish struct {
	once    sync.Once
	ops     []*planner.Operation
	prior   *state.State // transformation and library state the plan was made from
	outputs map[string]any
	err     error
}

func (b *plannedPublish) includes(urn string) bool {
	return b != nil && slices.ContainsFunc(b.ops, func(op *planner.Operation) bool {
		return op.Resource.URN() == urn
	})
}

// stageTask stages one planned change.
type stageTask struct {
	op *planner.Operation
}

func (t *stageTask) Id() string {
	return t.op.Resource.URN()
}

func (t *stageTask) Dependencies() []string {
	return nil
}

// Provider wraps BaseProvider and adds transformations-specific functionality
type Provider struct {
	*provider.BaseProvider
	store            transformations.TransformationStore
	pendingDeletes   pendingDeletes
	planned          *plannedPublish
	stageConcurrency int
}

// NewProvider creates a new transformations provider with all resource handlers
func NewProvider(client *client.Client) *Provider {
	// Create the transformation store
	store := transformations.NewRudderTransformationStore(client)
	return NewProviderWithStore(store)
}

// NewProviderWithStore creates a new transformations provider with a custom store
// This is useful for testing with mock stores
func NewProviderWithStore(store transformations.TransformationStore) *Provider {
	// Register handlers - libraries first (dependencies), then transformations
	handlers := []provider.Handler{
		library.NewHandler(store),
		transformation.NewHandler(store),
	}

	return &Provider{
		BaseProvider: provider.NewBaseProvider(handlers),
		store:        store,
		// The planned batch stages changes the syncer would otherwise have
		// applied concurrently.
		stageConcurrency: config.GetConfig().Concurrency.Syncer,
	}
}

func (p *Provider) LoadLegacySpec(path string, s *specs.Spec) error {
	return fmt.Errorf("transformation specs require version '%s', got '%s'. Legacy versions are not supported", specs.SpecVersionV1, s.Version)
}

// SupportedMatchPatterns declares the (kind, version) pairs this provider fully handles.
// Transformations only support the V1 version; legacy versions are explicitly rejected.
func (p *Provider) SupportedMatchPatterns() []vrules.MatchPattern {
	var patterns []vrules.MatchPattern
	patterns = append(patterns, prules.V1VersionPatterns(ttypes.TransformationResourceType)...)
	patterns = append(patterns, prules.V1VersionPatterns(ttypes.LibraryResourceType)...)
	return patterns
}

func (p *Provider) SyntacticRules() []vrules.Rule {
	return []vrules.Rule{
		trules.NewTransformationSpecSyntaxValidRule(),
		lrules.NewLibrarySpecSyntaxValidRule(),
	}
}

func (p *Provider) SemanticRules() []vrules.Rule {
	return []vrules.Rule{
		trules.NewTransformationSemanticValidRule(),
		lrules.NewLibrarySemanticValidRule(),
	}
}

// RuleDocEntries returns the authored documentation fragments embedded with
// the transformations provider, joined to registered rules by the docs generator.
func (p *Provider) RuleDocEntries() []vdocs.RuleDocEntry {
	entries, _ := vdocs.LoadRuleDocEntries(tfdocs.FragmentsFS, ".")
	return entries
}

// MapRemoteToState overrides BaseProvider.MapRemoteToState to populate dependencies for transformations
// Dependencies are extracted from the remote transformation's Imports field and resolved to library URNs
func (p *Provider) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	// Call base implementation to get initial state
	st, err := p.BaseProvider.MapRemoteToState(collection)
	if err != nil {
		return nil, err
	}

	// Build handleName -> library URN mapping from remote libraries
	libraryResources := collection.GetAll(library.HandlerMetadata.ResourceType)
	handleNameToURN := make(map[string]string)

	for _, remoteLib := range libraryResources {
		lib := remoteLib.Data.(*model.RemoteLibrary)
		libraryURN := resources.URN(lib.ExternalID, library.HandlerMetadata.ResourceType)
		handleNameToURN[lib.ImportName] = libraryURN
	}

	// Populate dependencies for each transformation using the Imports field
	transformationResources := collection.GetAll(transformation.HandlerMetadata.ResourceType)

	for _, remoteTrans := range transformationResources {
		trans := remoteTrans.Data.(*model.RemoteTransformation)
		transURN := resources.URN(trans.ExternalID, transformation.HandlerMetadata.ResourceType)
		resourceState := st.GetResource(transURN)

		// Resolve imports to library URNs
		dependencies := make([]string, 0, len(trans.Imports))
		for _, handleName := range trans.Imports {
			libraryURN, exists := handleNameToURN[handleName]
			if !exists {
				log.Warn("transformation references library not found in managed resources",
					"transformation", trans.ExternalID,
					"import_name", handleName)
				continue
			}
			dependencies = append(dependencies, libraryURN)
		}

		// Update resource state with dependencies
		resourceState.Dependencies = dependencies
	}

	return st, nil
}

// ResourceGraph overrides BaseProvider.ResourceGraph to add transformation→library dependencies
// by parsing transformation code to extract library imports
func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	graph := resources.NewGraph()

	// libraries
	libraryHandler, ok := p.GetHandler(library.HandlerMetadata.ResourceType)
	if !ok {
		return nil, fmt.Errorf("library handler not found")
	}

	libraryResources, err := libraryHandler.Resources()
	if err != nil {
		return nil, fmt.Errorf("getting library resources: %w", err)
	}

	// Build handleName → URN mapping
	handleNameToURN := make(map[string]string)

	for _, r := range libraryResources {
		lib, ok := r.RawData().(*model.LibraryResource)
		if !ok {
			return nil, fmt.Errorf("invalid RawData type for library %s: expected *model.LibraryResource", r.URN())
		}

		handleNameToURN[lib.ImportName] = r.URN()
		graph.AddResource(r)
	}

	// transformations
	transformationHandler, ok := p.GetHandler(transformation.HandlerMetadata.ResourceType)
	if !ok {
		return nil, fmt.Errorf("transformation handler not found")
	}

	transformationResources, err := transformationHandler.Resources()
	if err != nil {
		return nil, fmt.Errorf("getting transformation resources: %w", err)
	}

	for _, r := range transformationResources {
		tf, ok := r.RawData().(*model.TransformationResource)
		if !ok {
			return nil, fmt.Errorf("invalid RawData type for transformation %s: expected *model.TransformationResource", r.URN())
		}

		codeParser, err := parser.NewParser(tf.Language)
		if err != nil {
			return nil, fmt.Errorf("creating parser for %s: %w", r.URN(), err)
		}

		handleNames, err := codeParser.ExtractImports(tf.Code)
		if err != nil {
			log.Warn("failed to parse imports for transformation",
				"transformation", r.URN(),
				"error", err)
		}

		// Resolve handleNames to library URNs and add dependencies
		for _, handleName := range handleNames {
			libraryURN, exists := handleNameToURN[handleName]
			if !exists {
				log.Warn("transformation import could not be resolved during graph construction",
					"transformation", r.ID(),
					"import_name", handleName)
				continue
			}

			graph.AddDependency(r.URN(), libraryURN)
		}
		graph.AddResource(r)
	}

	return graph, nil
}

// CheckPlan never refuses a plan. It keeps the plan's transformation and
// library changes for plannedPublish.
func (p *Provider) CheckPlan(_ context.Context, plan *planner.Plan, st *state.State) error {
	planned := &plannedPublish{prior: state.EmptyState()}
	for _, op := range plan.Operations {
		if _, ok := p.GetHandler(op.Resource.Type()); !ok || op.Type == planner.Delete {
			continue
		}
		planned.ops = append(planned.ops, op)
	}
	// Copied now because the syncer writes to st while operations run.
	for urn, rs := range st.Resources {
		if _, ok := p.GetHandler(rs.Type); ok {
			planned.prior.Resources[urn] = rs
		}
	}
	p.planned = planned
	return nil
}

// CreateRaw, UpdateRaw and ImportRaw return the planned batch's outcome for the
// changes CheckPlan kept. Anything else is only staged, for ConsolidateSync to
// publish.
func (p *Provider) CreateRaw(ctx context.Context, r *resources.Resource) (any, error) {
	if !p.planned.includes(r.URN()) {
		return p.BaseProvider.CreateRaw(ctx, r)
	}
	return p.applyPlanned(ctx, r.URN())
}

func (p *Provider) UpdateRaw(ctx context.Context, r *resources.Resource, oldData any, oldState any) (any, error) {
	if !p.planned.includes(r.URN()) {
		return p.BaseProvider.UpdateRaw(ctx, r, oldData, oldState)
	}
	return p.applyPlanned(ctx, r.URN())
}

func (p *Provider) ImportRaw(ctx context.Context, r *resources.Resource, remoteID string) (any, error) {
	if !p.planned.includes(r.URN()) {
		return p.BaseProvider.ImportRaw(ctx, r, remoteID)
	}
	return p.applyPlanned(ctx, r.URN())
}

func (p *Provider) applyPlanned(ctx context.Context, urn string) (any, error) {
	planned := p.planned
	planned.once.Do(func() {
		planned.outputs, planned.err = p.stageAndPublish(ctx, planned)
	})
	if planned.err != nil {
		return nil, planned.err
	}
	return planned.outputs[urn], nil
}

// stageAndPublish stages every planned change, then publishes them together
// with what must go live alongside them. It returns their states marked
// unmodified, so ConsolidateSync does not publish them again.
func (p *Provider) stageAndPublish(ctx context.Context, planned *plannedPublish) (map[string]any, error) {
	tasks := lo.Map(planned.ops, func(op *planner.Operation, _ int) tasker.Task {
		return &stageTask{op: op}
	})
	staged := tasker.NewResults[any]()
	errs := tasker.RunTasks(ctx, tasks, p.stageConcurrency, false, func(task tasker.Task) error {
		t, ok := task.(*stageTask)
		if !ok {
			return fmt.Errorf("task is not a stage task")
		}
		out, err := p.stage(ctx, t.op, planned.prior)
		if err != nil {
			return err
		}
		staged.Store(t.Id(), out)
		return nil
	})
	if len(errs) > 0 {
		return nil, fmt.Errorf("staging transformations and libraries: %w", errors.Join(errs...))
	}

	st := state.EmptyState()
	maps.Copy(st.Resources, planned.prior.Resources)
	for _, op := range planned.ops {
		out, _ := staged.Get(op.Resource.URN())
		st.AddResource(&state.ResourceState{ID: op.Resource.ID(), Type: op.Resource.Type(), OutputRaw: out})
	}

	graph, err := p.ResourceGraph()
	if err != nil {
		return nil, fmt.Errorf("building resource graph: %w", err)
	}
	req, err := p.buildBatchPublishRequest(graph, st)
	if err != nil {
		return nil, fmt.Errorf("building batch publish request: %w", err)
	}
	resp, err := p.store.BatchPublish(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("batch publishing %d transformations and %d libraries: %w",
			len(req.Transformations), len(req.Libraries), err)
	}
	if !resp.Published {
		return nil, rejectedPublishError(resp.ValidationOutput)
	}
	log.Info("Published transformations and libraries ahead of their dependents",
		"transformations", len(req.Transformations), "libraries", len(req.Libraries))

	outputs := make(map[string]any, len(planned.ops))
	for _, op := range planned.ops {
		out := st.GetResource(op.Resource.URN()).OutputRaw
		switch s := out.(type) {
		case *model.TransformationState:
			s.Modified = false
		case *model.LibraryState:
			s.Modified = false
		}
		outputs[op.Resource.URN()] = out
	}
	return outputs, nil
}

// stage applies one planned change as an unpublished draft, as the syncer
// would have.
func (p *Provider) stage(ctx context.Context, op *planner.Operation, prior *state.State) (any, error) {
	r := op.Resource
	switch op.Type {
	case planner.Update:
		old := prior.GetResource(r.URN())
		if old == nil {
			return nil, fmt.Errorf("resource not found in state: %s", r.URN())
		}
		return p.BaseProvider.UpdateRaw(ctx, r, old.InputRaw, old.OutputRaw)
	case planner.Import:
		return p.BaseProvider.ImportRaw(ctx, r, r.ImportMetadata().RemoteId)
	default:
		return p.BaseProvider.CreateRaw(ctx, r)
	}
}

// rejectedPublishError names what failed the backend's publish tests. The full
// results are left to `transformations test`: while operations run, the
// apply's progress display owns the terminal.
func rejectedPublishError(out transformations.ValidationOutput) error {
	failed := lo.FilterMap(out.Libraries, func(lib transformations.LibraryTestResult, _ int) (string, bool) {
		return fmt.Sprintf("library %q", lib.Name), !lib.Pass
	})
	failed = append(failed, lo.FilterMap(out.Transformations, func(tr transformations.TransformationTestResult, _ int) (string, bool) {
		return fmt.Sprintf("transformation %q", tr.Name), !tr.Pass
	})...)
	return fmt.Errorf("batch publish validation failed for %s: run `rudder-cli transformations test --modified` for the results",
		strings.Join(failed, ", "))
}

// DeleteRaw overrides BaseProvider.DeleteRaw to defer deletion until ConsolidateSync.
// Instead of immediately deleting resources, it records the delete intent so that
// batch publish can update the published dependency graph first.
func (p *Provider) DeleteRaw(_ context.Context, _ string, resourceType string, _ any, oldState any) error {
	switch resourceType {
	case transformation.HandlerMetadata.ResourceType:
		st, ok := oldState.(*model.TransformationState)
		if !ok {
			return fmt.Errorf("invalid state type for transformation delete")
		}
		p.pendingDeletes.mu.Lock()
		p.pendingDeletes.transformations = append(p.pendingDeletes.transformations, st.ID)
		p.pendingDeletes.mu.Unlock()

	case library.HandlerMetadata.ResourceType:
		st, ok := oldState.(*model.LibraryState)
		if !ok {
			return fmt.Errorf("invalid state type for library delete")
		}
		p.pendingDeletes.mu.Lock()
		p.pendingDeletes.libraries = append(p.pendingDeletes.libraries, st.ID)
		p.pendingDeletes.mu.Unlock()

	default:
		return fmt.Errorf("unsupported resource type for delete: %s", resourceType)
	}
	return nil
}

// ConsolidateSync publishes any change still waiting to go live, then runs the
// deferred deletes. In an apply nothing is normally left waiting: plannedPublish
// has already published the plan's changes before their dependents ran.
func (p *Provider) ConsolidateSync(ctx context.Context, graph *resources.Graph, st *state.State) error {
	req, err := p.buildBatchPublishRequest(graph, st)
	if err != nil {
		return fmt.Errorf("building batch publish request: %w", err)
	}

	if len(req.Transformations) > 0 || len(req.Libraries) > 0 {
		resp, err := p.store.BatchPublish(ctx, req)
		if err != nil {
			return fmt.Errorf("batch publishing %d transformations and %d libraries: %w",
				len(req.Transformations), len(req.Libraries), err)
		}

		if !resp.Published {
			// Convert response to TestResults and display using shared result displayer
			results := p.buildTestResultsFromResponse(req, resp)
			displayer := display.NewResultDisplayer(false)
			displayer.Display(results)
			return fmt.Errorf("batch publish validation failed")
		}

		log.Info("Successfully published transformations and libraries",
			"transformations", len(req.Transformations), "libraries", len(req.Libraries))
	}

	// Phase 2: Execute deferred deletes (now safe — published state is current)
	if err := p.executePendingDeletes(ctx); err != nil {
		return fmt.Errorf("executing pending deletes: %w", err)
	}

	return nil
}

// executePendingDeletes processes all deferred delete operations.
// Transformations are deleted first since they depend on libraries.
func (p *Provider) executePendingDeletes(ctx context.Context) error {
	p.pendingDeletes.mu.RLock()
	defer p.pendingDeletes.mu.RUnlock()

	for _, id := range p.pendingDeletes.transformations {
		if err := p.store.DeleteTransformation(ctx, id); err != nil {
			return fmt.Errorf("deleting transformation %s: %w", id, err)
		}
	}

	for _, id := range p.pendingDeletes.libraries {
		if err := p.store.DeleteLibrary(ctx, id); err != nil {
			return fmt.Errorf("deleting library %s: %w", id, err)
		}
	}

	log.Info("Successfully deleted transformations and libraries",
		"transformations", len(p.pendingDeletes.transformations), "libraries", len(p.pendingDeletes.libraries))
	return nil
}

// buildBatchPublishRequest builds the batch publish request by extracting version IDs from state.
// Only publishes modified resources and their dependencies/dependents.
// Includes test suites for transformations when available.
func (p *Provider) buildBatchPublishRequest(graph *resources.Graph, st *state.State) (*transformations.BatchPublishRequest, error) {
	transformationURNs, libraryURNs := p.collectModifiedResources(st)

	var (
		transformationsToPublish = make(map[string]struct{})
		librariesToPublish       = make(map[string]struct{})
	)

	for _, urn := range transformationURNs {
		transformationsToPublish[urn] = struct{}{}

		for _, libDep := range graph.GetDependencies(urn) {
			librariesToPublish[libDep] = struct{}{}
		}
	}

	for _, urn := range libraryURNs {
		librariesToPublish[urn] = struct{}{}

		for _, tranURN := range graph.GetDependents(urn) {
			// This transformation needs to be published because its library changed
			transformationsToPublish[tranURN] = struct{}{}

			for _, libURN := range graph.GetDependencies(tranURN) {
				librariesToPublish[libURN] = struct{}{}
			}
		}
	}

	return p.buildRequestFromURNs(graph, st, transformationsToPublish, librariesToPublish)
}

func (p *Provider) collectModifiedResources(st *state.State) (transformationURNs []string, libraryURNs []string) {
	for _, resource := range st.Resources {

		switch resource.Type {
		case library.HandlerMetadata.ResourceType:
			if libState, ok := resource.OutputRaw.(*model.LibraryState); ok {
				if libState.Modified {
					urn := resources.URN(resource.ID, resource.Type)
					libraryURNs = append(libraryURNs, urn)
				}
			}

		case transformation.HandlerMetadata.ResourceType:
			if transState, ok := resource.OutputRaw.(*model.TransformationState); ok {
				if transState.Modified {
					urn := resources.URN(resource.ID, resource.Type)
					transformationURNs = append(transformationURNs, urn)
				}
			}
		}
	}

	return transformationURNs, libraryURNs
}

func (p *Provider) buildRequestFromURNs(
	graph *resources.Graph,
	st *state.State,
	transformationURNs map[string]struct{},
	libraryURNs map[string]struct{},
) (*transformations.BatchPublishRequest, error) {
	req := &transformations.BatchPublishRequest{
		Transformations: []transformations.BatchPublishTransformation{},
		Libraries:       []transformations.BatchPublishLibrary{},
	}

	for libURN := range libraryURNs {
		lib, err := p.buildLibraryPublishItem(st, libURN)
		if err != nil {
			return nil, err
		}
		if lib != nil {
			req.Libraries = append(req.Libraries, *lib)
		}
	}

	for transURN := range transformationURNs {
		trans, err := p.buildTransformationPublishItem(graph, st, transURN)
		if err != nil {
			return nil, err
		}
		if trans != nil {
			req.Transformations = append(req.Transformations, *trans)
		}
	}

	return req, nil
}

func (p *Provider) buildLibraryPublishItem(st *state.State, libURN string) (*transformations.BatchPublishLibrary, error) {
	stateResource := st.GetResource(libURN)

	libState, ok := stateResource.OutputRaw.(*model.LibraryState)
	if !ok {
		return nil, fmt.Errorf("resource %s has invalid OutputRaw type for library", libURN)
	}
	if libState.VersionID == "" {
		return nil, fmt.Errorf("resource %s has empty version ID", libURN)
	}

	return &transformations.BatchPublishLibrary{
		VersionID: libState.VersionID,
	}, nil
}

func (p *Provider) buildTransformationPublishItem(graph *resources.Graph, st *state.State, transURN string) (*transformations.BatchPublishTransformation, error) {
	stateResource := st.GetResource(transURN)

	transState, ok := stateResource.OutputRaw.(*model.TransformationState)
	if !ok {
		return nil, fmt.Errorf("resource %s has invalid OutputRaw type for transformation", transURN)
	}
	if transState.VersionID == "" {
		return nil, fmt.Errorf("resource %s has empty version ID", transURN)
	}

	testSuite, err := p.resolveTestSuite(graph, transURN)
	if err != nil {
		return nil, fmt.Errorf("resolving test suite for %s: %w", transURN, err)
	}

	return &transformations.BatchPublishTransformation{
		VersionID: transState.VersionID,
		TestSuite: testSuite,
	}, nil
}

func (p *Provider) resolveTestSuite(graph *resources.Graph, transURN string) ([]transformations.TestDefinition, error) {
	graphResource, exists := graph.GetResource(transURN)
	if !exists {
		return nil, fmt.Errorf("resource %s not found in graph", transURN)
	}

	transResource, ok := graphResource.RawData().(*model.TransformationResource)
	if !ok {
		return nil, fmt.Errorf("resource %s has invalid RawData type for transformation", transURN)
	}

	testDefs, err := p.resolveTestDefinitions(transResource)
	if err != nil {
		return nil, fmt.Errorf("resolving test definitions: %w", err)
	}

	testSuite := make([]transformations.TestDefinition, 0, len(testDefs))
	for _, td := range testDefs {
		testSuite = append(testSuite, *td)
	}

	return testSuite, nil
}

func (p *Provider) resolveTestDefinitions(trans *model.TransformationResource) ([]*transformations.TestDefinition, error) {
	return testorchestrator.ResolveTestDefinitions(trans)
}

func (p *Provider) buildTestResultsFromResponse(
	req *transformations.BatchPublishRequest,
	resp *transformations.BatchPublishResponse,
) *testorchestrator.TestResults {
	testDefsByVersionID := lo.SliceToMap(req.Transformations, func(trans transformations.BatchPublishTransformation) (string, []*transformations.TestDefinition) {
		defs := lo.Map(trans.TestSuite, func(t transformations.TestDefinition, i int) *transformations.TestDefinition {
			return &trans.TestSuite[i]
		})
		return trans.VersionID, defs
	})

	// Map results to their definitions by matching versionID
	trResults := lo.Map(resp.ValidationOutput.Transformations, func(tr transformations.TransformationTestResult, _ int) *testorchestrator.TransformationTestWithDefinitions {
		return &testorchestrator.TransformationTestWithDefinitions{
			Result:      &tr,
			Definitions: testDefsByVersionID[tr.VersionID],
		}
	})

	return &testorchestrator.TestResults{
		Libraries:       resp.ValidationOutput.Libraries,
		Transformations: trResults,
	}
}
