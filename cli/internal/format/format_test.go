package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// parses the given YAML into a generic value for semantic comparison.
func parse(t *testing.T, data []byte) any {
	t.Helper()
	var v any
	require.NoError(t, yaml.Unmarshal(data, &v))
	return v
}

func TestSource_NormalizesIndentation(t *testing.T) {
	input := []byte("version: rudderstack/v0.1\nmetadata:\n    name: a\nspec:\n    id: b\n")
	want := "version: rudderstack/v0.1\nmetadata:\n  name: a\nspec:\n  id: b\n"

	got, err := Source(input)
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

func TestSource_PreservesKeyOrder(t *testing.T) {
	input := []byte("zebra: 1\napple: 2\nmango: 3\n")

	got, err := Source(input)
	require.NoError(t, err)
	assert.Equal(t, "zebra: 1\napple: 2\nmango: 3\n", string(got))
}

func TestSource_PreservesComments(t *testing.T) {
	input := []byte("# leading comment\nkey: value # inline comment\n")

	got, err := Source(input)
	require.NoError(t, err)
	assert.Contains(t, string(got), "# leading comment")
	assert.Contains(t, string(got), "# inline comment")
}

func TestSource_Idempotent(t *testing.T) {
	inputs := [][]byte{
		[]byte("version: rudderstack/v0.1\nmetadata:\n    name: a\nspec:\n    id: b\n"),
		[]byte("# head\nlist:\n  - one\n  - two\nnested:\n  a:\n    b: c\n"),
		[]byte("key: value # inline\nother:   spaced\n"),
		[]byte("a:\n- x\n- y\n"),
	}

	for _, in := range inputs {
		once, err := Source(in)
		require.NoError(t, err)
		twice, err := Source(once)
		require.NoError(t, err)
		assert.Equal(t, string(once), string(twice), "formatting must be idempotent for %q", string(in))
	}
}

func TestSource_SemanticsPreserved(t *testing.T) {
	input := []byte("version: rudderstack/v0.1\nmetadata:\n    name: a\nspec:\n  list:\n  - 1\n  - 2\n  flag: true\n")

	got, err := Source(input)
	require.NoError(t, err)
	assert.Equal(t, parse(t, input), parse(t, got), "formatted output must parse to the same value")
}

func TestSource_EmptyAndBlank(t *testing.T) {
	for _, in := range []string{"", "\n", "   \n"} {
		got, err := Source([]byte(in))
		require.NoError(t, err)
		// An empty document round-trips to empty output.
		assert.Empty(t, string(got))
	}
}

func TestSource_InvalidYAMLErrors(t *testing.T) {
	_, err := Source([]byte("key: : : bad\n"))
	assert.Error(t, err)
}
