package transformations_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	tc "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
)

const linkedDestinationType = "linked-destination"

// linkedDestination stands in for a destination linked to a transformation.
type linkedDestination struct {
	Transformation *resources.PropertyRef
}

// linkingProvider serves the transformations provider alongside a destination
// stand-in that, like the backend, can link only a published transformation.
type linkingProvider struct {
	*transformations.Provider
	published func(transformationID string) bool
}

func (p *linkingProvider) CreateRaw(ctx context.Context, r *resources.Resource) (any, error) {
	if r.Type() != linkedDestinationType {
		return p.Provider.CreateRaw(ctx, r)
	}
	transformationID := r.RawData().(*linkedDestination).Transformation.Value
	if !p.published(transformationID) {
		return nil, fmt.Errorf("transformation %s is not published", transformationID)
	}
	return &linkedDestination{}, nil
}

// publishLog records the version IDs a mock store was asked to publish, safely
// across the syncer's concurrent operations.
type publishLog struct {
	mu       sync.Mutex
	calls    int
	versions []string
}

func (l *publishLog) record(req *tc.BatchPublishRequest) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.calls++
	for _, t := range req.Transformations {
		l.versions = append(l.versions, t.VersionID)
	}
	for _, lib := range req.Libraries {
		l.versions = append(l.versions, lib.VersionID)
	}
}

func (l *publishLog) has(versionID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Contains(l.versions, versionID)
}

func transformationSpec(id, code string) *specs.Spec {
	return &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    "transformation",
		Spec: map[string]any{
			"id":       id,
			"name":     id,
			"language": "javascript",
			"code":     code,
		},
	}
}

func librarySpec(id, importName, code string) *specs.Spec {
	return &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    "transformation-library",
		Spec: map[string]any{
			"id":          id,
			"name":        id,
			"language":    "javascript",
			"code":        code,
			"import_name": importName,
		},
	}
}

func loadSpecs(t *testing.T, p *transformations.Provider, specs ...*specs.Spec) *resources.Graph {
	t.Helper()

	for i, s := range specs {
		require.NoError(t, p.LoadSpec(fmt.Sprintf("spec-%d.yaml", i), s))
	}
	graph, err := p.ResourceGraph()
	require.NoError(t, err)
	return graph
}

func graphResource(t *testing.T, graph *resources.Graph, urn string) *resources.Resource {
	t.Helper()

	r, ok := graph.GetResource(urn)
	require.True(t, ok, "resource %s not in graph", urn)
	return r
}

// The backend links only a published transformation, and a destination's
// operation waits only on the operation of the transformation it links. So a
// transformation created in an apply must be published before that operation
// returns, whatever else is staged with it and in whichever order the syncer
// runs the plan.
func TestApplyPublishesTransformationBeforeLinkingIt(t *testing.T) {
	t.Parallel()

	const (
		transformationCode = "export function transformEvent(event, metadata) { return event; }"
		importingCode      = "import mathLibrary from 'mathLibrary';\nexport function transformEvent(event, metadata) { return event; }"
	)

	cases := []struct {
		name            string
		specs           []*specs.Spec
		remoteLibraries []*tc.TransformationLibrary
		wantPublished   []string
	}{
		{
			name:          "new transformation",
			specs:         []*specs.Spec{transformationSpec("trans-1", transformationCode)},
			wantPublished: []string{"trans-1-v1"},
		},
		{
			// Run one at a time, the syncer creates before it updates, so the
			// library's update comes after the destination's create.
			name: "new transformation importing a library updated in the same apply",
			specs: []*specs.Spec{
				librarySpec("lib-1", "mathLibrary", "export function add(a, b) { return b + a; }"),
				transformationSpec("trans-1", importingCode),
			},
			remoteLibraries: []*tc.TransformationLibrary{{
				ID:         "remote-lib-1",
				VersionID:  "lib-1-v1",
				ExternalID: "lib-1",
				Name:       "lib-1",
				Language:   "javascript",
				ImportName: "mathLibrary",
				Code:       "export function add(a, b) { return a + b; }",
			}},
			wantPublished: []string{"trans-1-v1", "lib-1-v2"},
		},
		{
			name: "new transformation importing an unchanged library",
			specs: []*specs.Spec{
				librarySpec("lib-1", "mathLibrary", "export function add(a, b) { return a + b; }"),
				transformationSpec("trans-1", importingCode),
			},
			remoteLibraries: []*tc.TransformationLibrary{{
				ID:         "remote-lib-1",
				VersionID:  "lib-1-v1",
				ExternalID: "lib-1",
				Name:       "lib-1",
				Language:   "javascript",
				ImportName: "mathLibrary",
				Code:       "export function add(a, b) { return a + b; }",
			}},
			wantPublished: []string{"trans-1-v1", "lib-1-v1"},
		},
	}

	for _, tt := range cases {
		for _, concurrency := range []int{1, 4} {
			t.Run(fmt.Sprintf("%s/concurrency=%d", tt.name, concurrency), func(t *testing.T) {
				t.Parallel()

				var log publishLog
				store := &testutil.MockTransformationStore{
					ListTransformationsFunc: func(context.Context) ([]*tc.Transformation, error) {
						return nil, nil
					},
					ListLibrariesFunc: func(context.Context) ([]*tc.TransformationLibrary, error) {
						return tt.remoteLibraries, nil
					},
					CreateTransformationFunc: func(_ context.Context, req *tc.CreateTransformationRequest, _ bool) (*tc.Transformation, error) {
						return &tc.Transformation{ID: "remote-" + req.ExternalID, VersionID: req.ExternalID + "-v1"}, nil
					},
					UpdateLibraryFunc: func(_ context.Context, id string, _ *tc.UpdateLibraryRequest, _ bool) (*tc.TransformationLibrary, error) {
						if id != "remote-lib-1" {
							return nil, fmt.Errorf("library %s not found", id)
						}
						return &tc.TransformationLibrary{ID: id, VersionID: "lib-1-v2"}, nil
					},
					BatchPublishFunc: func(_ context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
						log.record(req)
						return &tc.BatchPublishResponse{Published: true}, nil
					},
				}

				p := transformations.NewProviderWithStore(store)
				target := loadSpecs(t, p, tt.specs...)
				target.AddResource(resources.NewResource(
					"dest-1",
					linkedDestinationType,
					resources.ResourceData{},
					nil,
					resources.WithRawData(&linkedDestination{
						Transformation: handler.CreatePropertyRef(
							"transformation:trans-1",
							func(s *model.TransformationState) (string, error) { return s.ID, nil },
						),
					}),
				))

				lp := &linkingProvider{
					Provider: p,
					published: func(transformationID string) bool {
						return transformationID == "remote-trans-1" && log.has("trans-1-v1")
					},
				}
				s, err := syncer.New(lp, &client.Workspace{ID: "ws-1"},
					syncer.WithConcurrency(concurrency),
					syncer.WithReporter(testutils.NewMockReporter()),
				)
				require.NoError(t, err)

				require.NoError(t, s.Sync(context.Background(), target))

				assert.Equal(t, 1, log.calls, "the apply must publish once")
				assert.ElementsMatch(t, tt.wantPublished, log.versions)
			})
		}
	}
}

