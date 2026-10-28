package specdocs_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/builtin"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/specdocs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func newDestinations(t *testing.T) (*specdocs.Destinations, *definitions.Registry) {
	t.Helper()

	registry, err := builtin.NewRegistry(true)
	require.NoError(t, err)
	return specdocs.NewDestinations(registry), registry
}

func ptr[T any](v T) *T { return &v }

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
		doc := describe(t, docs, specdocs.DescribeOptions{Type: registered.Type, Version: &registered.Version, ID: ptr(id)})

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
		{
			// api_version defaults to v2, which requires user_id_identifier_type.
			destType: "customerio",
			id:       "storefront-customerio",
			wantNull: []string{"datacenter", "user_id_identifier_type"},
			fill:     map[string]any{"datacenter": "US", "user_id_identifier_type": "id"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.destType, func(t *testing.T) {
			t.Parallel()

			doc := describe(t, docs, specdocs.DescribeOptions{Type: tc.destType, Source: ptr("javascript"), ID: ptr(tc.id)})
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

// Across every type and every source it accepts, a caller fills each null and
// secret slot of the template from the document alone, then adds each key a
// requiredIf, requiredUnless or requiredWith rule asks for given those values,
// as the template's first lines say. Booleans are filled with false and then
// with true, to open both branches they decide. Validate accepts the result
// with no error, envelope included, except in customValidatorBranches.
func TestFilledSpecTemplatesValidate(t *testing.T) {
	t.Parallel()

	docs, registry := newDestinations(t)
	rule := destination.NewSpecSyntaxValidRule(registry)

	for _, registered := range registry.Definitions() {
		for _, sourceType := range sortedSourceTypes() {
			doc, err := docs.Describe(specdocs.DescribeOptions{
				Type: registered.Type, Version: &registered.Version, Source: ptr(sourceType), ID: ptr("dest-id"),
			})
			var docErr *specdocs.Error
			if errors.As(err, &docErr) && docErr.Code == specdocs.CodeUnsupportedSource {
				continue
			}
			require.NoError(t, err)

			for _, boolean := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s from %s with %t", registered.Type, sourceType, boolean), func(t *testing.T) {
					t.Parallel()

					fill := filler{doc: doc, boolean: boolean}
					spec := templateSpec(t, doc.SpecTemplate)
					for key, value := range spec.config {
						if value == nil || isSlot(value) {
							spec.config[key] = fill.value(t, fill.fields()[key])
						}
					}
					if added := fill.followConditions(t, spec.config); len(added) > 0 {
						t.Logf("the filled values require %v", added)
					}

					results := validateSpec(rule, spec)
					if boolean || !slices.Contains(customValidatorBranches, registered.Type) {
						assert.Empty(t, results)
						return
					}
					assert.NotEmpty(t, results, "remove %s from customValidatorBranches", registered.Type)
					for _, result := range results {
						key := strings.TrimPrefix(result.Reference, "/spec/config/")
						checkedBy := checkedByAt(doc, strings.ReplaceAll(key, "/", "."))
						assert.NotEmpty(t, checkedBy, "%s: %s", result.Reference, result.Message)
					}
				})
			}
		}
	}
}

// customValidatorBranches are the types whose custom validators require keys
// under conditions the document cannot state, once use_rudder_storage is
// false: postgres needs the bucket provider's credentials, and snowflake
// s3.role_based_auth for AWS. Validate names those keys; the template's note
// says it will.
var customValidatorBranches = []string{"postgres", "snowflake"}

// The template of a type with custom validators says up front that validate
// may name keys it leaves out, which postgres and snowflake do. Parsing the
// template as YAML drops comments, so the other template tests cannot see it.
func TestSpecTemplateNotesCustomValidators(t *testing.T) {
	t.Parallel()

	const (
		header = "# Replace each null with a value. A value you fill in can trigger another key's\n" +
			"# requiredIf, requiredUnless or requiredWith rule, which then requires that key.\n"
		note = "# A custom validator (checkedBy) can require more keys under conditions the\n" +
			"# docs cannot state; validate names them.\n"
	)
	docs, _ := newDestinations(t)

	cases := []struct {
		destType string
		want     string
	}{
		{destType: "postgres", want: header + note + "version: rudder/v1\n"},
		{destType: "snowflake", want: header + note + "version: rudder/v1\n"},
		{destType: "webhook", want: header + "version: rudder/v1\n"},
	}
	for _, tc := range cases {
		t.Run(tc.destType, func(t *testing.T) {
			t.Parallel()

			doc := describe(t, docs, specdocs.DescribeOptions{Type: tc.destType, Source: ptr("javascript"), ID: ptr("shop")})
			assert.True(t, strings.HasPrefix(doc.SpecTemplate, tc.want), "template starts:\n%s", doc.SpecTemplate[:min(len(doc.SpecTemplate), 400)])
		})
	}
}

