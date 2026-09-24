package namer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKebabCase_Name(t *testing.T) {
	t.Parallel()
	strategy := NewKebabCase()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple phrase", "Hello World", "hello-world"},
		{"with numbers", "User123 Signed Up", "user123-signed-up"},
		{"special chars", "User@Signed#Up", "user-signed-up"},
		{"multiple spaces", "Hello   World", "hello-world"},
		{"leading trailing", " Hello World ", "hello-world"},
		{"empty", "", ""},
		{"single word", "Hello", "hello"},
		{"all special", "@#$%", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := strategy.Name(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExternalIdNamer_Name(t *testing.T) {
	t.Parallel()
	n := NewExternalIdNamer(NewKebabCase())

	tests := []struct {
		name     string
		inputs   []string
		expected []string
	}{
		{"unique names", []string{"HelloWorld", "Another Event"}, []string{"helloworld", "another-event"}},
		{"collisions", []string{"Hello World", "Hello World", "Hello World"}, []string{"hello-world", "hello-world-1", "hello-world-2"}},
		{"extra collisions", []string{"Test", "Test", "Test", "Test"}, []string{"test", "test-1", "test-2", "test-3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for i, input := range tt.inputs {
				result, err := n.Name(ScopeName{Name: input, Scope: "testscope"})
				assert.NoError(t, err)
				assert.Equal(t, tt.expected[i], result)
			}
		})
	}
}

func TestExternalIdNamer_Load(t *testing.T) {
	t.Parallel()
	n := NewExternalIdNamer(NewKebabCase())

	tests := []struct {
		name    string
		names   []ScopeName
		wantErr bool
		errMsg  string
	}{
		{"empty", []ScopeName{}, false, ""},
		{"no duplicates", []ScopeName{{Name: "three", Scope: "testscope"}, {Name: "four", Scope: "testscope"}}, false, ""},
		{"with duplicates", []ScopeName{{Name: "one", Scope: "testscope"}, {Name: "one", Scope: "testscope"}}, true, "loading name: one errored with: duplicate name exception"},
		{"extra duplicates", []ScopeName{{Name: "test", Scope: "testscope"}, {Name: "test", Scope: "testscope"}, {Name: "test", Scope: "testscope"}}, true, "loading name: test errored with: duplicate name exception"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := n.Load(tt.names)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollisionHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     string
		existing []string
		expected string
	}{
		{"simple collision", "base", []string{"base"}, "base-1"},
		{"multiple collisions", "base", []string{"base", "base-1", "base-2"}, "base-3"},
		{"edge empty", "", []string{}, "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := collisionHandler(tt.base, tt.existing)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSnakeCase_Name(t *testing.T) {
	t.Parallel()
	strategy := NewSnakeCase()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple phrase", "Hello World", "hello_world"},
		{"with numbers", "User123 Signed Up", "user123_signed_up"},
		{"special chars", "User@Signed#Up", "user_signed_up"},
		{"multiple spaces", "Hello   World", "hello_world"},
		{"leading trailing", " Hello World ", "hello_world"},
		{"empty", "", ""},
		{"single word", "Hello", "hello"},
		{"all special", "@#$%", ""},
		{"camelCase input", "multipleOf", "multiple_of"},
		{"camelCase multiple words", "minLength", "min_length"},
		{"camelCase longer", "exclusiveMinimum", "exclusive_minimum"},
		{"mixed camelCase and separators", "itemTypes-test", "item_types_test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := strategy.Name(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCamelCase_Name(t *testing.T) {
	t.Parallel()
	strategy := NewCamelCase()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple phrase", "Hello World", "helloWorld"},
		{"with numbers", "User123 Signed Up", "user123SignedUp"},
		{"special chars", "User@Signed#Up", "userSignedUp"},
		{"multiple spaces", "Hello   World", "helloWorld"},
		{"leading trailing", " Hello World ", "helloWorld"},
		{"empty", "", ""},
		{"single word", "Hello", "hello"},
		{"all special", "@#$%", ""},
		{"snake_case input", "abc_def", "abcDef"},
		{"kebab-case input", "abc-def", "abcDef"},
		{"mixed separators", "abc_def-ghi", "abcDefGhi"},
		{"already camelCase", "abcDef", "abcDef"},
		{"numbers in middle", "abc123Def", "abc123Def"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := strategy.Name(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveIDs(t *testing.T) {
	t.Parallel()

	t.Run("keeps upstream externalIDs and generates the rest", func(t *testing.T) {
		n := NewExternalIdNamer(NewKebabCase())

		ids, err := ResolveIDs(n, "source", []IDCandidate{
			{Key: "r1", Name: "Prod HTTP", ExternalID: "prod-http"},
			{Key: "r2", Name: "Staging HTTP"},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"r1": "prod-http", "r2": "staging-http"}, ids)
	})

	t.Run("a generated name never lands on a kept one", func(t *testing.T) {
		n := NewExternalIdNamer(NewKebabCase())

		// Both resolve to "prod-http": one keeps it, the other must not get it.
		ids, err := ResolveIDs(n, "source", []IDCandidate{
			{Key: "managed", Name: "Prod HTTP", ExternalID: "prod-http"},
			{Key: "unmanaged", Name: "Prod HTTP"},
		})
		require.NoError(t, err)
		assert.Equal(t, "prod-http", ids["managed"])
		assert.NotEqual(t, "prod-http", ids["unmanaged"])
	})

	t.Run("an externalID already loaded by the caller is not an error", func(t *testing.T) {
		n := NewExternalIdNamer(NewKebabCase())
		// Import preloads the local project's IDs, which for a managed resource
		// are the same identifiers ResolveIDs then reserves.
		require.NoError(t, n.Load([]ScopeName{{Name: "prod-http", Scope: "source"}}))

		ids, err := ResolveIDs(n, "source", []IDCandidate{
			{Key: "r1", Name: "Prod HTTP", ExternalID: "prod-http"},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"r1": "prod-http"}, ids)
	})

	t.Run("scopes are independent", func(t *testing.T) {
		n := NewExternalIdNamer(NewKebabCase())

		sources, err := ResolveIDs(n, "source", []IDCandidate{{Key: "s", Name: "Checkout"}})
		require.NoError(t, err)
		destinations, err := ResolveIDs(n, "destination", []IDCandidate{{Key: "d", Name: "Checkout"}})
		require.NoError(t, err)

		assert.Equal(t, "checkout", sources["s"])
		assert.Equal(t, "checkout", destinations["d"])
	})
}
