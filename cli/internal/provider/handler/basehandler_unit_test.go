package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type unitSpec struct{}

type unitRes struct {
	ID string
}

type unitState struct {
	ID string
}

type unitRemote struct {
	metadata RemoteResourceMetadata
}

func (r unitRemote) Metadata() RemoteResourceMetadata {
	return r.metadata
}

type unitImpl struct {
	createFn          func(context.Context, *unitRes) (*unitState, error)
	updateFn          func(context.Context, *unitRes, *unitRes, *unitState) (*unitState, error)
	deleteFn          func(context.Context, string, *unitRes, *unitState) error
	importFn          func(context.Context, *unitRes, string) (*unitState, error)
	mapRemoteToState  func(*unitRemote, URNResolver) (*unitRes, *unitState, error)
	formatForExportFn func(map[string]*unitRemote, namer.Namer, resolver.ReferenceResolver) ([]writer.FormattableEntity, error)
}

func (m *unitImpl) Metadata() HandlerMetadata {
	return HandlerMetadata{SpecKind: "unit", ResourceType: "unit-resource", SpecMetadataName: "common"}
}

func (m *unitImpl) NewSpec() *unitSpec {
	return &unitSpec{}
}

func (m *unitImpl) ExtractResourcesFromSpec(_ string, _ *unitSpec) (map[string]*unitRes, error) {
	return map[string]*unitRes{}, nil
}

func (m *unitImpl) LoadRemoteResources(_ context.Context) ([]*unitRemote, error) {
	return nil, nil
}

func (m *unitImpl) LoadImportableResources(_ context.Context) ([]*unitRemote, error) {
	return nil, nil
}

func (m *unitImpl) MapRemoteToState(remote *unitRemote, urnResolver URNResolver) (*unitRes, *unitState, error) {
	if m.mapRemoteToState != nil {
		return m.mapRemoteToState(remote, urnResolver)
	}
	return &unitRes{ID: remote.metadata.ExternalID}, &unitState{ID: remote.metadata.ID}, nil
}

func (m *unitImpl) Create(ctx context.Context, data *unitRes) (*unitState, error) {
	if m.createFn != nil {
		return m.createFn(ctx, data)
	}
	return &unitState{ID: data.ID}, nil
}

func (m *unitImpl) Update(ctx context.Context, newData *unitRes, oldData *unitRes, oldState *unitState) (*unitState, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, newData, oldData, oldState)
	}
	return &unitState{ID: newData.ID}, nil
}

func (m *unitImpl) Import(ctx context.Context, data *unitRes, remoteID string) (*unitState, error) {
	if m.importFn != nil {
		return m.importFn(ctx, data, remoteID)
	}
	return &unitState{ID: remoteID}, nil
}

func (m *unitImpl) Delete(ctx context.Context, id string, oldData *unitRes, oldState *unitState) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, oldData, oldState)
	}
	return nil
}

func (m *unitImpl) FormatForExport(collection map[string]*unitRemote, idNamer namer.Namer, inputResolver resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
	if m.formatForExportFn != nil {
		return m.formatForExportFn(collection, idNamer, inputResolver)
	}
	return nil, nil
}

func TestBaseHandler_CreateUpdateDeleteImport(t *testing.T) {
	t.Parallel()

	h := NewHandler[unitSpec, unitRes, unitState, unitRemote](&unitImpl{})
	ctx := context.Background()

	_, err := h.Create(ctx, "invalid")
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidDataType{}, err)

	createOut, err := h.Create(ctx, &unitRes{ID: "new"})
	require.NoError(t, err)
	assert.Equal(t, &unitState{ID: "new"}, createOut)

	_, err = h.Update(ctx, "bad", &unitRes{}, &unitState{})
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidDataType{}, err)

	_, err = h.Update(ctx, &unitRes{}, "bad", &unitState{})
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidDataType{}, err)

	_, err = h.Update(ctx, &unitRes{}, &unitRes{}, "bad")
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidDataType{}, err)

	updateOut, err := h.Update(ctx, &unitRes{ID: "u"}, &unitRes{ID: "old"}, &unitState{ID: "state"})
	require.NoError(t, err)
	assert.Equal(t, &unitState{ID: "u"}, updateOut)

	err = h.Delete(ctx, "id", "bad", &unitState{})
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidDataType{}, err)

	err = h.Delete(ctx, "id", &unitRes{}, "bad")
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidDataType{}, err)

	err = h.Delete(ctx, "id", &unitRes{ID: "old"}, &unitState{ID: "state"})
	require.NoError(t, err)

	_, err = h.Import(ctx, "bad", "remote-1")
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidDataType{}, err)

	importOut, err := h.Import(ctx, &unitRes{ID: "in"}, "remote-1")
	require.NoError(t, err)
	assert.Equal(t, &unitState{ID: "remote-1"}, importOut)
}

