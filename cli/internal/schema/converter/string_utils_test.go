package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringUtils_Sanitize(t *testing.T) {
	t.Parallel()

	su := &StringUtils{}

	t.Run("BasicMode", func(t *testing.T) {
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
			{"Complex", "Hello-World.Test Case", "hello_world_test_case"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := su.sanitize(tt.input, SanitizationModeBasic)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("EventMode", func(t *testing.T) {
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
			{"Multiple underscores", "hello___world", "hello_world"},
			{"Leading/trailing underscores", "_hello_world_", "hello_world"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := su.sanitize(tt.input, SanitizationModeEvent)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestStringUtils_SanitizeBasic(t *testing.T) {
	t.Parallel()

	su := &StringUtils{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Simple string", "hello", "hello"},
		{"With dots", "hello.world", "hello_world"},
		{"With hyphens", "hello-world", "hello_world"},
		{"With spaces", "hello world", "hello_world"},
		{"Mixed case", "Hello World", "hello_world"},
		{"Multiple separators", "hello.world-test case", "hello_world_test_case"},
		{"Numbers", "test123", "test123"},
		{"Alphanumeric with separators", "test1.2-3 4", "test1_2_3_4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := su.sanitizeBasic(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringUtils_SanitizeEventID(t *testing.T) {
	t.Parallel()

	su := &StringUtils{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Simple string", "hello", "hello"},
		{"Path separators", "hello/world\\test", "hello_world_test"},
		{"Query characters", "hello?world", "hello_world"},
		{"Angle brackets", "hello<world>test", "hello_world_test"},
		{"Quotes", "hello\"world'test", "hello_world_test"},
		{"Parentheses", "hello(world)test", "hello_world_test"},
		{"Brackets", "hello[world]test", "hello_world_test"},
		{"Braces", "hello{world}test", "hello_world_test"},
		{"Special symbols", "hello|#%&*+=@!test", "hello_test"},
		{"Whitespace", "hello\tworld\ntest\rend", "hello_world_test_end"},
		{"Multiple underscores", "hello___world____test", "hello_world_test"},
		{"Leading underscores", "___hello", "hello"},
		{"Trailing underscores", "hello___", "hello"},
		{"Mixed case", "Hello_World", "hello_world"},
		{"Complex example", "User/Profile[123]?name=John&age=30", "user_profile_123_name_john_age_30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := su.sanitizeEventID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringUtils_GenerateHash(t *testing.T) {
	t.Parallel()

	su := &StringUtils{}

	t.Run("WithoutPrefix", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			content  string
			expected string // First 8 chars of MD5
		}{
			{"Simple string", "hello", "5d41402a"}, // MD5 of "hello"
			{"Empty string", "", "d41d8cd9"},       // MD5 of ""
			{"Numbers", "123", "202cb962"},         // MD5 of "123"
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := su.generateHash(tt.content, "")
				assert.Equal(t, tt.expected, result)
				assert.Len(t, result, 8)
			})
		}
	})

	t.Run("WithPrefix", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			content  string
			prefix   string
			expected string
		}{
			{"Simple with prefix", "hello", "test", "493f21ff"},     // MD5 of "test_hello"
			{"Empty content with prefix", "", "prefix", "0f034bb3"}, // MD5 of "prefix_"
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := su.generateHash(tt.content, tt.prefix)
				assert.Equal(t, tt.expected, result)
				assert.Len(t, result, 8)
			})
		}
	})

	t.Run("Consistency", func(t *testing.T) {
		t.Parallel()

		result1 := su.generateHash("test", "prefix")
		result2 := su.generateHash("test", "prefix")
		assert.Equal(t, result1, result2, "Hash should be consistent")
	})
}

func TestStringUtils_EnsureUnique(t *testing.T) {
	t.Parallel()

	su := &StringUtils{}

	t.Run("CounterStrategy", func(t *testing.T) {
		t.Parallel()

		usedNames := make(map[string]bool)

		// First call should return original name
		result1 := su.ensureUnique("test", usedNames, UniquenessStrategyCounter, 0)
		assert.Equal(t, "test", result1)
		assert.True(t, usedNames["test"])

		// Second call should return test_1
		result2 := su.ensureUnique("test", usedNames, UniquenessStrategyCounter, 0)
		assert.Equal(t, "test_1", result2)
		assert.True(t, usedNames["test_1"])

		// Third call should return test_2
		result3 := su.ensureUnique("test", usedNames, UniquenessStrategyCounter, 0)
		assert.Equal(t, "test_2", result3)
		assert.True(t, usedNames["test_2"])
	})

	t.Run("LetterSuffixStrategy", func(t *testing.T) {
		t.Parallel()

		usedNames := make(map[string]bool)

		// First call should return original name
		result1 := su.ensureUnique("test", usedNames, UniquenessStrategyLetterSuffix, 0)
		assert.Equal(t, "test", result1)

		// Second call should return testA
		result2 := su.ensureUnique("test", usedNames, UniquenessStrategyLetterSuffix, 0)
		assert.Equal(t, "testA", result2)

		// Third call should return testB
		result3 := su.ensureUnique("test", usedNames, UniquenessStrategyLetterSuffix, 0)
		assert.Equal(t, "testB", result3)
	})

	t.Run("LetterSuffixWithMaxLength", func(t *testing.T) {
		t.Parallel()

		usedNames := make(map[string]bool)

		// Test with max length constraint
		result1 := su.ensureUnique("verylongname", usedNames, UniquenessStrategyLetterSuffix, 10)
		assert.Equal(t, "verylongname", result1) // First should be accepted even if over limit

		result2 := su.ensureUnique("verylongname", usedNames, UniquenessStrategyLetterSuffix, 10)
		assert.Equal(t, "verylongnA", result2) // Should be truncated and suffixed
		assert.LessOrEqual(t, len(result2), 10)
	})

	t.Run("LetterSuffixExhaustion", func(t *testing.T) {
		t.Parallel()

		usedNames := make(map[string]bool)

		// Use up single letters
		for i := 0; i < 27; i++ { // test, testA through testZ
			su.ensureUnique("test", usedNames, UniquenessStrategyLetterSuffix, 0)
		}

		// Next should use double letters
		result := su.ensureUnique("test", usedNames, UniquenessStrategyLetterSuffix, 0)
		assert.Equal(t, "testAA", result)
	})
}

func TestStringUtils_EnsureUniqueWithCounter(t *testing.T) {
	t.Parallel()

	su := &StringUtils{}

	usedNames := make(map[string]bool)

	// Test sequential counter generation
	names := []string{"test", "test_1", "test_2", "test_3"}
	for i, expected := range names {
		result := su.ensureUniqueWithCounter("test", usedNames)
		assert.Equal(t, expected, result, "Iteration %d should produce %s", i, expected)
		assert.True(t, usedNames[expected])
	}
}

func TestStringUtils_EnsureUniqueWithLetterSuffix(t *testing.T) {
	t.Parallel()

	su := &StringUtils{}

	t.Run("NormalCase", func(t *testing.T) {
		t.Parallel()

		usedNames := make(map[string]bool)

		// Test sequential letter generation
		expectedSequence := []string{"testA", "testB", "testC"}
		usedNames["test"] = true // Simulate base name already used

		for i, expected := range expectedSequence {
			result := su.ensureUniqueWithLetterSuffix("test", usedNames, 0)
			assert.Equal(t, expected, result, "Iteration %d should produce %s", i, expected)
			assert.True(t, usedNames[expected])
		}
	})

	t.Run("WithMaxLength", func(t *testing.T) {
		t.Parallel()

		usedNames := make(map[string]bool)
		usedNames["verylongname"] = true // Base name used

		result := su.ensureUniqueWithLetterSuffix("verylongname", usedNames, 10)
		assert.LessOrEqual(t, len(result), 10)
		assert.True(t, usedNames[result])
	})

	t.Run("ExtremelyShortMaxLength", func(t *testing.T) {
		t.Parallel()

		usedNames := make(map[string]bool)
		usedNames["verylongname"] = true // Base name used

		result := su.ensureUniqueWithLetterSuffix("verylongname", usedNames, 3)
		assert.Equal(t, "GenTypeA", result) // Should fall back to GenType pattern
		assert.True(t, usedNames[result])
	})

	t.Run("ExhaustSingleLetters", func(t *testing.T) {
		t.Parallel()

		usedNames := make(map[string]bool)
		usedNames["test"] = true

		// Use up all single letters A-Z
		for i := 0; i < 26; i++ {
			su.ensureUniqueWithLetterSuffix("test", usedNames, 0)
		}

		// Next should be double letter
		result := su.ensureUniqueWithLetterSuffix("test", usedNames, 0)
		assert.Equal(t, "testAA", result)
	})
}
