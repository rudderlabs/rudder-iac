package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load(t *testing.T) {
	// Helper to create a temporary test directory with files
	setupTestDir := func(t *testing.T, files map[string]string) string {
		t.Helper()
		tmpDir, err := os.MkdirTemp("", "loader_test_")
		require.NoError(t, err)

		for name, content := range files {
			filePath := filepath.Join(tmpDir, name)
			err := os.MkdirAll(filepath.Dir(filePath), 0755)
			require.NoError(t, err)
			err = os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}
		return tmpDir
	}

	// Valid spec content
	validSpecContent1 := `
version: rudder/0.1
kind: source
metadata:
  name: MyTestSource1
spec:
  type: javascript
`
	validSpecContent2 := `
version: rudder/0.1
kind: destination
metadata:
  name: MyTestDestination1
spec:
  type: S3
  config:
    bucket: "my-bucket"
`
	// Invalid YAML content (e.g. unclosed sequence)
	invalidYAMLContent := `
version: rudder/0.1
kind: source
metadata:
  name: MyTestSourceInvalidYAML
spec:
  type: [
    "item1"
  # Missing closing bracket
`
	// Valid YAML, but invalid spec structure (e.g. missing 'kind')
	// specs.New checks for version, kind, metadata, spec top-level keys.
	// Let's make one that's valid YAML but would fail specs.New validation.
	invalidSpecStructureContent := `
version: rudder/0.1
# kind: MissingKind
spec:
  type: SomeType
`

	testCases := []struct {
		name          string
		files         map[string]string // relative path to content
		expectedSpecs int
		expectError   bool
		errorContains string // substring to check in error message
	}{
		{
			name: "Valid Project (Comprehensive)",
			files: map[string]string{
				"source1.yaml":          validSpecContent1,
				"destination1.yml":      validSpecContent2,
				"subdir/source2.yaml":   validSpecContent1,
				"subdir/deep/dest2.yml": validSpecContent2,
			},
			expectedSpecs: 4,
			expectError:   false,
		},
		{
			name:          "Empty Directory",
			files:         map[string]string{},
			expectedSpecs: 0,
			expectError:   false,
		},
		{
			name: "Directory with No Spec Files (Other Extensions Only)",
			files: map[string]string{
				"notes.txt": "some notes",
				"image.png": "binarydata",
				"README.md": "read me",
			},
			expectedSpecs: 0,
			expectError:   false,
		},
		{
			name: "Directory with Invalid YAML Syntax",
			files: map[string]string{
				"invalid.yaml": invalidYAMLContent,
			},
			expectedSpecs: 0,
			expectError:   true,
			errorContains: "parsing spec file",
		},
		{
			name: "Directory with Invalid Spec Structure",
			files: map[string]string{
				"invalid_structure.yaml": invalidSpecStructureContent,
			},
			expectedSpecs: 0,
			expectError:   true,
			errorContains: "parsing spec file", // specs.New should return an error
		},
		{
			name: "Mixed Valid and Invalid Files (Error Halts Processing)",
			files: map[string]string{
				"source1.yaml": validSpecContent1,
				"invalid.yml":  invalidYAMLContent, // This should cause the error
				"dest1.yaml":   validSpecContent2,  // This might not be processed
			},
			expectedSpecs: 0, // Or 1 if source1.yaml is processed before error, but WalkDir usually stops
			expectError:   true,
			errorContains: "parsing spec file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t, tc.files)
			defer os.RemoveAll(tmpDir)

			l := &loader.Loader{}
			loadedSpecs, err := l.Load(tmpDir)

			if tc.expectError {
				require.Error(t, err, "Expected an error for test case: %s", tc.name)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains, "Error message mismatch for test case: %s", tc.name)
				}
			} else {
				require.NoError(t, err, "Did not expect an error for test case: %s", tc.name)
				assert.Len(t, loadedSpecs, tc.expectedSpecs, "Number of loaded specs mismatch for test case: %s", tc.name)
				if tc.expectedSpecs > 0 {
					for path, spec := range loadedSpecs {
						assert.NotNil(t, spec, "Loaded spec should not be nil for path: %s in test case: %s", path, tc.name)
						assert.NotEmpty(t, spec.Kind, "Spec kind should not be empty for path: %s in test case: %s", path, tc.name)
						assert.NotNil(t, spec.Metadata, "Spec metadata should not be nil for path: %s in test case: %s", path, tc.name)
						name, ok := spec.Metadata["name"].(string)
						assert.True(t, ok, "Spec metadata.name should be a string for path: %s in test case: %s", path, tc.name)
						assert.NotEmpty(t, name, "Spec metadata.name should not be empty for path: %s in test case: %s", path, tc.name)
						assert.True(t, filepath.IsAbs(path), "Loaded spec path should be absolute: %s in test case: %s", path, tc.name)
						assert.Contains(t, path, tmpDir, "Loaded spec path should be within the temp directory: %s in test case: %s", path, tc.name)
					}
				}
			}
		})
	}

	t.Run("Non-existent Location", func(t *testing.T) {
		// Create a path that is highly unlikely to exist
		nonExistentPath := filepath.Join(os.TempDir(), "non_existent_loader_test_dir_12345abcde")
		l := &loader.Loader{}
		_, err := l.Load(nonExistentPath)
		require.Error(t, err, "Expected an error for non-existent location")
		// filepath.WalkDir returns an error that includes the path.
		// The exact error message can vary slightly by OS (e.g., "no such file or directory" vs "cannot find the path specified").
		// Checking for "walking path" and the path itself is a robust way.
		assert.Contains(t, err.Error(), nonExistentPath, "Error message should contain the non-existent path")
	})
}