func TestBaseHandler_MapRemoteToState(t *testing.T) {
	t.Parallel()

	t.Run("returns invalid data type for wrong remote payload", func(t *testing.T) {
		h := NewHandler[unitSpec, unitRes, unitState, unitRemote](&unitImpl{})
		collection := resources.NewRemoteResources()
		collection.Set("unit-resource", map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: "wrong"},
		})

		_, err := h.MapRemoteToState(collection)
		require.Error(t, err)
		assert.IsType(t, &ErrInvalidDataType{}, err)
	})

	t.Run("propagates impl errors", func(t *testing.T) {
		expectedErr := errors.New("map failed")
		h := NewHandler[unitSpec, unitRes, unitState, unitRemote](&unitImpl{
			mapRemoteToState: func(_ *unitRemote, _ URNResolver) (*unitRes, *unitState, error) {
				return nil, nil, expectedErr
			},
		})

		remote := &unitRemote{metadata: RemoteResourceMetadata{ID: "r1", ExternalID: "ext-1"}}
		collection := resources.NewRemoteResources()
		collection.Set("unit-resource", map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: remote},
		})

		_, err := h.MapRemoteToState(collection)
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("skips nil input resources", func(t *testing.T) {
		h := NewHandler[unitSpec, unitRes, unitState, unitRemote](&unitImpl{
			mapRemoteToState: func(_ *unitRemote, _ URNResolver) (*unitRes, *unitState, error) {
				return nil, nil, nil
			},
		})

		remote := &unitRemote{metadata: RemoteResourceMetadata{ID: "r1", ExternalID: "ext-1"}}
		collection := resources.NewRemoteResources()
		collection.Set("unit-resource", map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: remote},
		})

		s, err := h.MapRemoteToState(collection)
		require.NoError(t, err)
		assert.Empty(t, s.Resources)
	})

	t.Run("maps resources to state", func(t *testing.T) {
		h := NewHandler[unitSpec, unitRes, unitState, unitRemote](&unitImpl{})
		remote := &unitRemote{metadata: RemoteResourceMetadata{ID: "r1", ExternalID: "ext-1"}}

		collection := resources.NewRemoteResources()
		collection.Set("unit-resource", map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: remote},
		})

		s, err := h.MapRemoteToState(collection)
		require.NoError(t, err)
		res := s.GetResource(resources.URN("ext-1", "unit-resource"))
		require.NotNil(t, res)
		assert.Equal(t, &unitRes{ID: "ext-1"}, res.InputRaw)
		assert.Equal(t, &unitState{ID: "r1"}, res.OutputRaw)
	})
}

func TestBaseHandler_FormatForExport(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for empty collection", func(t *testing.T) {
		h := NewHandler[unitSpec, unitRes, unitState, unitRemote](&unitImpl{})
		collection := resources.NewRemoteResources()

		entities, err := h.FormatForExport(collection, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, entities)
	})

	t.Run("returns invalid type when remote payload is wrong", func(t *testing.T) {
		h := NewHandler[unitSpec, unitRes, unitState, unitRemote](&unitImpl{})
		collection := resources.NewRemoteResources()
		collection.Set("unit-resource", map[string]*resources.RemoteResource{
			"r1": {ID: "r1", ExternalID: "ext-1", Data: 10},
		})

		_, err := h.FormatForExport(collection, nil, nil)
		require.Error(t, err)
		assert.IsType(t, &ErrInvalidDataType{}, err)
	})

	t.Run("passes typed map keyed by external id", func(t *testing.T) {
		var got map[string]*unitRemote
		h := NewHandler[unitSpec, unitRes, unitState, unitRemote](&unitImpl{
			formatForExportFn: func(collection map[string]*unitRemote, _ namer.Namer, _ resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
				got = collection
				return []writer.FormattableEntity{{RelativePath: "out.yaml", Content: map[string]any{"ok": true}}}, nil
			},
		})

		remote := &unitRemote{metadata: RemoteResourceMetadata{ID: "r1", ExternalID: "ext-1"}}
		collection := resources.NewRemoteResources()
		collection.Set("unit-resource", map[string]*resources.RemoteResource{
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

	ref := CreatePropertyRef[unitState]("unit-resource:id", func(s *unitState) (string, error) {
		return s.ID, nil
	})

	value, err := ref.Resolve(&unitState{ID: "ok"})
	require.NoError(t, err)
	assert.Equal(t, "ok", value)

	_, err = ref.Resolve("bad")
	require.Error(t, err)
	assert.IsType(t, &ErrInvalidDataType{}, err)
}

func TestErrInvalidDataType_Error(t *testing.T) {
	t.Parallel()

	err := (&ErrInvalidDataType{Expected: (*unitState)(nil), Actual: "bad"}).Error()
	assert.Contains(t, err, "invalid resource data type")
	assert.Contains(t, err, "string")
	assert.Contains(t, err, "*handler.unitState")
}
