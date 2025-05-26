package project_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

// MockLoader is a mock implementation of the project.Loader interface for testing.
type MockLoader struct {
	LoadFunc func(location string) (map[string]*specs.Spec, error)
}

// Load calls the mock LoadFunc.
func (m *MockLoader) Load(location string) (map[string]*specs.Spec, error) {
	if m.LoadFunc != nil {
		return m.LoadFunc(location)
	}
	return nil, errors.New("MockLoader.LoadFunc is not set")
}

func TestNewProject_Load_Error(t *testing.T) {
	provider := testutils.NewMockProvider()
	mockLoader := &MockLoader{}
	p := project.New("test_location", provider, project.WithLoader(mockLoader))

	assert.NotNil(t, p)
	mockLoader.LoadFunc = func(location string) (map[string]*specs.Spec, error) {
		assert.Equal(t, "test_location", location)
		return nil, errors.New("custom loader called")
	}
	err := p.Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom loader called")
}

func TestProject_Load_Success(t *testing.T) {
	mockProvider := testutils.NewMockProvider()
	mockLoader := &MockLoader{}

	proj := project.New("test_dir", mockProvider, project.WithLoader(mockLoader))

	expectedSpecs := map[string]*specs.Spec{
		"path/to/spec1.yaml": {Kind: "Source"},
		"path/to/spec2.yaml": {Kind: "Destination"},
	}

	mockLoader.LoadFunc = func(location string) (map[string]*specs.Spec, error) {
		return expectedSpecs, nil
	}

	mockProvider.SupportedKinds = []string{"Source", "Destination"}

	err := proj.Load()
	require.NoError(t, err)

	assert.Equal(t, 2, len(mockProvider.LoadSpecCalledWithArgs), "LoadSpec should be called for each spec")
	// Order might not be guaranteed from map iteration, so check presence
	foundSpec1 := false
	foundSpec2 := false
	for _, arg := range mockProvider.LoadSpecCalledWithArgs {
		if arg.Path == "path/to/spec1.yaml" && arg.Spec.Kind == "Source" {
			foundSpec1 = true
		}
		if arg.Path == "path/to/spec2.yaml" && arg.Spec.Kind == "Destination" {
			foundSpec2 = true
		}
	}
	assert.True(t, foundSpec1, "Spec1 should have been loaded")
	assert.True(t, foundSpec2, "Spec2 should have been loaded")

	assert.Equal(t, 1, mockProvider.ValidateCalledCount, "Validate should be called once")
}

func TestProject_Load_UnsupportedKind(t *testing.T) {
	mockProvider := testutils.NewMockProvider()
	mockLoader := &MockLoader{}

	proj := project.New("test_dir", mockProvider, project.WithLoader(mockLoader))

	specsWithUnsupportedKind := map[string]*specs.Spec{
		"path/to/spec.yaml": {Kind: "UnsupportedKind"},
	}

	mockLoader.LoadFunc = func(location string) (map[string]*specs.Spec, error) {
		return specsWithUnsupportedKind, nil
	}

	mockProvider.SupportedKinds = []string{"Source", "Destination"} // Does not include "UnsupportedKind"

	err := proj.Load()
	require.Error(t, err)
	var unsupportedKindErr specs.ErrUnsupportedKind
	require.True(t, errors.As(err, &unsupportedKindErr), "error should be of type specs.ErrUnsupportedKind")
	assert.Equal(t, "UnsupportedKind", unsupportedKindErr.Kind)
}

func TestProject_Load_ProviderLoadSpecError(t *testing.T) {
	mockProvider := testutils.NewMockProvider()
	mockLoader := &MockLoader{}

	proj := project.New("test_dir", mockProvider, project.WithLoader(mockLoader))

	validSpecs := map[string]*specs.Spec{
		"path/to/spec.yaml": {Kind: "Source"},
	}

	mockLoader.LoadFunc = func(location string) (map[string]*specs.Spec, error) {
		return validSpecs, nil
	}

	mockProvider.SupportedKinds = []string{"Source"}
	expectedErr := errors.New("provider LoadSpec failed")
	mockProvider.LoadSpecErr = expectedErr

	err := proj.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider failed to load spec from path path/to/spec.yaml")
	assert.True(t, errors.Is(err, expectedErr))
}

func TestProject_Load_ProviderValidateError(t *testing.T) {
	mockProvider := testutils.NewMockProvider()
	mockLoader := &MockLoader{}

	proj := project.New("test_dir", mockProvider, project.WithLoader(mockLoader))

	validSpecs := map[string]*specs.Spec{
		"path/to/spec.yaml": {Kind: "Source"},
	}

	mockLoader.LoadFunc = func(location string) (map[string]*specs.Spec, error) {
		return validSpecs, nil
	}

	mockProvider.SupportedKinds = []string{"Source"}
	mockProvider.LoadSpecErr = nil
	expectedErr := errors.New("provider Validate failed")
	mockProvider.ValidateErr = expectedErr

	err := proj.Load()
	require.Error(t, err)
	assert.True(t, errors.Is(err, expectedErr))
	assert.Equal(t, 1, mockProvider.ValidateCalledCount)
}

func TestProject_GetResourceGraph_Success(t *testing.T) {
	mockProvider := testutils.NewMockProvider()
	proj := project.New("test_dir", mockProvider) // Loader doesn't matter for this test

	expectedGraph := &resources.Graph{}
	mockProvider.GetResourceGraphVal = expectedGraph
	mockProvider.GetResourceGraphErr = nil

	graph, err := proj.GetResourceGraph()
	require.NoError(t, err)
	assert.Same(t, expectedGraph, graph) // Check if it's the exact same instance
	assert.Equal(t, 1, mockProvider.GetResourceGraphCalledCount)
}

func TestProject_GetResourceGraph_Error(t *testing.T) {
	mockProvider := testutils.NewMockProvider()
	proj := project.New("test_dir", mockProvider)

	expectedErr := errors.New("GetResourceGraph failed")
	mockProvider.GetResourceGraphVal = nil
	mockProvider.GetResourceGraphErr = expectedErr

	graph, err := proj.GetResourceGraph()
	require.Error(t, err)
	assert.Nil(t, graph)
	assert.True(t, errors.Is(err, expectedErr))
	assert.Equal(t, 1, mockProvider.GetResourceGraphCalledCount)
}