// checkedByAt returns the custom validators on the field at a dotted config
// path, matching list items by "[]".
func checkedByAt(doc specdocs.DestinationDoc, path string) []string {
	for _, f := range doc.Fields {
		if strings.ReplaceAll(f.Key, "[]", "") == path {
			return f.CheckedBy
		}
	}
	return nil
}

// filler writes the values a caller would from the document alone.
type filler struct {
	doc specdocs.DestinationDoc
	// boolean fills every boolean key.
	boolean bool
}

func (f filler) fields() map[string]specdocs.Field {
	return fieldsByKey(f.doc)
}

// followConditions adds, until none is left, each top-level key whose
// requiredIf, requiredUnless or requiredWith rule holds on the config's values,
// or on the defaults where the config leaves a key out.
func (f filler) followConditions(t *testing.T, config map[string]any) []string {
	t.Helper()

	var (
		fields = f.fields()
		added  []string
	)
	holds := func(c specdocs.Condition) bool {
		value, written := config[c.Key]
		if key := fields[c.Key]; !written && key.Default != nil && !key.DefaultRequiresParent {
			value, written = key.Default, true
		}
		return written && fmt.Sprint(value) == fmt.Sprint(c.Value)
	}
	isWritten := func(key string) bool {
		_, written := config[key]
		return written
	}

	for changed := true; changed; {
		changed = false
		for _, field := range f.doc.Fields {
			if strings.Contains(field.Key, ".") || isWritten(field.Key) {
				continue
			}
			requiredIf := len(field.RequiredIf) > 0 && !slices.ContainsFunc(field.RequiredIf, func(c specdocs.Condition) bool { return !holds(c) })
			requiredUnless := len(field.RequiredUnless) > 0 && !slices.ContainsFunc(field.RequiredUnless, holds)
			if !requiredIf && !requiredUnless && !slices.ContainsFunc(field.RequiredWith, isWritten) {
				continue
			}
			config[field.Key] = f.value(t, field)
			added = append(added, field.Key)
			changed = true
		}
	}
	return added
}

// fakeStrings are tried in order for a string key; the first its pattern and
// limits accept is used.
var fakeStrings = []string{
	"fake-value",
	"https://hooks.example.com/rudder",
	"arn:aws:iam::123456789012:role/fake",
	"G-ABCDEF1234",
	"1:123456789012:web:abcdef1234567890",
	"123456789012",
	"fake@example.com",
	"us-east-1",
	"FakeValue1",
	"9092",
	"UA-123456-1",
	"AW-123456789",
	"-----BEGIN PRIVATE KEY-----\nfake\n-----END PRIVATE KEY-----",
}

// value is a value the field's own rules accept: the first enum value,
// f.boolean, the lower limit of a number, the first of fakeStrings the
// pattern and length limits accept, or a list of one such item, whose keys
// are filled when it is an object.
func (f filler) value(t *testing.T, field specdocs.Field) any {
	t.Helper()

	if itemType, isList := strings.CutPrefix(field.Type, "array<"); isList {
		item := field
		item.Key, item.Type, item.Min, item.Max = field.Key+"[]", strings.TrimSuffix(itemType, ">"), nil, nil
		return []any{f.value(t, item)}
	}
	switch {
	case field.Type == "object":
		object := map[string]any{}
		for _, child := range f.doc.Fields {
			name, isChild := strings.CutPrefix(child.Key, field.Key+".")
			if isChild && !strings.ContainsAny(name, ".[") {
				object[name] = f.value(t, child)
			}
		}
		return object
	case len(field.Enum) > 0:
		return field.Enum[0]
	case field.Type == "boolean":
		return f.boolean
	case field.Type == "integer" || field.Type == "number":
		if field.Min != nil {
			return *field.Min
		}
		return 1
	case field.Type == "string":
		for _, candidate := range fakeStrings {
			if acceptsString(f.doc, field, candidate) {
				return candidate
			}
		}
	}
	require.Failf(t, "no fake value", "%s: type %s, pattern %v", field.Key, field.Type, f.doc.Patterns[field.Pattern])
	return nil
}

