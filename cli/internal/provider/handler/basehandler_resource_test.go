package handler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testResourceType = "test-resource"

type testResourceSpec struct{}

type testResource struct {
	ID string
}

type testResourceState struct {
	ID string
}

type testResourceRemote struct {
	metadata handler.RemoteResourceMetadata
}

func (r testResourceRemote) Metadata() handler.RemoteResourceMetadata {
	return r.metadata
}

type testResourceHandler struct {
	createFn          func(context.Context, *testResource) (*testResourceState, error)
	updateFn          func(context.Context, *testResource, *testResource, *testResourceState) (*testResourceState, error)
	deleteFn          func(context.Context, string, *testResource, *testResourceState) error
	importFn          func(context.Context, *testResource, string) (*testResourceState, error)
	mapRemoteToState  func(*testResourceRemote, handler.URNResolver) (*testResource, *testResourceState, error)
	formatForExportFn func(map[string]*testResourceRemote, namer.Namer, resolver.ReferenceResolver) ([]writer.FormattableEntity, error)
}

func (m *testResourceHandler) Metadata() handler.HandlerMetadata {
	return handler.HandlerMetadata{SpecKind: "test-resource", ResourceType: testResourceType, SpecMetadataName: "common"}
}

func (m *testResourceHandler) NewSpec() *testResourceSpec {
	return &testResourceSpec{}
}

func (m *testResourceHandler) ExtractResourcesFromSpec(_ string, _ *testResourceSpec) (map[string]*testResource, error) {
	return map[string]*testResource{}, nil
}

func (m *testResourceHandler) LoadRemoteResources(_ context.Context) ([]*testResourceRemote, error) {
	return nil, nil
}

func (m *testResourceHandler) LoadImportableResources(_ context.Context) ([]*testResourceRemote, error) {
	return nil, nil
}

func (m *testResourceHandler) MapRemoteToState(remote *testResourceRemote, urnResolver handler.URNResolver) (*testResource, *testResourceState, error) {
	if m.mapRemoteToState != nil {
		return m.mapRemoteToState(remote, urnResolver)
	}
	return &testResource{ID: remote.metadata.ExternalID}, &testResourceState{ID: remote.metadata.ID}, nil
}

func (m *testResourceHandler) Create(ctx context.Context, data *testResource) (*testResourceState, error) {
	if m.createFn != nil {
		return m.createFn(ctx, data)
	}
	return &testResourceState{ID: data.ID}, nil
}

func (m *testResourceHandler) Update(ctx context.Context, newData *testResource, oldData *testResource, oldState *testResourceState) (*testResourceState, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, newData, oldData, oldState)
	}
	return &testResourceState{ID: newData.ID}, nil
}

func (m *testResourceHandler) Import(ctx context.Context, data *testResource, remoteID string) (*testResourceState, error) {
	if m.importFn != nil {
		return m.importFn(ctx, data, remoteID)
	}
	return &testResourceState{ID: remoteID}, nil
}

func (m *testResourceHandler) Delete(ctx context.Context, id string, oldData *testResource, oldState *testResourceState) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, oldData, oldState)
	}
	return nil
}

func (m *testResourceHandler) FormatForExport(collection map[string]*testResourceRemote, idNamer namer.Namer, inputResolver resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
	if m.formatForExportFn != nil {
		return m.formatForExportFn(collection, idNamer, inputResolver)
	}
	return nil, nil
}

func TestBaseHandler_Lifecycle_Operations(t *testing.T) {
	t.Parallel()

	h := handler.NewHandler(&testResourceHandler{})
	ctx := context.Background()

	_, err := h.Create(ctx, "invalid")
	require.Error(t, err)
	assert.IsType(t, &handler.ErrInvalidDataType{}, err)

	createOut, err := h.Create(ctx, &testResource{ID: "new"})
	require.NoError(t, err)
	assert.Equal(t, &testResourceState{ID: "new"}, createOut)

	_, err = h.Update(ctx, "bad", &testResource{}, &testResourceState{})
	require.Error(t, err)
	assert.IsType(t, &handler.ErrInvalidDataType{}, err)

	_, err = h.Update(ctx, &testResource{}, "bad", &testResourceState{})
	require.Error(t, err)
	assert.IsType(t, &handler.ErrInvalidDataType{}, err)

	_, err = h.Update(ctx, &testResource{}, &testResource{}, "bad")
	require.Error(t, err)
	assert.IsType(t, &handler.ErrInvalidDataType{}, err)

	updateOut, err := h.Update(ctx, &testResource{ID: "u"}, &testResource{ID: "old"}, &testResourceState{ID: "state"})
	require.NoError(t, err)
	assert.Equal(t, &testResourceState{ID: "u"}, updateOut)

	err = h.Delete(ctx, "id", "bad", &testResourceState{})
	require.Error(t, err)
	assert.IsType(t, &handler.ErrInvalidDataType{}, err)

	err = h.Delete(ctx, "id", &testResource{}, "bad")
	require.Error(t, err)
	assert.IsType(t, &handler.ErrInvalidDataType{}, err)

	err = h.Delete(ctx, "id", &testResource{ID: "old"}, &testResourceState{ID: "state"})
	require.NoError(t, err)

	_, err = h.Import(ctx, "bad", "remote-1")
	require.Error(t, err)
	assert.IsType(t, &handler.ErrInvalidDataType{}, err)

	importOut, err := h.Import(ctx, &testResource{ID: "in"}, "remote-1")
	require.NoError(t, err)
	assert.Equal(t, &testResourceState{ID: "remote-1"}, importOut)
}

