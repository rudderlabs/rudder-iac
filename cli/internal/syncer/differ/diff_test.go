package differ_test

import (
	"fmt"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareData(t *testing.T) {
	data1 := resources.ResourceData{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	data2 := resources.ResourceData{
		"key1": "value1",
		"key2": "value3",
		"key4": "value4",
	}

	diffs, _ := differ.CompareData(data1, data2)

	assert.Len(t, diffs, 3)

	assert.Contains(t, diffs, "key2")
	assert.Contains(t, diffs, "key3")
	assert.Contains(t, diffs, "key4")

	assert.Equal(t, diffs["key2"].SourceValue, "value2")
	assert.Equal(t, diffs["key2"].TargetValue, "value3")

	assert.Equal(t, diffs["key3"].SourceValue, "value3")
	assert.Nil(t, diffs["key3"].TargetValue)

	assert.Nil(t, diffs["key4"].SourceValue)
	assert.Equal(t, diffs["key4"].TargetValue, "value4")
}

func TestComputeDiff(t *testing.T) {
	g1 := resources.NewGraph()
	g2 := resources.NewGraph()

	g1.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g1.AddResource(resources.NewResource("r1", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g1.AddResource(resources.NewResource("r2", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))

	g2.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g2.AddResource(resources.NewResource("r1", "some-type", resources.ResourceData{"key1": "value1", "key2": "value3"}, []string{}))
	g2.AddResource(resources.NewResource("r3", "some-type", resources.ResourceData{"key1": "value1", "key2": "value3"}, []string{}))
	g2.AddResource(resources.NewResource("r4", "some-type", resources.ResourceData{"key1": "value1", "key2": "value4"}, []string{}, resources.WithResourceImportMetadata("remote-id-r4", "workspace-id")))

	diff := differ.ComputeDiff(g1, g2, differ.DiffOptions{WorkspaceID: "workspace-id"})

	assert.Len(t, diff.NewResources, 1)
	assert.Len(t, diff.ImportableResources, 1)
	assert.Len(t, diff.UpdatedResources, 1)
	assert.Len(t, diff.RemovedResources, 1)
	assert.Len(t, diff.UnmodifiedResources, 1)

	assert.Contains(t, diff.NewResources, "some-type:r3")
	assert.Contains(t, diff.ImportableResources, "some-type:r4")
	assert.Equal(t, diff.UpdatedResources["some-type:r1"], differ.ResourceDiff{URN: "some-type:r1", Diffs: map[string]differ.PropertyDiff{"key2": {Property: "key2", SourceValue: "value2", TargetValue: "value3"}}})
	assert.Contains(t, diff.RemovedResources, "some-type:r2")
	assert.Contains(t, diff.UnmodifiedResources, "some-type:r0")
}

// TestCompareData_Secret covers the secret-aware rules on the Data() path, where
// the concrete secret.String value lives directly in the resource map. It also
// asserts the secret-only verdict CompareData returns alongside the diffs.
func TestCompareData_Secret(t *testing.T) {
	t.Run("equal known secrets do not diff", func(t *testing.T) {
		diffs, secretOnly := differ.CompareData(
			resources.ResourceData{"token": secret.New("hunter2")},
			resources.ResourceData{"token": secret.New("hunter2")},
		)
		assert.Empty(t, diffs)
		assert.False(t, secretOnly)
	})

	t.Run("different known secrets diff, flagged Secret, keep both values for masking", func(t *testing.T) {
		diffs, secretOnly := differ.CompareData(
			resources.ResourceData{"token": secret.New("hunter2")},
			resources.ResourceData{"token": secret.New("hunter3")},
		)
		assert.Equal(t, map[string]differ.PropertyDiff{
			"token": {Property: "token", SourceValue: secret.New("hunter2"), TargetValue: secret.New("hunter3"), SecretOnly: true},
		}, diffs)
		assert.True(t, secretOnly)
	})

	t.Run("unknown remote always diffs and is flagged Secret", func(t *testing.T) {
		diffs, secretOnly := differ.CompareData(
			resources.ResourceData{"token": secret.New("hunter2")},
			resources.ResourceData{"token": secret.NewUnknown()},
		)
		assert.Equal(t, map[string]differ.PropertyDiff{
			"token": {Property: "token", SourceValue: secret.New("hunter2"), TargetValue: secret.NewUnknown(), SecretOnly: true},
		}, diffs)
		assert.True(t, secretOnly)
	})

	t.Run("secret nested in a map alone makes the map diff secret-only", func(t *testing.T) {
		diffs, secretOnly := differ.CompareData(
			resources.ResourceData{"config": map[string]any{"token": secret.New("hunter2")}},
			resources.ResourceData{"config": map[string]any{"token": secret.NewUnknown()}},
		)
		require.Contains(t, diffs, "config")
		assert.True(t, diffs["config"].SecretOnly)
		assert.True(t, secretOnly)
	})

	t.Run("secret nested in a map with a real sibling is a real diff", func(t *testing.T) {
		diffs, secretOnly := differ.CompareData(
			resources.ResourceData{"config": map[string]any{"token": secret.New("hunter2"), "name": "a"}},
			resources.ResourceData{"config": map[string]any{"token": secret.NewUnknown(), "name": "b"}},
		)
		require.Contains(t, diffs, "config")
		assert.False(t, diffs["config"].SecretOnly)
		assert.False(t, secretOnly)
	})

	t.Run("secret inside a slice re-applies every run, not classified secret-only, never leaks", func(t *testing.T) {
		// Slices fall back to reflect.DeepEqual, so a secret nested inside one
		// bypasses the secret case: an unknown remote never equals local, so it
		// diffs (re-applies) on every run, but the diff is not flagged Secret.
		// Format still masks, so the real value never leaks through the render path.
		slices := map[string]struct{ local, remote any }{
			"[]map[string]any": {
				local:  []map[string]any{{"token": secret.New("hunter2")}},
				remote: []map[string]any{{"token": secret.NewUnknown()}},
			},
			"[]any": {
				local:  []any{map[string]any{"token": secret.New("hunter2")}},
				remote: []any{map[string]any{"token": secret.NewUnknown()}},
			},
		}
		for name, tc := range slices {
			t.Run(name, func(t *testing.T) {
				// Two independent compares against the always-unknown remote both
				// diff: the resource re-applies on every run.
				for run := 1; run <= 2; run++ {
					diffs, secretOnly := differ.CompareData(
						resources.ResourceData{"creds": tc.remote},
						resources.ResourceData{"creds": tc.local},
					)
					require.Contains(t, diffs, "creds", "unknown remote must diff on run %d (re-applied every run)", run)
					assert.False(t, diffs["creds"].SecretOnly, "slice-nested secret is not classified secret-only")
					assert.False(t, secretOnly)

					rendered := fmt.Sprintf("%v -> %v", diffs["creds"].SourceValue, diffs["creds"].TargetValue)
					assert.NotContains(t, rendered, "hunter2", "real secret value leaked through render path: %s", rendered)
				}
			})
		}
	})

	t.Run("pointer flavor mirrors value flavor", func(t *testing.T) {
		local, remote := secret.New("hunter2"), secret.New("hunter3")
		diffs, secretOnly := differ.CompareData(
			resources.ResourceData{"token": &local},
			resources.ResourceData{"token": &remote},
		)
		// The pointer branch dereferences to the secret.String case, which records
		// the dereferenced values and the Secret flag.
		assert.Equal(t, map[string]differ.PropertyDiff{
			"token": {Property: "token", SourceValue: secret.New("hunter2"), TargetValue: secret.New("hunter3"), SecretOnly: true},
		}, diffs)
		assert.True(t, secretOnly)

		same := secret.New("hunter2")
		equal, secretOnly := differ.CompareData(
			resources.ResourceData{"token": &local},
			resources.ResourceData{"token": &same},
		)
		assert.Empty(t, equal)
		assert.False(t, secretOnly)
	})
}

// TestDiff_HasNonsecretDiff locks the import guard's discriminator. ResourceDiff.SecretOnly
// is precomputed by ComputeDiff; here we set it directly to exercise the guard logic.
// (That ComputeDiff sets it correctly is covered by TestComputeDiff_Secret.)
func TestDiff_HasNonsecretDiff(t *testing.T) {
	secretOnly := &differ.Diff{UpdatedResources: map[string]differ.ResourceDiff{
		"some-type:r0": {URN: "some-type:r0", SecretOnly: true,
			Diffs: map[string]differ.PropertyDiff{"token": {SecretOnly: true}}},
	}}
	assert.True(t, secretOnly.HasDiff(), "a secret-only resource is still a diff (it re-applies)")
	assert.False(t, secretOnly.HasNonSecretDiff(), "but it is not real drift")

	mixed := &differ.Diff{UpdatedResources: map[string]differ.ResourceDiff{
		"some-type:r0": {URN: "some-type:r0", SecretOnly: false,
			Diffs: map[string]differ.PropertyDiff{"token": {SecretOnly: true}, "name": {}}},
	}}
	assert.True(t, mixed.HasNonSecretDiff())

	assert.True(t, (&differ.Diff{NewResources: []string{"x"}}).HasNonSecretDiff())
	assert.True(t, (&differ.Diff{RemovedResources: []string{"x"}}).HasNonSecretDiff())
}

// secretRawData is a typed RawData struct that adopts a secret. The field is a
// *secret.String, mirroring *PropertyRef: a pointer is the form that survives the
// struct→map decode the differ relies on (see TestSecret_SurvivesStructToMap).
type secretRawData struct {
	Name  string
	Token *secret.String
}

// TestSecret_SurvivesStructToMap locks down the decode-preservation contract: the
// struct→map step the differ performs on RawData must not flatten the concrete
// secret type, otherwise the secret-aware rules can never fire.
func TestSecret_SurvivesStructToMap(t *testing.T) {
	t.Run("pointer secret is preserved as concrete type", func(t *testing.T) {
		tok := secret.New("hunter2")
		var out map[string]any
		require.NoError(t, mapstructure.Decode(secretRawData{Name: "main", Token: &tok}, &out))

		got, ok := out["Token"].(*secret.String)
		require.True(t, ok, "expected *secret.String, got %T", out["Token"])
		assert.Equal(t, "hunter2", got.Reveal())
	})

	// A value secret.String has no exported fields, so mapstructure decomposes it
	// into an empty map rather than preserving it. This is why adopters use a
	// *secret.String on the RawData path, exactly as PropertyRef uses a pointer.
	t.Run("value secret does not survive, justifying the pointer contract", func(t *testing.T) {
		type valueRawData struct {
			Token secret.String
		}
		var out map[string]any
		require.NoError(t, mapstructure.Decode(valueRawData{Token: secret.New("hunter2")}, &out))

		_, ok := out["Token"].(secret.String)
		assert.False(t, ok, "value secret.String unexpectedly survived as a concrete type")
	})
}

// TestComputeDiff_Secret exercises the full RawData path end to end: typed structs
// carrying a *secret.String are decoded to maps and compared by the differ.
func TestComputeDiff_Secret(t *testing.T) {
	rawWith := func(s secret.String) *secretRawData {
		return &secretRawData{Name: "main", Token: &s}
	}
	resWith := func(id string, s secret.String) *resources.Resource {
		return resources.NewResource(id, "some-type", resources.ResourceData{}, []string{}, resources.WithRawData(rawWith(s)))
	}

	t.Run("unknown remote forces an always-re-applied, secret-only update", func(t *testing.T) {
		local := resources.NewGraph()
		remote := resources.NewGraph()
		local.AddResource(resWith("r0", secret.New("hunter2")))
		remote.AddResource(resWith("r0", secret.NewUnknown()))

		diff := differ.ComputeDiff(local, remote, differ.DiffOptions{})
		require.Contains(t, diff.UpdatedResources, "some-type:r0")
		assert.True(t, diff.UpdatedResources["some-type:r0"].IsSecretOnly())
		assert.False(t, diff.HasNonSecretDiff(), "a secret-only update is not real drift")
		assert.NotContains(t, diff.UnmodifiedResources, "some-type:r0")
	})

	t.Run("real field change alongside a secret is a real diff", func(t *testing.T) {
		local := resources.NewGraph()
		remote := resources.NewGraph()
		local.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{}, []string{},
			resources.WithRawData(&secretRawData{Name: "local", Token: ptr(secret.New("hunter2"))})))
		remote.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{}, []string{},
			resources.WithRawData(&secretRawData{Name: "remote", Token: ptr(secret.NewUnknown())})))

		diff := differ.ComputeDiff(local, remote, differ.DiffOptions{})
		require.Contains(t, diff.UpdatedResources, "some-type:r0")
		assert.False(t, diff.UpdatedResources["some-type:r0"].IsSecretOnly())
		assert.True(t, diff.HasNonSecretDiff())
	})
}

func ptr(s secret.String) *secret.String { return &s }
