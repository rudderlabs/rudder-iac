package helpers

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceKeyToFileName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "event with colon",
			input:    "event:public_api_request",
			expected: "event_public_api_request",
		},
		{
			name:     "source with colon and dash",
			input:    "source:my-source",
			expected: "source_my-source",
		},
		{
			name:     "with spaces",
			input:    "with spaces",
			expected: "with_spaces",
		},
		{
			name:     "uppercase conversion",
			input:    "UPPERCASE",
			expected: "uppercase",
		},
		{
			name:     "multiple invalid chars",
			input:    "test:name with/spaces\\and*more",
			expected: "test_name_with_spaces_and_more",
		},
		{
			name:     "leading and trailing invalid chars",
			input:    ":test_name:",
			expected: "test_name",
		},
	}

	sfm, err := NewStateFileManager(t.TempDir())
	require.Nil(t, err)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			result := sfm.resourceURNToFileName(c.input)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestStateFileManager(t *testing.T) {
	t.Parallel()

	sfm, err := NewStateFileManager(filepath.Join(".", "testdata"))
	require.Nil(t, err)

	t.Run("StateFileManager_LoadExpectedState", func(t *testing.T) {
		t.Parallel()

		t.Run("successful loading", func(t *testing.T) {
			t.Parallel()

			state, err := sfm.LoadExpectedState("event:public_api_request")
			require.Nil(t, err)

			assert.Equal(t, "public_api_request", state["name"])

			if state["id"] != "event-123" {
				t.Errorf("Expected id 'event-123', got %v", state["id"])
			}
		})

		t.Run("non-existent file", func(t *testing.T) {
			t.Parallel()

			_, err := sfm.LoadExpectedState("non_existent_resource")
			assert.NotNil(t, err)
		})

		t.Run("malformed JSON", func(t *testing.T) {
			t.Parallel()

			_, err := sfm.LoadExpectedState("invalid_json")
			assert.NotNil(t, err)
		})
	})

	t.Run("StateFileManager_ListResources", func(t *testing.T) {
		t.Parallel()

		resources, err := sfm.ListResources()
		assert.Nil(t, err)

		assert.ElementsMatch(t, []string{
			"event_public_api_request",
			"source_my-source",
			"invalid_json",
			"non_json.txt",
		}, resources)
	})

	t.Run("StateFileManager_LoadExpectedVersion", func(t *testing.T) {
		t.Parallel()

		t.Run("existing version file", func(t *testing.T) {
			t.Parallel()

			version, ok := sfm.LoadExpectedVersion()
			require.True(t, ok)
			assert.Equal(t, "1.2.3", version)
		})

		t.Run("missing version file", func(t *testing.T) {
			t.Parallel()

			newSfm, err := NewStateFileManager(t.TempDir())
			require.Nil(t, err)

			version, ok := newSfm.LoadExpectedVersion()
			require.False(t, ok)
			assert.Equal(t, "0.0.0", version)
		})
	})
}
