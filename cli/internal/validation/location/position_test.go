package location

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLDataIndex(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		path     string
		wantLine int
		wantCol  int
	}{
		{
			name:     "simple scalar",
			yaml:     "version: rudder/v0.1",
			path:     "/version",
			wantLine: 1,
			wantCol:  10,
		},
		{
			name: "nested maps",
			yaml: `
metadata:
  name: "test-resource"
spec:
  key: value`,
			path:     "/metadata/name",
			wantLine: 3,
			wantCol:  9,
		},
		{
			name: "array elements",
			yaml: `
spec:
  properties:
    - id: "p1"
    - id: "p2"`,
			path:     "/spec/properties/0/id",
			wantLine: 4,
			wantCol:  11,
		},
		{
			name: "deep nesting",
			yaml: `
spec:
  properties:
    - id: "p1"
      config:
        enum:
          - "v1"
          - "v2"`,
			path:     "/spec/properties/0/config/enum/1",
			wantLine: 8,
			wantCol:  13,
		},
		{
			name:     "root lookup",
			yaml:     "version: v1",
			path:     "",
			wantLine: 1,
			wantCol:  1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idx, err := YAMLDataIndex([]byte(tc.yaml))
			require.NoError(t, err)

			pos := idx.Lookup(tc.path)
			require.NotNil(t, pos, "Position not found for path %s", tc.path)
			assert.Equal(t, tc.wantLine, pos.Line, "Line mismatch")
			assert.Equal(t, tc.wantCol, pos.Column, "Column mismatch")
		})
	}
}

func TestPathIndex_Lookup_EdgeCases(t *testing.T) {
	t.Run("non-existent path", func(t *testing.T) {
		yamlData := "version: v1"
		idx, err := YAMLDataIndex([]byte(yamlData))
		require.NoError(t, err)

		pos := idx.Lookup("/non/existent")
		assert.Nil(t, pos)
	})

	t.Run("empty yaml", func(t *testing.T) {
		idx, err := YAMLDataIndex([]byte(""))
		require.NoError(t, err)
		assert.Nil(t, idx.Lookup("/any"))
	})

	t.Run("malformed yaml", func(t *testing.T) {
		idx, err := YAMLDataIndex([]byte("invalid: yaml: :"))
		assert.Error(t, err)
		assert.Nil(t, idx)
	})
}
