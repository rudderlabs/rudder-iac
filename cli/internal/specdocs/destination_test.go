package specdocs_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/builtin"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/specdocs"
)

func newDestinations(t *testing.T) (*specdocs.Destinations, *definitions.Registry) {
	t.Helper()

	registry, err := builtin.NewRegistry(true)
	require.NoError(t, err)
	return specdocs.NewDestinations(registry, builtin.IsVerified), registry
}

func describe(t *testing.T, docs *specdocs.Destinations, opts specdocs.DescribeOptions) specdocs.DestinationDoc {
	t.Helper()

	doc, err := docs.Describe(opts)
	require.NoError(t, err)
	return doc
}

// The docs output names each secret's variable before any config exists, and
// import names it from a real config later. Both must agree, or a spec the
// agent writes and one import writes would reference different variables.
func TestSecretVariablesMatchImport(t *testing.T) {
	t.Parallel()

	const id = "storefront-dest"
	docs, registry := newDestinations(t)

	for _, registered := range registry.Definitions() {
		doc := describe(t, docs, specdocs.DescribeOptions{Type: registered.Type, Version: registered.Version, ID: id})

		config := map[string]any{}
		for _, key := range registered.SecretKeys() {
			setPath(t, config, strings.Split(key, "."), registered)
		}
		require.NoError(t, secret.MaskSecrets(config, id, registered.SecretKeys()))

		var secrets int
		for _, field := range doc.Fields {
			if !field.Secret {
				continue
			}
			secrets++
			want := secret.Reference(strings.ReplaceAll(field.Var, "<N>", "0"))
			assert.Equal(t, want, leaf(config, field.Key), "%s %s", registered.Type, field.Key)
			assert.Equal(t, "'"+secret.Reference(field.Var)+"'", field.Slot, "%s %s", registered.Type, field.Key)
		}
		assert.Equal(t, len(registered.SecretKeys()), secrets, registered.Type)
	}
}

// setPath writes a placeholder at a secret path, giving each list it passes
// through one item, so masking reaches it the way it reaches imported config.
func setPath(t *testing.T, config map[string]any, segments []string, registered *definitions.RegisteredDefinition) {
	t.Helper()

	desc := registered.Describe()
	prefix := ""
	current := config
	for i, segment := range segments {
		key := segment
		if prefix != "" {
			key = prefix + "." + segment
		}
		if i == len(segments)-1 {
			current[segment] = "placeholder"
			return
		}

		if isList(desc, key) {
			items, _ := current[segment].([]any)
			if len(items) == 0 {
				items = []any{map[string]any{}}
				current[segment] = items
			}
			current = items[0].(map[string]any)
			prefix = key + "[]"
			continue
		}
		next, _ := current[segment].(map[string]any)
		if next == nil {
			next = map[string]any{}
			current[segment] = next
		}
		current = next
		prefix = key
	}
}

func isList(desc definitions.Description, key string) bool {
	return slices.ContainsFunc(desc.Fields, func(f definitions.Field) bool {
		return strings.HasPrefix(f.Key, key+"[].")
	})
}

func leaf(config map[string]any, key string) any {
	var current any = config
	for _, part := range strings.Split(key, ".") {
		name, isList := strings.CutSuffix(part, "[]")
		current = current.(map[string]any)[name]
		if isList {
			current = current.([]any)[0]
		}
	}
	return current
}