func acceptsString(doc specdocs.DestinationDoc, f specdocs.Field, value string) bool {
	if f.Min != nil && len(value) < *f.Min || f.Max != nil && len(value) > *f.Max {
		return false
	}
	if f.Pattern == "" {
		return true
	}
	pattern := doc.Patterns[f.Pattern]
	if pattern.Reject != "" && regexp.MustCompile(pattern.Reject).MatchString(value) {
		return false
	}
	return regexp.MustCompile(pattern.Regex).MatchString(value)
}

// An agent writes consent_management from the JSON document alone: an entry
// for each provider, a resolution_strategy where requiredIf asks for one, and
// a consent value the pattern accepts. Validate accepts that block for every
// type and source token, and rejects a second entry for a provider, as unique
// says.
func TestConsentBlockWrittenFromTheDocumentValidates(t *testing.T) {
	t.Parallel()

	const item = "consent_management.<sourceToken>[]"
	docs, registry := newDestinations(t)

	for _, registered := range registry.Definitions() {
		t.Run(fmt.Sprintf("%s v%d", registered.Type, registered.Version), func(t *testing.T) {
			t.Parallel()

			doc := describeJSON(t, docs, specdocs.DescribeOptions{Type: registered.Type, Version: &registered.Version})
			fields := map[string]map[string]any{}
			for _, f := range doc["fields"].([]any) {
				field := f.(map[string]any)
				fields[field["key"].(string)] = field
			}
			var (
				provider = fields[item+".provider"]
				strategy = fields[item+".resolution_strategy"]
				consents = fields[item+".consents"]
				pattern  = doc["patterns"].(map[string]any)[consents["pattern"].(string)].(map[string]any)
				value    = "analytics"
			)
			require.Regexp(t, pattern["regex"], value)
			assert.Equal(t, true, provider["unique"])

			var entries []any
			for _, name := range provider["enum"].([]any) {
				entry := map[string]any{"provider": name, "consents": []any{value}}
				for _, c := range strategy["requiredIf"].([]any) {
					condition := c.(map[string]any)
					if condition["key"] == item+".provider" && condition["value"] == name {
						entry["resolution_strategy"] = strategy["enum"].([]any)[0]
					}
				}
				entries = append(entries, entry)
			}
			block := map[string]any{}
			for token := range doc["connectionModes"].(map[string]any) {
				block[token] = entries
			}
			assert.Empty(t, consentErrors(registered, block))

			for token := range block {
				block[token] = append(entries, entries[0])
			}
			assert.NotEmpty(t, consentErrors(registered, block), "a second entry for a provider")
		})
	}
}

func consentErrors(registered *definitions.RegisteredDefinition, block map[string]any) []definitions.ConfigError {
	var errors []definitions.ConfigError
	for _, configErr := range registered.ValidateConfig(map[string]any{"consent_management": block}) {
		if strings.HasPrefix(configErr.Path, "/consent_management") {
			errors = append(errors, configErr)
		}
	}
	return errors
}

// Every ID docs accepts writes a template the spec rule accepts, and every ID
// that would not is rejected up front.
func TestIDsMatchTheSpecRule(t *testing.T) {
	t.Parallel()

	docs, registry := newDestinations(t)
	rule := destination.NewSpecSyntaxValidRule(registry)

	for _, id := range []string{"yes", "on", "null", "-x", "Foo_Bar", "ab", strings.Repeat("a", 100)} {
		doc := describe(t, docs, specdocs.DescribeOptions{Type: "webhook", Source: ptr("javascript"), ID: ptr(id)})
		spec := templateSpec(t, doc.SpecTemplate)
		spec.config["webhook_url"] = "https://hooks.example.com/rudder"
		assert.Empty(t, validateSpec(rule, spec), id)
	}

	for _, id := range []string{"a", "_", strings.Repeat("a", 101), "ıdshop", "my shop", "my.shop", "9shop"} {
		_, err := docs.Describe(specdocs.DescribeOptions{Type: "webhook", Source: ptr("javascript"), ID: ptr(id)})
		var docErr *specdocs.Error
		if assert.ErrorAs(t, err, &docErr, id) {
			assert.Equal(t, specdocs.CodeInvalidID, docErr.Code, id)
		}
	}
}

