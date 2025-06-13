package models

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultEventTypeConfig(t *testing.T) {
	t.Parallel()

	config := DefaultEventTypeConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.EventMappings)

	// Check default mappings
	expectedMappings := map[string]string{
		"identify": "$.properties",
		"page":     "$.properties",
		"screen":   "$.properties",
		"group":    "$.properties",
		"alias":    "$.properties",
	}

	assert.Equal(t, expectedMappings, config.EventMappings)
}

func TestLoadEventTypeConfig(t *testing.T) {
	t.Parallel()

	t.Run("EmptyConfigPath", func(t *testing.T) {
		t.Parallel()

		config, err := LoadEventTypeConfig("")
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Should return default config
		defaultConfig := DefaultEventTypeConfig()
		assert.Equal(t, defaultConfig.EventMappings, config.EventMappings)
	})

	t.Run("FileNotFound", func(t *testing.T) {
		t.Parallel()

		config, err := LoadEventTypeConfig("/nonexistent/config.yaml")
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("ValidConfig", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "config.yaml")

		configContent := `
event_mappings:
  identify: "$.traits"
  page: "$.context.traits"
  screen: "$.properties"
  group: "$.properties"
  alias: "$.traits"
`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := LoadEventTypeConfig(configFile)
		require.NoError(t, err)
		assert.NotNil(t, config)

		expectedMappings := map[string]string{
			"identify": "$.traits",
			"page":     "$.context.traits",
			"screen":   "$.properties",
			"group":    "$.properties",
			"alias":    "$.traits",
		}

		assert.Equal(t, expectedMappings, config.EventMappings)
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "invalid.yaml")

		invalidYAML := `
event_mappings:
  identify: "$.traits"
  page: "$.context.traits
  # Missing closing quote
`

		err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
		require.NoError(t, err)

		config, err := LoadEventTypeConfig(configFile)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to parse YAML")
	})

	t.Run("InvalidMapping", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "invalid_mapping.yaml")

		configContent := `
event_mappings:
  identify: "$.invalid.path"
  page: "$.context.traits"
`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := LoadEventTypeConfig(configFile)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "invalid JSONPath")
	})
}

func TestGetJSONPathForEventType(t *testing.T) {
	t.Parallel()

	config := &EventTypeConfig{
		EventMappings: map[string]string{
			"identify": "$.traits",
			"page":     "$.context.traits",
			"screen":   "$.properties",
			"group":    "$.properties",
			"alias":    "$.traits",
		},
	}

	cases := []struct {
		name         string
		eventType    string
		expectedPath string
	}{
		{
			name:         "TrackAlwaysProperties",
			eventType:    "track",
			expectedPath: "$.properties",
		},
		{
			name:         "IdentifyFromConfig",
			eventType:    "identify",
			expectedPath: "$.traits",
		},
		{
			name:         "PageFromConfig",
			eventType:    "page",
			expectedPath: "$.context.traits",
		},
		{
			name:         "ScreenFromConfig",
			eventType:    "screen",
			expectedPath: "$.properties",
		},
		{
			name:         "GroupFromConfig",
			eventType:    "group",
			expectedPath: "$.properties",
		},
		{
			name:         "AliasFromConfig",
			eventType:    "alias",
			expectedPath: "$.traits",
		},
		{
			name:         "UnknownEventTypeDefault",
			eventType:    "unknown",
			expectedPath: "$.properties",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			path := config.GetJSONPathForEventType(c.eventType)
			assert.Equal(t, c.expectedPath, path)
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("ValidMappings", func(t *testing.T) {
		t.Parallel()

		config := &EventTypeConfig{
			EventMappings: map[string]string{
				"identify": "$.traits",
				"page":     "$.context.traits",
				"screen":   "$.properties",
			},
		}

		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("InvalidJSONPath", func(t *testing.T) {
		t.Parallel()

		config := &EventTypeConfig{
			EventMappings: map[string]string{
				"identify": "$.invalid.path",
				"page":     "$.context.traits",
			},
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSONPath")
		assert.Contains(t, err.Error(), "$.invalid.path")
		assert.Contains(t, err.Error(), "identify")
	})

	t.Run("MultipleInvalidPaths", func(t *testing.T) {
		t.Parallel()

		config := &EventTypeConfig{
			EventMappings: map[string]string{
				"identify": "$.invalid.path",
				"page":     "$.another.invalid",
			},
		}

		err := config.Validate()
		assert.Error(t, err)
		// Should fail on first invalid path
		assert.Contains(t, err.Error(), "invalid JSONPath")
	})

	t.Run("EmptyMappings", func(t *testing.T) {
		t.Parallel()

		config := &EventTypeConfig{
			EventMappings: map[string]string{},
		}

		err := config.Validate()
		assert.NoError(t, err) // Empty mappings are valid
	})
}

func TestHasCustomMappings(t *testing.T) {
	t.Parallel()

	t.Run("DefaultMappings", func(t *testing.T) {
		t.Parallel()

		config := DefaultEventTypeConfig()
		assert.False(t, config.HasCustomMappings())
	})

	t.Run("CustomMappings", func(t *testing.T) {
		t.Parallel()

		config := &EventTypeConfig{
			EventMappings: map[string]string{
				"identify": "$.traits",
				"page":     "$.context.traits", // Different from default
				"screen":   "$.properties",
				"group":    "$.properties",
				"alias":    "$.properties",
			},
		}

		assert.True(t, config.HasCustomMappings())
	})

	t.Run("PartialCustomMappings", func(t *testing.T) {
		t.Parallel()

		config := &EventTypeConfig{
			EventMappings: map[string]string{
				"identify": "$.properties", // Same as default
				"page":     "$.properties", // Same as default
				"screen":   "$.traits",     // Different from default
			},
		}

		assert.True(t, config.HasCustomMappings())
	})

	t.Run("EmptyMappings", func(t *testing.T) {
		t.Parallel()

		config := &EventTypeConfig{
			EventMappings: map[string]string{},
		}

		assert.False(t, config.HasCustomMappings())
	})
}

func TestValidJSONPaths(t *testing.T) {
	t.Parallel()

	validPaths := []string{
		"$.traits",
		"$.context.traits",
		"$.properties",
	}

	for _, path := range validPaths {
		t.Run("ValidPath_"+path, func(t *testing.T) {
			t.Parallel()

			config := &EventTypeConfig{
				EventMappings: map[string]string{
					"test": path,
				},
			}

			err := config.Validate()
			assert.NoError(t, err, "Path %s should be valid", path)
		})
	}
}

func TestInvalidJSONPaths(t *testing.T) {
	t.Parallel()

	invalidPaths := []string{
		"$.invalid",
		"$.context.invalid",
		"$.properties.invalid",
		"traits",          // Missing $
		"$context.traits", // Missing dot
		"",                // Empty
	}

	for _, path := range invalidPaths {
		t.Run("InvalidPath_"+path, func(t *testing.T) {
			t.Parallel()

			config := &EventTypeConfig{
				EventMappings: map[string]string{
					"test": path,
				},
			}

			err := config.Validate()
			assert.Error(t, err, "Path %s should be invalid", path)
		})
	}
}
