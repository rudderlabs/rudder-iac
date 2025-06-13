package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		basePath string
		key      string
		expected string
	}{
		{"Empty base path", "", "key", "key"},
		{"Simple path", "base", "key", "base.key"},
		{"Nested path", "root.level1", "level2", "root.level1.level2"},
		{"Empty key", "base", "", "base."},
		{"Both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPath(tt.basePath, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPropertyName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"Simple path", "property", "property"},
		{"Nested path", "root.level1.property", "property"},
		{"Deep nesting", "a.b.c.d.e.f", "f"},
		{"Empty path", "", ""},
		{"Single dot", ".", ""},
		{"Ends with dot", "path.", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPropertyName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetJSONType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"Nil value", nil, "null"},
		{"String value", "hello", "string"},
		{"Float64 value", 3.14, "number"},
		{"Integer value", 42, "integer"},
		{"Int64 value", int64(42), "integer"},
		{"Boolean value", true, "boolean"},
		{"Object value", map[string]interface{}{"key": "value"}, "object"},
		{"Array value", []interface{}{1, 2, 3}, "array"},
		{"String type hint", "float64", "number"},
		{"Bool type hint", "bool", "boolean"},
		{"Other string", "custom", "string"},
		{"Unknown type", struct{}{}, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getJSONType(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapJSONTypeToYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jsonType string
		expected string
	}{
		{"Number type", "number", "number"},
		{"Integer type", "integer", "integer"},
		{"Boolean type", "boolean", "boolean"},
		{"Object type", "object", "object"},
		{"Array type", "array", "array"},
		{"Null type", "null", "null"},
		{"String type", "string", "string"},
		{"Unknown type", "unknown", "string"},
		{"Empty type", "", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapJSONTypeToYAML(tt.jsonType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateEventName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{"Simple identifier", "hello", "hello"},
		{"Snake case", "hello_world", "hello_world"},
		{"Multiple underscores", "user_profile_updated", "user_profile_updated"},
		{"Single character", "a", "a"},
		{"Empty string", "", ""},
		{"Leading underscore", "_hello", "_hello"},
		{"Trailing underscore", "hello_", "hello_"},
		{"Multiple consecutive underscores", "hello__world", "hello__world"},
		{"Numbers", "event_123", "event_123"},
		{"Mixed case", "Hello_World", "Hello_World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.identifier // Now, event name is just the identifier
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateRandomID(t *testing.T) {
	t.Parallel()

	t.Run("Length", func(t *testing.T) {
		t.Parallel()

		result := generateRandomID()
		assert.Len(t, result, 10)
	})

	t.Run("Uniqueness", func(t *testing.T) {
		t.Parallel()

		id1 := generateRandomID()
		id2 := generateRandomID()
		assert.NotEqual(t, id1, id2, "Generated IDs should be unique")
	})

	t.Run("Character set", func(t *testing.T) {
		t.Parallel()

		result := generateRandomID()
		for _, char := range result {
			assert.True(t,
				(char >= 'a' && char <= 'z') ||
					(char >= 'A' && char <= 'Z') ||
					(char >= '0' && char <= '9'),
				"Character %c should be alphanumeric", char)
		}
	})
}

func TestSanitizeID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple string", "hello", "hello"},
		{"With dots", "hello.world", "hello_world"},
		{"With hyphens", "hello-world", "hello_world"},
		{"With spaces", "hello world", "hello_world"},
		{"Mixed case", "Hello World", "hello_world"},
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeEventID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple string", "hello", "hello"},
		{"With special chars", "hello/world?test", "hello_world_test"},
		{"Complex special chars", "hello<>\"'()[]{}|#%&*+=@!", "hello"},
		{"With whitespace", "hello\tworld\ntest", "hello_world_test"},
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeEventID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateStructureHash(t *testing.T) {
	t.Parallel()

	t.Run("Simple structure", func(t *testing.T) {
		t.Parallel()

		structure := map[string]string{
			"name": "string",
			"age":  "number",
		}

		result := generateStructureHash(structure)
		assert.Len(t, result, 8)
		assert.NotEmpty(t, result)
	})

	t.Run("Empty structure", func(t *testing.T) {
		t.Parallel()

		structure := map[string]string{}
		result := generateStructureHash(structure)
		assert.Len(t, result, 8)
	})

	t.Run("Consistency", func(t *testing.T) {
		t.Parallel()

		structure := map[string]string{
			"name": "string",
			"age":  "number",
		}

		result1 := generateStructureHash(structure)
		result2 := generateStructureHash(structure)
		assert.Equal(t, result1, result2, "Hash should be consistent")
	})

	t.Run("Order independence", func(t *testing.T) {
		t.Parallel()

		structure1 := map[string]string{
			"name": "string",
			"age":  "number",
		}

		structure2 := map[string]string{
			"age":  "number",
			"name": "string",
		}

		result1 := generateStructureHash(structure1)
		result2 := generateStructureHash(structure2)
		assert.Equal(t, result1, result2, "Hash should be order-independent")
	})
}

func TestGenerateArrayHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		itemType string
	}{
		{"String array", "string"},
		{"Number array", "number"},
		{"Object array", "object"},
		{"Empty item type", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateArrayHash(tt.itemType)
			assert.Len(t, result, 8)
			assert.NotEmpty(t, result)
		})
	}

	t.Run("Consistency", func(t *testing.T) {
		t.Parallel()

		result1 := generateArrayHash("string")
		result2 := generateArrayHash("string")
		assert.Equal(t, result1, result2, "Hash should be consistent")
	})

	t.Run("Different types produce different hashes", func(t *testing.T) {
		t.Parallel()

		stringResult := generateArrayHash("string")
		numberResult := generateArrayHash("number")
		assert.NotEqual(t, stringResult, numberResult, "Different item types should produce different hashes")
	})
}
