package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

type ConversionCase struct {
	Name      string
	LocalJSON string
	APIJSON   string
}

func AssertConversion(t *testing.T, props []converter.ConfigProperty, cases []ConversionCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			var local map[string]any
			require.NoError(t, json.Unmarshal([]byte(tc.LocalJSON), &local))

			var expectedAPI map[string]any
			require.NoError(t, json.Unmarshal([]byte(tc.APIJSON), &expectedAPI))

			actualAPI, err := converter.LocalToAPI(props, local)
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(expectedAPI, actualAPI))

			actualLocal, err := converter.APIToLocal(props, expectedAPI)
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(local, actualLocal))
		})
	}
}

func AssertConversionFromFiles(t *testing.T, props []converter.ConfigProperty, testdataDir string) {
	t.Helper()

	entries, err := os.ReadDir(testdataDir)
	require.NoError(t, err)

	cases := make([]ConversionCase, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".local.json") {
			continue
		}

		baseName := strings.TrimSuffix(entry.Name(), ".local.json")
		localPath := filepath.Join(testdataDir, entry.Name())
		apiPath := filepath.Join(testdataDir, baseName+".api.json")

		localBytes, err := os.ReadFile(localPath)
		require.NoError(t, err)

		apiBytes, err := os.ReadFile(apiPath)
		require.NoError(t, err)

		cases = append(cases, ConversionCase{
			Name:      baseName,
			LocalJSON: string(localBytes),
			APIJSON:   string(apiBytes),
		})
	}

	AssertConversion(t, props, cases)
}
