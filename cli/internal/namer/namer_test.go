package namer

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
