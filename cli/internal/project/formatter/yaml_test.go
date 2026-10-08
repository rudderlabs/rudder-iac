package formatter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYAMLFormatter_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       any
		expected    []byte
		expectError bool
	}{
		{
			name: "simple map",
			input: map[string]interface{}{
				"name":  "test",
				"value": 42,
			},
			expected: []byte(heredoc.Doc(`
name: "test"
value: 42
`)),
		},
		{
			name: "nested map",
			input: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": "value",
					"num":   100,
				},
			},
			expected: []byte(heredoc.Doc(`
parent:
  child: "value"
  num: 100
`)),
		},
		{
			name: "string quoting",
			input: map[string]interface{}{
				"str1": "hello",
				"str2": "world",
				"num":  123,
			},
			expected: []byte(heredoc.Doc(`
num: 123
str1: "hello"
str2: "world"
`)),
		},
		{
			name:  "empty map",
			input: map[string]interface{}{},
			expected: []byte(heredoc.Doc(`{}
`)),
		},
		{
			name: "complex nested",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "example",
					"labels": map[string]interface{}{
						"env": "prod",
					},
				},
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			expected: []byte(heredoc.Doc(`
metadata:
  name: "example"
  labels:
    env: "prod"
spec:
  replicas: 3
`)),
		},
		{
			name: "array of strings",
			input: map[string]interface{}{
				"items": []string{"first", "second", "third"},
			},
			expected: []byte(heredoc.Doc(`
items:
  - "first"
  - "second"
  - "third"
`)),
		},
		{
			name: "mixed types",
			input: map[string]interface{}{
				"string": "text",
				"int":    42,
				"float":  3.14,
				"bool":   true,
			},
			expected: []byte(heredoc.Doc(`
bool: true
float: 3.14
int: 42
string: "text"
`)),
		},
		{
			name: "deep nesting",
			input: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "deep",
					},
				},
			},
			expected: []byte(heredoc.Doc(`
level1:
  level2:
    level3: "deep"
`)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			formatter := YAMLFormatter{}
			output, err := formatter.Format(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.YAMLEq(t, strings.TrimSpace(string(tt.expected)), string(output))
			}
		})
	}
}

// Variable substitution tokens must come out single-quoted. A bare "{{ .VAR }}"
// slot turns a JSON-blob secret into a YAML flow mapping once substituted, so
// the field becomes a map instead of a string. Tokens embedded in larger
// strings stay double-quoted: only a whole-scalar token is a reference.
func TestYAMLFormatter_SingleQuotesVariableTokens(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"accessKey": "{{ .BOOKS_ACCESS_KEY }}",
		"items":     []string{"{{ .ITEM_TOKEN }}"},
		"partial":   "prefix {{ .EMBEDDED }} suffix",
	}

	output, err := YAMLFormatter{}.Format(input)
	require.NoError(t, err)

	assert.Equal(t, heredoc.Doc(`
		accessKey: '{{ .BOOKS_ACCESS_KEY }}'
		items:
		  - '{{ .ITEM_TOKEN }}'
		partial: "prefix {{ .EMBEDDED }} suffix"
	`), string(output))
}

type mapResolver map[string]string

func (m mapResolver) Resolve(name string) (string, bool) {
	v, ok := m[name]
	return v, ok
}

// The generated slot must survive substitution and YAML parsing as a string
// for a JSON value, whether the var file holds it compact or pretty-printed.
// Substitution inserts values verbatim, so this is the end-to-end contract.
func TestYAMLFormatter_JSONSecretSlotParsesAsString(t *testing.T) {
	t.Parallel()

	// Dummy PEM markers with a fake body, built by concatenation so the fixture
	// is not mistaken for a real key. The \n sequences are literal backslash-n
	// inside the JSON string, as in a real service account key file.
	const (
		pemBegin = "-----BEGIN " + "PRIVATE KEY-----"
		pemEnd   = "-----END " + "PRIVATE KEY-----"
		pemBody  = pemBegin + `\nabc\ndef\n` + pemEnd + `\n`
	)

	//nolint:gosec // fixture, not a credential
	const prettyKey = `{
  "type": "service_account",
  "project_id": "demo",
  "private_key": "` + pemBody + `"
}`
	//nolint:gosec // fixture, not a credential
	const compactKey = `{"type":"service_account","project_id":"demo","private_key":"` + pemBody + `"}`

	tests := []struct {
		name  string
		value string
	}{
		{name: "pretty-printed multi-line JSON", value: prettyKey},
		{name: "compact JSON", value: compactKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			generated, err := YAMLFormatter{}.Format(map[string]any{
				"credentials": "{{ .BQ_CREDENTIALS }}",
			})
			require.NoError(t, err)

			substituted, errs := varsubst.NewSubstitutor(
				mapResolver{"BQ_CREDENTIALS": tt.value},
			).SubstituteBytes(generated)
			require.Empty(t, errs)

			var parsed map[string]any
			require.NoError(t, yaml.Unmarshal(substituted, &parsed))

			creds, ok := parsed["credentials"].(string)
			require.True(t, ok, "credentials must parse as a string, got %T", parsed["credentials"])

			var got, want map[string]any
			require.NoError(t, json.Unmarshal([]byte(creds), &got))
			require.NoError(t, json.Unmarshal([]byte(tt.value), &want))
			assert.Equal(t, want, got)
		})
	}
}

func TestYAMLFormatter_Extension(t *testing.T) {
	t.Parallel()
	formatter := YAMLFormatter{}
	assert.Equal(t, []string{"yaml", "yml"}, formatter.Extension())
}

func TestYAMLFormatter_StringQuotingBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected []byte
	}{
		{
			name: "keys unquoted values quoted",
			input: map[string]interface{}{
				"mykey": "myvalue",
			},
			expected: []byte(heredoc.Doc(`
mykey: "myvalue"
`)),
		},
		{
			name: "nested keys unquoted",
			input: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
			expected: []byte(heredoc.Doc(`
outer:
  inner: "value"
`)),
		},
		{
			name: "numbers not quoted",
			input: map[string]interface{}{
				"count":   10,
				"percent": 99.5,
			},
			expected: []byte(heredoc.Doc(`
count: 10
percent: 99.5
`)),
		},
		{
			name: "booleans not quoted",
			input: map[string]interface{}{
				"enabled":  true,
				"disabled": false,
			},
			expected: []byte(heredoc.Doc(`
disabled: false
enabled: true
`)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			formatter := YAMLFormatter{}
			output, err := formatter.Format(tt.input)
			require.NoError(t, err)
			if tt.name == "numbers not quoted" || tt.name == "booleans not quoted" {
				// Map key order is not guaranteed; assert content instead of full equality
				outStr := string(output)
				for _, want := range []string{"count: 10", "percent: 99.5", "disabled: false", "enabled: true"} {
					if strings.Contains(string(tt.expected), want) {
						assert.Contains(t, outStr, want)
					}
				}
			} else {
				assert.Equal(t, tt.expected, output)
			}
		})
	}
}

func TestYAMLFormatter_Format_PreservesNodeHeadComment(t *testing.T) {
	t.Parallel()

	var node yaml.Node
	require.NoError(t, node.Encode(map[string]any{"kind": "import-manifest"}))
	node.HeadComment = " generated — do not edit"

	out, err := YAMLFormatter{}.Format(&node)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(out), "#"), "got:\n%s", out)
	assert.Contains(t, string(out), "generated — do not edit")
	assert.Contains(t, string(out), `kind: "import-manifest"`)
}
