package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/registry"
)

// MockLoader is a mock implementation of project.Loader
type MockLoader struct {
	mock.Mock
}

func (m *MockLoader) Load(location string) (map[string]*specs.Spec, error) {
	args := m.Called(location)
	return args.Get(0).(map[string]*specs.Spec), args.Error(1)
}

// MockSpecLoader is a mock implementation of provider.SpecLoader
type MockSpecLoader struct {
	mock.Mock
}

func (m *MockSpecLoader) LoadSpec(path string, s *specs.Spec) error {
	args := m.Called(path, s)
	return args.Error(0)
}

func (m *MockSpecLoader) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	args := m.Called(path, s)
	return args.Get(0).(*specs.ParsedSpec), args.Error(1)
}

func (m *MockSpecLoader) ResourceGraph() (*resources.Graph, error) {
	args := m.Called()
	return args.Get(0).(*resources.Graph), args.Error(1)
}

// MockRule is a mock implementation of validation.Rule
type MockRule struct {
	mock.Mock
}

func (m *MockRule) ID() string {
	return m.Called().String(0)
}

func (m *MockRule) Validate(ctx *validation.ValidationContext, graph *resources.Graph) []validation.ValidationError {
	args := m.Called(ctx, graph)
	return args.Get(0).([]validation.ValidationError)
}

func (m *MockRule) Severity() validation.Severity {
	return m.Called().Get(0).(validation.Severity)
}

func (m *MockRule) Description() string {
	return m.Called().String(0)
}

func (m *MockRule) Examples() [][]byte {
	return m.Called().Get(0).([][]byte)
}

func (m *MockRule) AppliesTo() []string {
	return m.Called().Get(0).([]string)
}

func TestEngine_Validate(t *testing.T) {
	// Create a temporary file to simulate a real spec file for PathIndex
	tmpDir, err := os.MkdirTemp("", "engine-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specPath := filepath.Join(tmpDir, "spec.yaml")
	specContent := `version: rudder/v0.1
kind: test-kind
metadata:
  name: test-resource
spec:
  field: value
`
	err = os.WriteFile(specPath, []byte(specContent), 0644)
	assert.NoError(t, err)

	spec := &specs.Spec{
		Version: "rudder/v0.1",
		Kind:    "test-kind",
		Metadata: map[string]any{
			"name": "test-resource",
		},
		Spec: map[string]any{
			"field": "value",
		},
	}

	graph := resources.NewGraph()

	t.Run("Happy Path - No Errors", func(t *testing.T) {
		mockLoader := new(MockLoader)
		mockProvider := new(MockSpecLoader)
		reg := registry.NewRegistry()

		mockLoader.On("Load", "project").Return(map[string]*specs.Spec{
			specPath: spec,
		}, nil)

		mockProvider.On("LoadSpec", specPath, spec).Return(nil)
		mockProvider.On("ResourceGraph").Return(graph, nil)

		rule := new(MockRule)
		rule.On("AppliesTo").Return([]string{"test-kind"})
		rule.On("Validate", mock.Anything, graph).Return([]validation.ValidationError{})

		err := reg.Register(rule)
		assert.NoError(t, err)

		engine, err := NewEngine("project", reg, mockProvider, WithLoader(mockLoader))
		require.NoError(t, err)
		diagnostics := engine.Validate()

		assert.Empty(t, diagnostics)
		mockLoader.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
		rule.AssertExpectations(t)
	})

	t.Run("Validation Error", func(t *testing.T) {
		mockLoader := new(MockLoader)
		mockProvider := new(MockSpecLoader)
		reg := registry.NewRegistry()

		mockLoader.On("Load", "project").Return(map[string]*specs.Spec{
			specPath: spec,
		}, nil)

		mockProvider.On("LoadSpec", specPath, spec).Return(nil)
		mockProvider.On("ResourceGraph").Return(graph, nil)

		rule := new(MockRule)
		rule.On("ID").Return("test-rule")
		rule.On("AppliesTo").Return([]string{"test-kind"})
		rule.On("Severity").Return(validation.SeverityError)
		rule.On("Validate", mock.Anything, graph).Return([]validation.ValidationError{
			{
				Msg:      "test error",
				Fragment: "field",
				Pos:      location.Position{Line: 5, Column: 3},
			},
		})

		err := reg.Register(rule)
		assert.NoError(t, err)

		engine, err := NewEngine("project", reg, mockProvider, WithLoader(mockLoader))
		require.NoError(t, err)
		diagnostics := engine.Validate()

		assert.Len(t, diagnostics, 1)
		assert.Equal(t, "test-rule", diagnostics[0].Rule)
		assert.Equal(t, "test error", diagnostics[0].Message)
		assert.Equal(t, specPath, diagnostics[0].File)
		assert.Equal(t, 5, diagnostics[0].Position.Line)
	})

	t.Run("Parse/Load Error", func(t *testing.T) {
		mockLoader := new(MockLoader)
		mockProvider := new(MockSpecLoader)
		reg := registry.NewRegistry()

		mockLoader.On("Load", "project").Return(map[string]*specs.Spec{
			"invalid.yaml": spec,
		}, nil)

		mockProvider.On("LoadSpec", "invalid.yaml", spec).Return(fmt.Errorf("parse error"))
		mockProvider.On("ResourceGraph").Return(graph, nil)

		engine, err := NewEngine("project", reg, mockProvider, WithLoader(mockLoader))
		require.NoError(t, err)
		diagnostics := engine.Validate()

		assert.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "failed to load spec into provider")
		assert.Equal(t, "invalid.yaml", diagnostics[0].File)
	})
}