func TestPlannedPublish(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("a change that fails to stage fails every planned change and publishes nothing", func(t *testing.T) {
		t.Parallel()

		var published bool
		store := &testutil.MockTransformationStore{
			CreateLibraryFunc: func(context.Context, *tc.CreateLibraryRequest, bool) (*tc.TransformationLibrary, error) {
				return &tc.TransformationLibrary{ID: "remote-lib-1", VersionID: "lib-1-v1"}, nil
			},
			CreateTransformationFunc: func(context.Context, *tc.CreateTransformationRequest, bool) (*tc.Transformation, error) {
				return nil, errors.New("backend unavailable")
			},
			BatchPublishFunc: func(context.Context, *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
				published = true
				return &tc.BatchPublishResponse{Published: true}, nil
			},
		}

		p := transformations.NewProviderWithStore(store)
		graph := loadSpecs(t, p,
			librarySpec("lib-1", "mathLibrary", "export function add(a, b) { return a + b; }"),
			transformationSpec("trans-1", "export function transformEvent(event, metadata) { return event; }"),
		)
		lib := graphResource(t, graph, "transformation-library:lib-1")
		trans := graphResource(t, graph, "transformation:trans-1")
		require.NoError(t, p.CheckPlan(ctx, &planner.Plan{Operations: []*planner.Operation{
			{Type: planner.Create, Resource: lib},
			{Type: planner.Create, Resource: trans},
		}}, state.EmptyState()))

		_, libErr := p.CreateRaw(ctx, lib)
		_, transErr := p.CreateRaw(ctx, trans)

		require.ErrorContains(t, libErr, "backend unavailable")
		assert.Equal(t, libErr, transErr)
		assert.False(t, published)
	})

	t.Run("a rejected publish names what failed", func(t *testing.T) {
		t.Parallel()

		store := &testutil.MockTransformationStore{
			CreateLibraryFunc: func(context.Context, *tc.CreateLibraryRequest, bool) (*tc.TransformationLibrary, error) {
				return &tc.TransformationLibrary{ID: "remote-lib-1", VersionID: "lib-1-v1"}, nil
			},
			CreateTransformationFunc: func(context.Context, *tc.CreateTransformationRequest, bool) (*tc.Transformation, error) {
				return &tc.Transformation{ID: "remote-trans-1", VersionID: "trans-1-v1"}, nil
			},
			BatchPublishFunc: func(context.Context, *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
				return &tc.BatchPublishResponse{
					Published: false,
					ValidationOutput: tc.ValidationOutput{
						Libraries:       []tc.LibraryTestResult{{Name: "lib-1", VersionID: "lib-1-v1", Pass: true}},
						Transformations: []tc.TransformationTestResult{{Name: "trans-1", VersionID: "trans-1-v1", Pass: false}},
					},
				}, nil
			},
		}

		p := transformations.NewProviderWithStore(store)
		graph := loadSpecs(t, p,
			librarySpec("lib-1", "mathLibrary", "export function add(a, b) { return a + b; }"),
			transformationSpec("trans-1", "import mathLibrary from 'mathLibrary';\nexport function transformEvent(event, metadata) { return event; }"),
		)
		lib := graphResource(t, graph, "transformation-library:lib-1")
		trans := graphResource(t, graph, "transformation:trans-1")
		require.NoError(t, p.CheckPlan(ctx, &planner.Plan{Operations: []*planner.Operation{
			{Type: planner.Create, Resource: lib},
			{Type: planner.Create, Resource: trans},
		}}, state.EmptyState()))

		_, err := p.CreateRaw(ctx, trans)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `transformation "trans-1"`)
		assert.NotContains(t, err.Error(), `library "lib-1"`)
	})

	t.Run("planned deletes stay deferred to ConsolidateSync", func(t *testing.T) {
		t.Parallel()

		var (
			log     publishLog
			created []string
			deleted []string
		)
		store := &testutil.MockTransformationStore{
			CreateTransformationFunc: func(_ context.Context, req *tc.CreateTransformationRequest, _ bool) (*tc.Transformation, error) {
				created = append(created, req.ExternalID)
				return &tc.Transformation{ID: "remote-" + req.ExternalID, VersionID: req.ExternalID + "-v1"}, nil
			},
			BatchPublishFunc: func(_ context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
				log.record(req)
				return &tc.BatchPublishResponse{Published: true}, nil
			},
			DeleteTransformationFunc: func(_ context.Context, id string) error {
				deleted = append(deleted, id)
				return nil
			},
		}

		p := transformations.NewProviderWithStore(store)
		graph := loadSpecs(t, p, transformationSpec("trans-1", "export function transformEvent(event, metadata) { return event; }"))
		trans := graphResource(t, graph, "transformation:trans-1")
		var (
			removedData  = &model.TransformationResource{ID: "trans-2"}
			removedState = &model.TransformationState{ID: "remote-trans-2", VersionID: "trans-2-v1"}
			removed      = resources.NewResource("trans-2", transformation.HandlerMetadata.ResourceType,
				resources.ResourceData{}, nil, resources.WithRawData(removedData))
		)
		st := state.EmptyState()
		st.AddResource(&state.ResourceState{
			ID:        "trans-2",
			Type:      transformation.HandlerMetadata.ResourceType,
			InputRaw:  removedData,
			OutputRaw: removedState,
		})
		require.NoError(t, p.CheckPlan(ctx, &planner.Plan{Operations: []*planner.Operation{
			{Type: planner.Create, Resource: trans},
			{Type: planner.Delete, Resource: removed},
		}}, st))

		out, err := p.CreateRaw(ctx, trans)
		require.NoError(t, err)
		require.NoError(t, p.DeleteRaw(ctx, "trans-2", transformation.HandlerMetadata.ResourceType, removedData, removedState))
		st.AddResource(&state.ResourceState{ID: "trans-1", Type: transformation.HandlerMetadata.ResourceType, OutputRaw: out})
		st.RemoveResource("transformation:trans-2")
		require.NoError(t, p.ConsolidateSync(ctx, graph, st))

		assert.Equal(t, []string{"trans-1"}, created)
		assert.Equal(t, 1, log.calls)
		assert.Equal(t, []string{"trans-1-v1"}, log.versions)
		assert.Equal(t, []string{"remote-trans-2"}, deleted)
	})

	t.Run("an import is staged once and published", func(t *testing.T) {
		t.Parallel()

		var (
			log     publishLog
			updates int
		)
		store := &testutil.MockTransformationStore{
			GetTransformationFunc: func(_ context.Context, id string) (*tc.Transformation, error) {
				return &tc.Transformation{ID: id, VersionID: "trans-1-v1"}, nil
			},
			SetTransformationExternalIDFunc: func(context.Context, string, string) error {
				return nil
			},
			UpdateTransformationFunc: func(_ context.Context, id string, _ *tc.UpdateTransformationRequest, _ bool) (*tc.Transformation, error) {
				updates++
				return &tc.Transformation{ID: id, VersionID: "trans-1-v2"}, nil
			},
			BatchPublishFunc: func(_ context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
				log.record(req)
				return &tc.BatchPublishResponse{Published: true}, nil
			},
		}

		p := transformations.NewProviderWithStore(store)
		graph := loadSpecs(t, p, transformationSpec("trans-1", "export function transformEvent(event, metadata) { return event; }"))
		imported := resources.NewResource(
			"trans-1",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(graphResource(t, graph, "transformation:trans-1").RawData()),
			resources.WithResourceImportMetadata("remote-trans-1", "ws-1"),
		)
		require.NoError(t, p.CheckPlan(ctx, &planner.Plan{Operations: []*planner.Operation{
			{Type: planner.Import, Resource: imported},
		}}, state.EmptyState()))

		out, err := p.ImportRaw(ctx, imported, "remote-trans-1")

		require.NoError(t, err)
		assert.Equal(t, &model.TransformationState{ID: "remote-trans-1", VersionID: "trans-1-v2"}, out)
		assert.Equal(t, 1, updates)
		assert.Equal(t, []string{"trans-1-v2"}, log.versions)
	})
}
