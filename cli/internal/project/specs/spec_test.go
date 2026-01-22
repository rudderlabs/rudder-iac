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
		name                string
		yamlData            string
		expectValidateError bool
		expectNewError      bool
		errorContains       string
		expectedKind        string
		expectedName        string // Extracted from metadata.name
	}{
		{
			name:                "Valid Spec",
			yamlData:            validYAML,
			expectValidateError: false,
			expectNewError:      false,
			expectedKind:        "Destination",
			expectedName:        "MyTestDest",
		},
		{
			name:                "Missing Version",
			yamlData:            missingVersionYAML,
			expectValidateError: true,
			expectNewError:      false,
			errorContains:       "missing required field 'version'",
		},
		{
			name:                "Missing Kind",
			yamlData:            missingKindYAML,
			expectValidateError: true,
			expectNewError:      false,
			errorContains:       "missing required field 'kind'",
		},
		{
			name:                "Missing Metadata",
			yamlData:            missingMetadataYAML,
			expectValidateError: true,
			expectNewError:      false,
			errorContains:       "missing required field 'metadata'",
		},
		{
			name:                "Missing Spec field",
			yamlData:            missingSpecFieldYAML,
			expectValidateError: true,
			expectNewError:      false,
			errorContains:       "missing required field 'spec'",
		},
		{
			name:                "Invalid YAML syntax",
			yamlData:            invalidYAML,
			expectValidateError: false,
			expectNewError:      true,
			errorContains:       "unmarshaling yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := New([]byte(tc.yamlData))
			if tc.expectNewError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				require.Nil(t, spec)

				return
			} else {
				require.NoError(t, err)
				require.NotNil(t, spec)
			}

			err = spec.Validate()
			if tc.expectValidateError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
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