func validateSpec(rule vrules.Rule, spec parsedTemplate) []vrules.ValidationResult {
	return rule.Validate(&vrules.ValidationContext{
		Spec:    spec.document["spec"].(map[string]any),
		Kind:    destination.DestinationSpecKind,
		Version: specs.SpecVersionV1,
	})
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

// isSlot reports whether a template value is a secret's slot, which validate
// substitutes before it checks.
func isSlot(value any) bool {
	s, ok := value.(string)
	return ok && strings.HasPrefix(s, "{{ .")
}

func substituteSlots(config map[string]any, value string) map[string]any {
	out := make(map[string]any, len(config))
	for key, v := range config {
		if isSlot(v) {
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
			{Type: registered.Type, Version: &registered.Version},
			{Type: registered.Type, Version: &registered.Version, ID: ptr("dest-id")},
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
	web := describe(t, docs, specdocs.DescribeOptions{Type: "am", Source: ptr("javascript")})

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

// The JSON contract carries what the walker reads about nested required keys
// and open enums, not only the Markdown cells.
func TestDescribeCarriesNestedRequiredOpenAndEmptyEnums(t *testing.T) {
	t.Parallel()

	docs, _ := newDestinations(t)

	bq := fieldsByKey(describe(t, docs, specdocs.DescribeOptions{Type: "bq"}))
	assert.Equal(t, specdocs.Field{Key: "exclude_window.start_time", Type: "string", RequiredWhenParentPresent: true},
		bq["exclude_window.start_time"])
	assert.Equal(t, specdocs.Field{Key: "exclude_window.end_time", Type: "string", RequiredWhenParentPresent: true},
		bq["exclude_window.end_time"])

	braze := fieldsByKey(describe(t, docs, specdocs.DescribeOptions{Type: "braze"}))
	assert.Equal(t, specdocs.Field{
		Key:      "data_center",
		Type:     "string",
		Required: true,
		Enum: []any{
			"US-01", "US-02", "US-03", "US-04", "US-05", "US-06", "US-07", "US-08",
			"EU-01", "EU-02", "EU-03", "AU-01",
		},
		OpenEnum:        true,
		AcceptsTemplate: true,
	}, braze["data_center"])

	adobe := fieldsByKey(describe(t, docs, specdocs.DescribeOptions{Type: "adobe_analytics"}))
	assert.Contains(t, adobe["events_to_types[].to"].Enum, "")
}

func fieldsByKey(doc specdocs.DestinationDoc) map[string]specdocs.Field {
	fields := make(map[string]specdocs.Field, len(doc.Fields))
	for _, f := range doc.Fields {
		fields[f.Key] = f
	}
	return fields
}

func TestMarkdownExplainsTheListIndexOnlyWhenUsed(t *testing.T) {
	t.Parallel()

	docs, _ := newDestinations(t)
	const note = "`<N>` stands for the item's position in its list, counting from 0."

	var webhook, am bytes.Buffer
	require.NoError(t, specdocs.RenderDestinationMarkdown(&webhook,
		describe(t, docs, specdocs.DescribeOptions{Type: "webhook", ID: ptr("shop-webhook")})))
	require.NoError(t, specdocs.RenderDestinationMarkdown(&am,
		describe(t, docs, specdocs.DescribeOptions{Type: "am", ID: ptr("shop-amplitude")})))

	assert.Contains(t, webhook.String(), note)
	assert.NotContains(t, am.String(), note)
}

func TestDescribeMarksUnverifiedTypes(t *testing.T) {
	t.Parallel()

	docs, _ := newDestinations(t)
	enable := map[string]string{
		"RUDDERSTACK_CLI_EXPERIMENTAL":          "true",
		"RUDDERSTACK_X_UNVERIFIED_DESTINATIONS": "true",
	}

	linkedinAds := describe(t, docs, specdocs.DescribeOptions{Type: "linkedin_ads"})
	assert.False(t, linkedinAds.Verified)
	assert.Equal(t, enable, linkedinAds.Enable)

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

	web, err := docs.List(specdocs.ListOptions{CLIVersion: "test", Source: ptr("javascript")})
	require.NoError(t, err)
	var webTypes []string
	for _, d := range web.Destinations {
		assert.Contains(t, d.SourceTypes, "web", d.Type)
		webTypes = append(webTypes, d.Type)
	}
	assert.NotContains(t, webTypes, "adj")
	assert.Contains(t, webTypes, "webhook")

	// An empty source is not a source type, not a source left out.
	_, err = docs.List(specdocs.ListOptions{CLIVersion: "test", Source: ptr("")})
	var docErr *specdocs.Error
	require.ErrorAs(t, err, &docErr)
	assert.Equal(t, specdocs.CodeUnknownSource, docErr.Code)
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
			opts: specdocs.DescribeOptions{Type: "webhook", Version: ptr[int64](7)},
			want: &specdocs.Error{
				Code:          specdocs.CodeUnknownDefinitionVersion,
				Message:       `destination type "webhook" has no definition version 7`,
				ValidVersions: []int64{1},
			},
		},
		{
			name: "version 0",
			opts: specdocs.DescribeOptions{Type: "webhook", Version: ptr[int64](0)},
			want: &specdocs.Error{
				Code:          specdocs.CodeUnknownDefinitionVersion,
				Message:       `destination type "webhook" has no definition version 0`,
				ValidVersions: []int64{1},
			},
		},
		{
			name: "the version before the source",
			opts: specdocs.DescribeOptions{Type: "webhook", Version: ptr[int64](7), Source: ptr("")},
			want: &specdocs.Error{
				Code:          specdocs.CodeUnknownDefinitionVersion,
				Message:       `destination type "webhook" has no definition version 7`,
				ValidVersions: []int64{1},
			},
		},
		{
			name: "an empty source",
			opts: specdocs.DescribeOptions{Type: "webhook", Source: ptr("")},
			want: &specdocs.Error{
				Code:        specdocs.CodeUnknownSource,
				Message:     `source type "" is not an event stream source spec type`,
				Suggestions: []string{},
				ValidTypes:  sortedSourceTypes(),
			},
		},
		{
			name: "a token is not a source type",
			opts: specdocs.DescribeOptions{Type: "webhook", Source: ptr("web")},
			want: &specdocs.Error{
				Code:        specdocs.CodeUnknownSource,
				Message:     `source type "web" is not an event stream source spec type`,
				Suggestions: []string{"javascript"},
				ValidTypes:  sortedSourceTypes(),
			},
		},
		{
			name: "a source the type does not accept",
			opts: specdocs.DescribeOptions{Type: "adj", Source: ptr("javascript")},
			want: &specdocs.Error{
				Code:    specdocs.CodeUnsupportedSource,
				Message: `destination type "adj" does not accept javascript sources, which connect as "web"`,
				ValidTypes: []string{
					"android", "android_kotlin", "cordova", "dotnet", "flutter", "go", "ios", "ios_swift",
					"java", "node", "php", "python", "react_native", "ruby", "rust", "unity",
				},
			},
		},
		{
			name: "an ID that makes an invalid variable name",
			opts: specdocs.DescribeOptions{Type: "webhook", ID: ptr("9.Shop")},
			want: &specdocs.Error{
				Code: specdocs.CodeInvalidID,
				Message: `--id "9.Shop" gives the secret variable name "9.SHOP", which does not match ^[A-Za-z_][A-Za-z0-9_]*$; ` +
					"use an ID of 2 to 100 letters, digits, underscores and hyphens that does not start with a digit",
			},
		},
		{
			name: "an empty ID",
			opts: specdocs.DescribeOptions{Type: "webhook", Source: ptr("javascript"), ID: ptr("")},
			want: &specdocs.Error{
				Code: specdocs.CodeInvalidID,
				Message: `--id "" is also the spec's display_name, which must be 2-100 characters and contain only ` +
					"letters, digits, underscores, spaces, periods, and hyphens; " +
					"use an ID of 2 to 100 letters, digits, underscores and hyphens that does not start with a digit",
			},
		},
		{
			name: "an ID the display_name rule rejects",
			opts: specdocs.DescribeOptions{Type: "webhook", ID: ptr("ıdshop")},
			want: &specdocs.Error{
				Code: specdocs.CodeInvalidID,
				Message: `--id "ıdshop" is also the spec's display_name, which must be 2-100 characters and contain only ` +
					"letters, digits, underscores, spaces, periods, and hyphens; " +
					"use an ID of 2 to 100 letters, digits, underscores and hyphens that does not start with a digit",
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