func TestSpecTemplateFailsOnlyOnItsNullKeys(t *testing.T) {
	t.Parallel()

	docs, registry := newDestinations(t)

	cases := []struct {
		destType string
		id       string
		wantNull []string
		fill     map[string]any
	}{
		{
			destType: "webhook",
			id:       "storefront-webhook",
			wantNull: []string{"webhook_url"},
			fill:     map[string]any{"webhook_url": "https://hooks.example.com/rudder"},
		},
		{
			destType: "am",
			id:       "storefront-amplitude",
			wantNull: []string{"residency_server"},
			// The slot stays in place: validate substitutes it before checking.
			fill: map[string]any{"residency_server": "standard"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.destType, func(t *testing.T) {
			t.Parallel()

			doc := describe(t, docs, specdocs.DescribeOptions{Type: tc.destType, Source: "javascript", ID: tc.id})
			spec := templateSpec(t, doc.SpecTemplate)
			assert.Equal(t, map[string]any{
				"version": "rudder/v1",
				"kind":    "destination",
				"metadata": map[string]any{
					"name": tc.id,
				},
				"spec": map[string]any{
					"id":                 tc.id,
					"display_name":       tc.id,
					"type":               tc.destType,
					"definition_version": 1,
					"enabled":            true,
					"config":             spec.config,
				},
			}, spec.document)

			registered, err := registry.Get(tc.destType, 1)
			require.NoError(t, err)

			var failing []string
			for _, configErr := range registered.ValidateConfig(spec.config) {
				failing = append(failing, strings.TrimPrefix(configErr.Path, "/"))
			}
			assert.ElementsMatch(t, tc.wantNull, failing)
			assert.ElementsMatch(t, tc.wantNull, spec.nullKeys())

			for key, value := range tc.fill {
				spec.config[key] = value
			}
			substituted := substituteSlots(spec.config, "fake-secret-value")
			assert.Empty(t, registered.ValidateConfig(substituted))
		})
	}
}

// Across every type and source, a template only ever leaves keys for the
// caller to fill: whatever it writes itself must already pass. Keys it leaves
// out may still be required by a condition, which the caller resolves from
// the fields' requiredIf rules.
func TestSpecTemplateWritesNothingInvalid(t *testing.T) {
	t.Parallel()

	docs, registry := newDestinations(t)

	for _, registered := range registry.Definitions() {
		for _, sourceType := range source.SourceTypes() {
			doc, err := docs.Describe(specdocs.DescribeOptions{Type: registered.Type, Source: sourceType, ID: "dest-id"})
			var docErr *specdocs.Error
			if errors.As(err, &docErr) && docErr.Code == specdocs.CodeUnsupportedSource {
				continue
			}
			require.NoError(t, err)

			spec := templateSpec(t, doc.SpecTemplate)
			toFill := append(spec.nullKeys(), spec.slotKeys()...)
			for _, configErr := range registered.ValidateConfig(spec.config) {
				key, _, _ := strings.Cut(strings.TrimPrefix(configErr.Path, "/"), "/")
				if slices.Contains(toFill, key) {
					continue
				}
				assert.NotContains(t, spec.config, key, "%s from %s: %s", registered.Type, sourceType, configErr.Message)
			}
		}
	}
}

type parsedTemplate struct {
	document map[string]any
	config   map[string]any
}

func templateSpec(t *testing.T, template string) parsedTemplate {
	t.Helper()

	require.NotEmpty(t, template)
	var document map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(template), &document))
	config, ok := document["spec"].(map[string]any)["config"].(map[string]any)
	require.True(t, ok, "spec.config is a mapping")
	return parsedTemplate{document: document, config: config}
}

func (p parsedTemplate) nullKeys() []string {
	var keys []string
	for key, value := range p.config {
		if value == nil {
			keys = append(keys, key)
		}
	}
	return keys
}

func (p parsedTemplate) slotKeys() []string {
	var keys []string
	for key, value := range p.config {
		if s, ok := value.(string); ok && strings.HasPrefix(s, "{{ .") {
			keys = append(keys, key)
		}
	}
	return keys
}

func substituteSlots(config map[string]any, value string) map[string]any {
	out := make(map[string]any, len(config))
	for key, v := range config {
		if s, ok := v.(string); ok && strings.HasPrefix(s, "{{ .") {
			out[key] = value
			continue
		}
		out[key] = v
	}
	return out
}

// A caller iterates the document's lists with jq or a JSON decoder, so none
// may come out null, whatever the type declares.
func TestListsAreNeverNull(t *testing.T) {
	t.Parallel()

	docs, registry := newDestinations(t)

	for _, registered := range registry.Definitions() {
		for _, opts := range []specdocs.DescribeOptions{
			{Type: registered.Type, Version: registered.Version},
			{Type: registered.Type, Version: registered.Version, ID: "dest-id"},
		} {
			doc := describeJSON(t, docs, opts)
			for _, key := range []string{"secretKeys", "fields", "customValidators"} {
				assert.IsType(t, []any{}, doc[key], "%s %s", registered.Type, key)
			}
			for token, modes := range doc["connectionModes"].(map[string]any) {
				assert.IsType(t, []any{}, modes.(map[string]any)["modes"], "%s %s", registered.Type, token)
			}
		}
	}

	list, err := docs.List(specdocs.ListOptions{})
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(encode(t, list), &decoded))
	for _, d := range decoded["destinations"].([]any) {
		summary := d.(map[string]any)
		assert.IsType(t, []any{}, summary["sourceTypes"], summary["type"])
		assert.IsType(t, []any{}, summary["definitionVersions"], summary["type"])
	}
}

func describeJSON(t *testing.T, docs *specdocs.Destinations, opts specdocs.DescribeOptions) map[string]any {
	t.Helper()

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(encode(t, describe(t, docs, opts)), &decoded))
	return decoded
}

func encode(t *testing.T, v any) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, specdocs.WriteJSON(&buf, v))
	return buf.Bytes()
}

