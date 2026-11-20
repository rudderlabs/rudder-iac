package destination

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestRemovedAPIKeys_NoneRightAfterACLIWrite pins that a destination the CLI has just
// written previews no removed keys. Each update snapshot under
// tests/testdata/expected/upstream is the API config read back after the e2e applied
// the matching update fixture. A key in it that removedAPIKeys reports is one the
// backend adds on every write, such as a schema default, so every plan would name it
// as removed while the next write puts it back.
func TestRemovedAPIKeys_NoneRightAfterACLIWrite(t *testing.T) {
	t.Parallel()

	registry, err := builtin.NewRegistry(true)
	require.NoError(t, err)
	h := &HandlerImpl{registry: registry}

	fixtures, err := filepath.Glob("../../../tests/testdata/destinations/update/*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)

	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			t.Parallel()

			raw, err := os.ReadFile(fixture)
			require.NoError(t, err)
			var document struct {
				Spec map[string]any `yaml:"spec"`
			}
			require.NoError(t, yaml.Unmarshal(raw, &document))
			var spec DestinationSpec
			require.NoError(t, mapstructure.Decode(document.Spec, &spec))
			resources, err := h.ExtractResourcesFromSpec("", &spec)
			require.NoError(t, err)
			data := resources[spec.ID]

			raw, err = os.ReadFile("../../../tests/testdata/expected/upstream/destinations/update/destination_" + spec.ID)
			require.NoError(t, err)
			var snapshot struct {
				Config json.RawMessage `json:"config"`
			}
			require.NoError(t, json.Unmarshal(raw, &snapshot))

			registered, err := registry.Get(data.Type, data.DefinitionVersion)
			require.NoError(t, err)
			removed, err := h.removedAPIKeys(registered, snapshot.Config, data)
			require.NoError(t, err)
			assert.Empty(t, removed, "the backend re-adds these on every write, so they are no removal")
		})
	}
}

// TestEveryRegisteredTypeHasAnUpdateFixture keeps a definition from skipping the
// guard above. A type with no update fixture has no snapshot to check, so a key the
// backend stamps on write would show as a removal on every plan for it.
func TestEveryRegisteredTypeHasAnUpdateFixture(t *testing.T) {
	t.Parallel()

	// Unverified types with no fixture yet. Its snapshot needs a live e2e run, so a
	// type leaves this list when its fixture lands.
	knownWithoutFixture := map[string]bool{
		"bingads_offline_conversions":        true,
		"fb_custom_audience":                 true,
		"google_adwords_offline_conversions": true,
		"google_adwords_remarketing_lists":   true,
		"iterable_audience":                  true,
		"linkedin_ads":                       true,
	}

	fixtures, err := filepath.Glob("../../../tests/testdata/destinations/update/*.yaml")
	require.NoError(t, err)
	covered := map[string]bool{}
	for _, fixture := range fixtures {
		raw, err := os.ReadFile(fixture)
		require.NoError(t, err)
		var document struct {
			Spec struct {
				Type string `yaml:"type"`
			} `yaml:"spec"`
		}
		require.NoError(t, yaml.Unmarshal(raw, &document))
		covered[document.Spec.Type] = true
	}

	registry, err := builtin.NewRegistry(true)
	require.NoError(t, err)
	for _, registered := range registry.Definitions() {
		if knownWithoutFixture[registered.Type] {
			assert.False(t, covered[registered.Type], "%s now has an update fixture: remove it from knownWithoutFixture", registered.Type)
			continue
		}
		assert.True(t, covered[registered.Type], "%s has no fixture in tests/testdata/destinations/update, so TestRemovedAPIKeys_NoneRightAfterACLIWrite never checks it", registered.Type)
	}
}
