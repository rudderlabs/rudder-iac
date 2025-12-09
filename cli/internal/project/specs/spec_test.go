package specs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	validYAML := `
version: rudder/v0.1
kind: Destination
metadata:
  name: MyTestDest
  labels:
    env: prod
spec:
  type: S3
  config:
    bucket: "my-bucket"
    region: "us-west-2"
`

	missingVersionYAML := `
kind: Source
metadata:
  name: MyTestSource
spec:
  type: javascript
`

	missingKindYAML := `
version: rudder/v0.1
metadata:
  name: MyTestSource
spec:
  type: javascript
`

	missingMetadataYAML := `
version: rudder/v0.1
kind: Source
spec:
  type: javascript
`

	missingSpecFieldYAML := `
version: rudder/v0.1
kind: Source
metadata:
  name: MyTestSource
`

	invalidYAML := `
version: rudder/v0.1
kind: Source
  metadata: # Invalid indentation
    name: MyTestSource
spec:
  type: javascript
`

	testCases := []struct {
		name          string
		yamlData      string
		expectError   bool
		errorContains string
		expectedKind  string
		expectedName  string // Extracted from metadata.name
	}{
		{
			name:         "Valid Spec",
			yamlData:     validYAML,
			expectError:  false,
			expectedKind: "Destination",
			expectedName: "MyTestDest",
		},
		{
			name:          "Missing Version",
			yamlData:      missingVersionYAML,
			expectError:   true,
			errorContains: "missing required field 'version'",
		},
		{
			name:          "Missing Kind",
			yamlData:      missingKindYAML,
			expectError:   true,
			errorContains: "missing required field 'kind'",
		},
		{
			name:          "Missing Metadata",
			yamlData:      missingMetadataYAML,
			expectError:   true,
			errorContains: "missing required field 'metadata'",
		},
		{
			name:          "Missing Spec field",
			yamlData:      missingSpecFieldYAML,
			expectError:   true,
			errorContains: "missing required field 'spec'",
		},
		{
			name:          "Invalid YAML syntax",
			yamlData:      invalidYAML,
			expectError:   true,
			errorContains: "unmarshaling yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := New([]byte(tc.yamlData))

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, spec)
			} else {
				require.NoError(t, err)
				require.NotNil(t, spec)
				assert.Equal(t, tc.expectedKind, spec.Kind)
				require.NotNil(t, spec.Metadata)
				name, ok := spec.Metadata["name"].(string)
				require.True(t, ok, "metadata.name should be a string")
				assert.Equal(t, tc.expectedName, name)
				// Check other required fields are present
				assert.NotEmpty(t, spec.Version)
				assert.NotNil(t, spec.Spec)
			}
		})
	}
}

func TestNew_StrictValidation(t *testing.T) {
	t.Run("Rejects unknown field at top level", func(t *testing.T) {
		yamlWithUnknownField := `
version: rudder/v0.1
kind: datacatalog
metadata:
  name: TestSpec
spec:
  id: test-id
unknown_field: "this should fail"
`
		spec, err := New([]byte(yamlWithUnknownField))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_field")
		assert.Nil(t, spec)
	})

	t.Run("Accepts valid spec without unknown fields", func(t *testing.T) {
		validYAML := `
version: rudder/v0.1
kind: datacatalog
metadata:
  name: TestSpec
spec:
  id: test-id
  description: "valid spec"
`
		spec, err := New([]byte(validYAML))
		require.NoError(t, err)
		require.NotNil(t, spec)
		assert.Equal(t, "datacatalog", spec.Kind)
	})
}