func TestDescribeNarrowsToTheSource(t *testing.T) {
	t.Parallel()

	docs, _ := newDestinations(t)
	all := describe(t, docs, specdocs.DescribeOptions{Type: "am"})
	web := describe(t, docs, specdocs.DescribeOptions{Type: "am", Source: "javascript"})

	assert.Equal(t, &specdocs.SourceRef{Type: "javascript", Token: "web"}, web.Source)
	assert.Equal(t, []string{"web"}, mapKeys(web.ConnectionModes))
	assert.Len(t, all.ConnectionModes, 11)
	assert.Less(t, len(web.Fields), len(all.Fields))
	for _, field := range web.Fields {
		if len(field.OnlyForSourceTypes) > 0 {
			assert.Equal(t, []string{"web"}, field.OnlyForSourceTypes, field.Key)
		}
	}
	assert.Empty(t, web.SpecTemplate, "a template needs an ID too")
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func TestDescribeMarksUnverifiedTypes(t *testing.T) {
	t.Parallel()

	docs, _ := newDestinations(t)
	enable := map[string]string{
		"RUDDERSTACK_CLI_EXPERIMENTAL":          "true",
		"RUDDERSTACK_X_UNVERIFIED_DESTINATIONS": "true",
	}

	adj := describe(t, docs, specdocs.DescribeOptions{Type: "adj"})
	assert.False(t, adj.Verified)
	assert.Equal(t, enable, adj.Enable)

	webhook := describe(t, docs, specdocs.DescribeOptions{Type: "webhook"})
	assert.True(t, webhook.Verified)
	assert.Nil(t, webhook.Enable)
}

func TestListFiltersBySource(t *testing.T) {
	t.Parallel()

	docs, registry := newDestinations(t)

	all, err := docs.List(specdocs.ListOptions{CLIVersion: "test"})
	require.NoError(t, err)
	assert.Len(t, all.Destinations, len(registry.SupportedTypes()))

	web, err := docs.List(specdocs.ListOptions{CLIVersion: "test", Source: "javascript"})
	require.NoError(t, err)
	var webTypes []string
	for _, d := range web.Destinations {
		assert.Contains(t, d.SourceTypes, "web", d.Type)
		webTypes = append(webTypes, d.Type)
	}
	assert.NotContains(t, webTypes, "adj")
	assert.Contains(t, webTypes, "webhook")
}

func TestDescribeErrors(t *testing.T) {
	t.Parallel()

	docs, registry := newDestinations(t)

	cases := []struct {
		name string
		opts specdocs.DescribeOptions
		want *specdocs.Error
	}{
		{
			name: "unknown type suggests by display name",
			opts: specdocs.DescribeOptions{Type: "bigquery"},
			want: &specdocs.Error{
				Code:        specdocs.CodeUnknownType,
				Message:     `destination type "bigquery" is not supported`,
				Suggestions: []string{"bq", "bqstream"},
				ValidTypes:  registry.SupportedTypes(),
			},
		},
		{
			name: "unknown version",
			opts: specdocs.DescribeOptions{Type: "webhook", Version: 7},
			want: &specdocs.Error{
				Code:          specdocs.CodeUnknownDefinitionVersion,
				Message:       `destination type "webhook" has no definition version 7`,
				Suggestions:   []string{},
				ValidVersions: []int64{1},
			},
		},
		{
			name: "a token is not a source type",
			opts: specdocs.DescribeOptions{Type: "webhook", Source: "web"},
			want: &specdocs.Error{
				Code:        specdocs.CodeUnknownSource,
				Message:     `source type "web" is not an event stream source spec type`,
				Suggestions: []string{"javascript"},
				ValidTypes:  sortedSourceTypes(),
			},
		},
		{
			name: "a source the type does not accept",
			opts: specdocs.DescribeOptions{Type: "adj", Source: "javascript"},
			want: &specdocs.Error{
				Code:        specdocs.CodeUnsupportedSource,
				Message:     `destination type "adj" does not accept javascript sources, which connect as "web"`,
				Suggestions: []string{},
				ValidTypes: []string{
					"android", "android_kotlin", "cordova", "dotnet", "flutter", "go", "ios", "ios_swift",
					"java", "node", "php", "python", "react_native", "ruby", "rust", "unity",
				},
			},
		},
		{
			name: "an ID that makes an invalid variable name",
			opts: specdocs.DescribeOptions{Type: "webhook", ID: "9.Shop"},
			want: &specdocs.Error{
				Code: specdocs.CodeInvalidID,
				Message: `--id "9.Shop" gives the secret variable name "9.SHOP", which does not match ^[A-Za-z_][A-Za-z0-9_]*$; ` +
					"use an ID of letters, digits, underscores and hyphens that does not start with a digit",
				Suggestions: []string{"shop"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := docs.Describe(tc.opts)
			var docErr *specdocs.Error
			require.ErrorAs(t, err, &docErr)
			assert.Equal(t, tc.want, docErr)
		})
	}
}

func sortedSourceTypes() []string {
	types := source.SourceTypes()
	slices.Sort(types)
	return types
}

func TestListSourcesReadsTheSourceSpec(t *testing.T) {
	t.Parallel()

	list := specdocs.ListSources("test")

	var types []string
	for _, s := range list.Sources {
		types = append(types, s.Type)
	}
	assert.Equal(t, sortedSourceTypes(), types)
	assert.Contains(t, list.Sources, specdocs.SourceRef{Type: "javascript", Token: "web"})
	assert.Contains(t, list.Sources, specdocs.SourceRef{Type: "node", Token: "cloud"})
	assert.Contains(t, list.Sources, specdocs.SourceRef{Type: "react_native", Token: "react_native"})
}