func TestBaseHandler_MapRemoteToState(t *testing.T) {
	t.Parallel()

	t.Run("returns invalid data type for wrong remote payload", func(t *testing.T) {
		h := handler.NewHandler(&testResourceHandler{})
		collection := resources.NewRemoteResources()
		collection.Set(testResourceType, map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: "wrong"},
		})

		_, err := h.MapRemoteToState(collection)
		require.Error(t, err)
		assert.IsType(t, &handler.ErrInvalidDataType{}, err)
	})

	t.Run("propagates impl errors", func(t *testing.T) {
		expectedErr := errors.New("map failed")
		h := handler.NewHandler(&testResourceHandler{
			mapRemoteToState: func(_ *testResourceRemote, _ handler.URNResolver) (*testResource, *testResourceState, error) {
				return nil, nil, expectedErr
			},
		})

		remote := &testResourceRemote{metadata: handler.RemoteResourceMetadata{ID: "r1", ExternalID: "ext-1"}}
		collection := resources.NewRemoteResources()
		collection.Set(testResourceType, map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: remote},
		})

		_, err := h.MapRemoteToState(collection)
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("skips nil input resources", func(t *testing.T) {
		h := handler.NewHandler(&testResourceHandler{
			mapRemoteToState: func(_ *testResourceRemote, _ handler.URNResolver) (*testResource, *testResourceState, error) {
				return nil, nil, nil
			},
		})

		remote := &testResourceRemote{metadata: handler.RemoteResourceMetadata{ID: "r1", ExternalID: "ext-1"}}
		collection := resources.NewRemoteResources()
		collection.Set(testResourceType, map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: remote},
		})

		s, err := h.MapRemoteToState(collection)
		require.NoError(t, err)
		assert.Empty(t, s.Resources)
	})

	t.Run("maps resources to state", func(t *testing.T) {
		h := handler.NewHandler(&testResourceHandler{})
		remote := &testResourceRemote{metadata: handler.RemoteResourceMetadata{ID: "r1", ExternalID: "ext-1"}}

		collection := resources.NewRemoteResources()
		collection.Set(testResourceType, map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: remote},
		})

		s, err := h.MapRemoteToState(collection)
		require.NoError(t, err)
		res := s.GetResource(resources.URN("ext-1", testResourceType))
		require.NotNil(t, res)
		assert.Equal(t, &testResource{ID: "ext-1"}, res.InputRaw)
		assert.Equal(t, &testResourceState{ID: "r1"}, res.OutputRaw)
	})
}

func TestBaseHandler_FormatForExport(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for empty collection", func(t *testing.T) {
		h := handler.NewHandler(&testResourceHandler{})
		collection := resources.NewRemoteResources()

		entities, err := h.FormatForExport(collection, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, entities)
	})

	t.Run("returns invalid type when remote payload is wrong", func(t *testing.T) {
		h := handler.NewHandler(&testResourceHandler{})
		collection := resources.NewRemoteResources()
		collection.Set(testResourceType, map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: 10},
		})

		_, err := h.FormatForExport(collection, nil, nil)
		require.Error(t, err)
		assert.IsType(t, &handler.ErrInvalidDataType{}, err)
	})

	t.Run("passes typed map keyed by external id", func(t *testing.T) {
		var got map[string]*testResourceRemote
		h := handler.NewHandler(&testResourceHandler{
			formatForExportFn: func(collection map[string]*testResourceRemote, _ namer.Namer, _ resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
				got = collection
				return []writer.FormattableEntity{{RelativePath: "out.yaml", Content: map[string]any{"ok": true}}}, nil
			},
		})

		remote := &testResourceRemote{metadata: handler.RemoteResourceMetadata{ID: "r1", ExternalID: "ext-1"}}
		collection := resources.NewRemoteResources()
		collection.Set(testResourceType, map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: remote},
		})

		entities, err := h.FormatForExport(collection, nil, nil)
		require.NoError(t, err)
		require.Len(t, entities, 1)
		assert.Equal(t, "out.yaml", entities[0].RelativePath)
		require.Len(t, got, 1)
		assert.Contains(t, got, "ext-1")
	})
}

func TestCreatePropertyRef(t *testing.T) {
	t.Parallel()

	ref := handler.CreatePropertyRef(testResourceType+":id", func(s *testResourceState) (string, error) {
		return s.ID, nil
	})

	value, err := ref.Resolve(&testResourceState{ID: "ok"})
	require.NoError(t, err)
	assert.Equal(t, "ok", value)

	_, err = ref.Resolve("bad")
	require.Error(t, err)
	assert.IsType(t, &handler.ErrInvalidDataType{}, err)
}

func TestErrInvalidDataType_Error(t *testing.T) {
	t.Parallel()

	err := (&handler.ErrInvalidDataType{Expected: (*testResourceState)(nil), Actual: "bad"}).Error()
	assert.Contains(t, err, "invalid resource data type")
	assert.Contains(t, err, "string")
	assert.Contains(t, err, "*handler_test.TestResourceState")
}
