package varsubst

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type mapResolver map[string]string

func (m mapResolver) Resolve(name string) (string, bool) {
	v, ok := m[name]
	return v, ok
}

func TestSubstituteBytes(t *testing.T) {
	tests := []struct {
		name      string
		resolvers []Resolver
		input     string
		wantData  string
		wantErrs  []SubstitutionError
	}{
		{
			name:      "single variable substitution",
			resolvers: []Resolver{mapResolver{"HOST": "localhost"}},
			input:     "host: {{ .HOST }}",
			wantData:  "host: localhost",
		},
		{
			name:      "multiple variables in one file",
			resolvers: []Resolver{mapResolver{"HOST": "localhost", "PORT": "5432"}},
			input:     "host: {{ .HOST }}\nport: {{ .PORT }}",
			wantData:  "host: localhost\nport: 5432",
		},
		{
			name:      "variable with default used when unresolved",
			resolvers: []Resolver{mapResolver{}},
			input:     "host: {{ .HOST | localhost }}",
			wantData:  "host: localhost",
		},
		{
			name:      "variable with default resolver takes precedence",
			resolvers: []Resolver{mapResolver{"HOST": "prod.example.com"}},
			input:     "host: {{ .HOST | localhost }}",
			wantData:  "host: prod.example.com",
		},
		{
			name:      "multiple tokens on one line",
			resolvers: []Resolver{mapResolver{"HOST": "localhost", "PORT": "5432", "DB": "mydb"}},
			input:     "url: https://{{ .HOST }}:{{ .PORT }}/{{ .DB }}",
			wantData:  "url: https://localhost:5432/mydb",
		},
		{
			name:      "token in YAML comment skipped",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     "# comment {{ .VAR }}",
			wantData:  "# comment {{ .VAR }}",
		},
		{
			name:      "token after inline comment skipped",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     "value # {{ .VAR }}",
			wantData:  "value # {{ .VAR }}",
		},
		{
			name:      "hash inside double quotes is not a comment",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     `"#not-comment" {{ .VAR }}`,
			wantData:  `"#not-comment" value`,
		},
		{
			name:      "hash inside single quotes is not a comment",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     `'#hash-in-single-quotes' {{ .VAR }}`,
			wantData:  `'#hash-in-single-quotes' value`,
		},
		{
			name:      "empty resolved value auto-quoted",
			resolvers: []Resolver{mapResolver{"HOST": ""}},
			input:     "host: {{ .HOST }}",
			wantData:  `host: ""`,
		},
		{
			name:      "empty resolved value not auto-quoted inside double quotes",
			resolvers: []Resolver{mapResolver{"HOST": ""}},
			input:     `host: "{{ .HOST }}"`,
			wantData:  `host: ""`,
		},
		{
			name:      "empty resolved value not auto-quoted inside single quotes",
			resolvers: []Resolver{mapResolver{"HOST": ""}},
			input:     `host: '{{ .HOST }}'`,
			wantData:  `host: ''`,
		},
		{
			name:      "default value containing closing brace",
			resolvers: []Resolver{mapResolver{}},
			input:     "pattern: {{ .RE | [a-z]{3} }}",
			wantData:  "pattern: [a-z]{3}",
		},
		{
			name:      "whitespace variants",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     "a: {{.VAR}}\nb: {{ .VAR }}\nc: {{  .VAR  }}",
			wantData:  "a: value\nb: value\nc: value",
		},
		{
			name:      "no variables passthrough unchanged",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     "host: localhost\nport: 5432",
			wantData:  "host: localhost\nport: 5432",
		},
		{
			name:      "all undefined collect all errors no partial substitution",
			resolvers: []Resolver{mapResolver{}},
			input:     "host: {{ .HOST }}\nport: {{ .PORT }}",
			wantData:  "host: {{ .HOST }}\nport: {{ .PORT }}",
			wantErrs: []SubstitutionError{
				{Name: "HOST", Line: 1, Column: 7, LineText: "host: {{ .HOST }}", Err: ErrUndefinedVariable},
				{Name: "PORT", Line: 2, Column: 7, LineText: "port: {{ .PORT }}", Err: ErrUndefinedVariable},
			},
		},
		{
			name:      "mixed resolved and undefined",
			resolvers: []Resolver{mapResolver{"HOST": "localhost"}},
			input:     "host: {{ .HOST }}\nport: {{ .PORT }}",
			wantData:  "host: localhost\nport: {{ .PORT }}",
			wantErrs: []SubstitutionError{
				{Name: "PORT", Line: 2, Column: 7, LineText: "port: {{ .PORT }}", Err: ErrUndefinedVariable},
			},
		},
		{
			name: "resolver priority order first resolver wins",
			resolvers: []Resolver{
				mapResolver{"HOST": "from-env"},
				mapResolver{"HOST": "from-file"},
			},
			input:    "host: {{ .HOST }}",
			wantData: "host: from-env",
		},
		{
			name:      "invalid variable name reports error",
			resolvers: []Resolver{mapResolver{}},
			input:     "host: {{ .123BAD }}",
			wantData:  "host: {{ .123BAD }}",
			wantErrs: []SubstitutionError{
				{Name: "123BAD", Line: 1, Column: 7, LineText: "host: {{ .123BAD }}", Err: ErrInvalidVarSyntax},
			},
		},
		{
			name:      "invalid variable name mixed with valid",
			resolvers: []Resolver{mapResolver{"HOST": "localhost"}},
			input:     "host: {{ .HOST }}\nport: {{ .9PORT }}",
			wantData:  "host: localhost\nport: {{ .9PORT }}",
			wantErrs: []SubstitutionError{
				{Name: "9PORT", Line: 2, Column: 7, LineText: "port: {{ .9PORT }}", Err: ErrInvalidVarSyntax},
			},
		},
		{
			// Only dot-prefixed tokens belong to substitution. Anything else is
			// another dialect's syntax (e.g. the RudderStack UI's
			// `{{ path || fallback }}`) and passes through untouched.
			name:      "missing dot prefix passes through untouched",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     "key: {{ VAR }}",
			wantData:  "key: {{ VAR }}",
		},
		{
			name:      "ui template passes through untouched",
			resolvers: []Resolver{mapResolver{"BUCKET": "value"}},
			input:     "bucket_name: {{ config.bucketName || my-bucket }}",
			wantData:  "bucket_name: {{ config.bucketName || my-bucket }}",
		},
		{
			name:      "ui template alongside a variable reference",
			resolvers: []Resolver{mapResolver{"PREFIX": "rudder/"}},
			input:     "a: {{ config.bucketName || my-bucket }}\nb: {{ .PREFIX }}",
			wantData:  "a: {{ config.bucketName || my-bucket }}\nb: rudder/",
		},
		{
			// A malformed *dotted* token is still claimed and still reported: the
			// narrower match must not silence genuine typos in variable names.
			name:      "malformed dotted token still reports error",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     "key: {{ .9BAD }}",
			wantData:  "key: {{ .9BAD }}",
			wantErrs: []SubstitutionError{
				{Name: "9BAD", Line: 1, Column: 6, LineText: "key: {{ .9BAD }}", Err: ErrInvalidVarSyntax},
			},
		},
		{
			name:      "multi-line value inside a plain scalar stays verbatim",
			resolvers: []Resolver{mapResolver{"VALUE": "first\nsecond"}},
			input:     "value: prefix-'{{ .VALUE }}'-suffix",
			wantData:  "value: prefix-'first\nsecond'-suffix",
		},
		{
			name:      "multi-line value after a prefix in a quoted scalar stays verbatim",
			resolvers: []Resolver{mapResolver{"VALUE": "first\nsecond"}},
			input:     "value: 'prefix {{ .VALUE }}'",
			wantData:  "value: 'prefix first\nsecond'",
		},
		{
			name:      "multi-line value before a suffix in a quoted scalar stays verbatim",
			resolvers: []Resolver{mapResolver{"VALUE": "first\nsecond"}},
			input:     "value: '{{ .VALUE }} suffix'",
			wantData:  "value: 'first\nsecond suffix'",
		},
		{
			name:      "multi-line value sharing a quoted scalar with another token stays verbatim",
			resolvers: []Resolver{mapResolver{"VALUE": "first\nsecond", "OTHER": "third"}},
			input:     "value: '{{ .VALUE }} {{ .OTHER }}'",
			wantData:  "value: 'first\nsecond third'",
		},
		{
			name:      "multi-line value inside a plain scalar after a lone CR line ending stays verbatim",
			resolvers: []Resolver{mapResolver{"CERT": "first\n  second"}},
			input:     "a: plain\ractual: 'literal'\nembedded: prefix-'{{ .CERT }}'-suffix\n",
			wantData:  "a: plain\ractual: 'literal'\nembedded: prefix-'first\n  second'-suffix\n",
		},
		{
			name:      "multi-line value whose quotes another value put back verbatim undoes stays verbatim",
			resolvers: []Resolver{mapResolver{"PREFIX": "one]\nother: plain", "CERT": "first\n  second"}},
			input:     "value: [{{ .PREFIX }}, '{{ .CERT }}']",
			wantData:  "value: [one]\nother: plain, 'first\n  second']",
		},
		{
			name:      "multi-line value in a double-quoted scalar continued across lines stays verbatim",
			resolvers: []Resolver{mapResolver{"CERT": "first\nsecond"}},
			input:     "cert: \"\\\n  {{ .CERT }}\"",
			wantData:  "cert: \"\\\n  first\nsecond\"",
		},
		{
			name:      "multi-line value that is not valid UTF-8 stays verbatim",
			resolvers: []Resolver{mapResolver{"VALUE": "first\n\xffsecond"}},
			input:     "value: '{{ .VALUE }}'",
			wantData:  "value: 'first\n\xffsecond'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewSubstitutor(tt.resolvers...)
			got, errs := sub.SubstituteBytes([]byte(tt.input))

			assert.Equal(t, tt.wantData, string(got))
			if tt.wantErrs != nil {
				assert.Equal(t, tt.wantErrs, errs)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

// A multi-line value that fills a quoted scalar keeps its line breaks once the
// spec is parsed.
func TestSubstituteBytes_MultilineWholeScalar(t *testing.T) {
	t.Parallel()

	cert := "first\nsecond"
	tests := []struct {
		name  string
		input string
		vars  mapResolver
		want  map[string]any
	}{
		{name: "single quoted", input: "value: '{{ .VALUE }}'", vars: mapResolver{"VALUE": cert}, want: map[string]any{"value": cert}},
		{name: "double quoted", input: `value: "{{ .VALUE }}"`, vars: mapResolver{"VALUE": cert}, want: map[string]any{"value": cert}},
		{name: "CRLF", input: "value: '{{ .VALUE }}'", vars: mapResolver{"VALUE": "first\r\nsecond\r\n"}, want: map[string]any{"value": "first\r\nsecond\r\n"}},
		{name: "blank line and leading spaces", input: "value: '{{ .VALUE }}'", vars: mapResolver{"VALUE": "first\n\n  indented\n"}, want: map[string]any{"value": "first\n\n  indented\n"}},
		{name: "quotes and backslashes", input: `value: "{{ .VALUE }}"`, vars: mapResolver{"VALUE": "first's \\\"value\"\nsecond"}, want: map[string]any{"value": "first's \\\"value\"\nsecond"}},
		{name: "unicode line breaks and control characters", input: "value: '{{ .VALUE }}'", vars: mapResolver{"VALUE": "a\u0085b\u2028c\u007fd\te"}, want: map[string]any{"value": "a\u0085b\u2028c\u007fd\te"}},
		{name: "flow mapping", input: "{value: '{{ .VALUE }}'}", vars: mapResolver{"VALUE": cert}, want: map[string]any{"value": cert}},
		{name: "nested mapping", input: "outer:\n  value: '{{ .VALUE }}'", vars: mapResolver{"VALUE": cert}, want: map[string]any{"outer": map[string]any{"value": cert}}},
		{name: "tag", input: "value: !!str '{{ .VALUE }}'", vars: mapResolver{"VALUE": cert}, want: map[string]any{"value": cert}},
		{name: "anchor", input: `value: &v "{{ .VALUE }}"`, vars: mapResolver{"VALUE": cert}, want: map[string]any{"value": cert}},
		{name: "tag on the line before", input: "value: !!str\n  '{{ .VALUE }}'", vars: mapResolver{"VALUE": cert}, want: map[string]any{"value": cert}},
		{name: "same variable twice", input: "a: '{{ .VALUE }}'\nb: \"{{ .VALUE }}\"", vars: mapResolver{"VALUE": cert}, want: map[string]any{"a": cert, "b": cert}},
		{name: "bare sibling", input: "value: '{{ .VALUE }}'\ncount: {{ .COUNT }}", vars: mapResolver{"VALUE": cert, "COUNT": "3"}, want: map[string]any{"value": cert, "count": 3}},
		{name: "sibling valid only once substituted", input: "value: '{{ .VALUE }}'\naddress: {{ .HOST }}-{{ .PORT }}", vars: mapResolver{"VALUE": cert, "HOST": "redis", "PORT": "6379"}, want: map[string]any{"value": cert, "address": "redis-6379"}},
		{name: "sibling that injects a key", input: "spec:\n  {{ .EXTRA }}\n  value: '{{ .VALUE }}'", vars: mapResolver{"EXTRA": "name: redis", "VALUE": cert}, want: map[string]any{"spec": map[string]any{"name": "redis", "value": cert}}},
		{name: "multi-line sibling that opens the flow sequence holding the value", input: "value: {{ .PREFIX }}, '{{ .VALUE }}']", vars: mapResolver{"PREFIX": "one\nother: [two", "VALUE": cert}, want: map[string]any{"value": "one", "other": []any{"two", cert}}},
		{name: "multi-line sibling that injects keys", input: "spec:\n  {{ .EXTRA }}\n  value: '{{ .VALUE }}'", vars: mapResolver{"EXTRA": "name: redis\n  enabled: true", "VALUE": cert}, want: map[string]any{"spec": map[string]any{"name": "redis", "enabled": true, "value": cert}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, errs := NewSubstitutor(tt.vars).SubstituteBytes([]byte(tt.input))
			require.Empty(t, errs)

			var parsed map[string]any
			require.NoError(t, yaml.Unmarshal(data, &parsed))
			assert.Equal(t, tt.want, parsed)
		})
	}
}

// Every escape strconv.Quote writes means the same in a YAML double-quoted
// scalar. U+0000-U+02FF covers every ASCII and Latin-1 control character; the
// rest are the Unicode line and paragraph separators, the BOM, noncharacters
// and runes outside the BMP.
func TestSubstituteBytes_MultilineValueRoundTrips(t *testing.T) {
	t.Parallel()

	runes := []rune{0x2028, 0x2029, 0xFEFF, 0xFFFE, 0xFFFF, 0x1F600, 0x10FFFF}
	for r := rune(0); r < 0x300; r++ {
		runes = append(runes, r)
	}
	for _, r := range runes {
		value := "first\n" + string(r) + "second"
		data, errs := NewSubstitutor(mapResolver{"VALUE": value}).SubstituteBytes([]byte(`value: "{{ .VALUE }}"`))
		require.Empty(t, errs, "U+%04X", r)

		var parsed map[string]string
		require.NoError(t, yaml.Unmarshal(data, &parsed), "U+%04X", r)
		assert.Equal(t, map[string]string{"value": value}, parsed, "U+%04X", r)
	}
}

// A one-line value keeps its quote-style semantics, and so does a multi-line
// value apart from its line breaks.
func TestSubstituteBytes_QuotedSlotSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		value string
		want  string
	}{
		{name: "single-quoted slot keeps a literal backslash-n", input: "value: '{{ .VALUE }}'", value: `first\nsecond`, want: `first\nsecond`},
		{name: "double-quoted slot turns a literal backslash-n into a line break", input: `value: "{{ .VALUE }}"`, value: `first\nsecond`, want: "first\nsecond"},
		{name: "single-quoted slot reads a doubled quote in a multi-line value as one", input: "value: '{{ .VALUE }}'", value: "it''s\nsecond", want: "it's\nsecond"},
		{name: "double-quoted slot keeps a doubled quote in a multi-line value", input: `value: "{{ .VALUE }}"`, value: "it''s\nsecond", want: "it''s\nsecond"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, errs := NewSubstitutor(mapResolver{"VALUE": tt.value}).SubstituteBytes([]byte(tt.input))
			require.Empty(t, errs)

			var parsed map[string]string
			require.NoError(t, yaml.Unmarshal(data, &parsed))
			assert.Equal(t, map[string]string{"value": tt.want}, parsed)
		})
	}
}

// A multi-line value in a quoted fragment of a longer quoted scalar is not the
// whole scalar, so it goes in verbatim and folds as it always has.
func TestSubstituteBytes_MultilineQuotedFragmentsInQuotedScalarsRemainEmbedded(t *testing.T) {
	t.Parallel()

	value := "first\nsecond"
	tests := []struct {
		name       string
		input      string
		want       string
		wantParsed string
	}{
		{name: "single quotes inside double quoted scalar", input: `value: "prefix '{{ .VALUE }}' suffix"`, want: `value: "prefix 'first` + "\n" + `second' suffix"`, wantParsed: "prefix 'first second' suffix"},
		{name: "double quotes inside single quoted scalar", input: `value: 'prefix "{{ .VALUE }}" suffix'`, want: `value: 'prefix "first` + "\n" + `second" suffix'`, wantParsed: `prefix "first second" suffix`},
		{name: "escaped double quotes", input: `value: "prefix \"{{ .VALUE }}\" suffix"`, want: `value: "prefix \"first` + "\n" + `second\" suffix"`, wantParsed: `prefix "first second" suffix`},
		{name: "flow collection", input: `{value: prefix-'{{ .VALUE }}'-suffix}`, want: `{value: prefix-'first` + "\n" + `second'-suffix}`, wantParsed: `prefix-'first second'-suffix`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, errs := NewSubstitutor(mapResolver{"VALUE": value}).SubstituteBytes([]byte(tt.input))
			require.Empty(t, errs)
			assert.Equal(t, tt.want, string(data))

			var parsed map[string]string
			require.NoError(t, yaml.Unmarshal(data, &parsed))
			assert.Equal(t, map[string]string{"value": tt.wantParsed}, parsed)
		})
	}
}

func TestIsInComment(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		matchStart int
		want       bool
	}{
		{
			name:       "hash comment",
			line:       "# comment {{ .VAR }}",
			matchStart: 10,
			want:       true,
		},
		{
			name:       "hash in double quotes not a comment",
			line:       `"#not-comment" {{ .VAR }}`,
			matchStart: 15,
			want:       false,
		},
		{
			name:       "inline comment",
			line:       "value # {{ .VAR }}",
			matchStart: 8,
			want:       true,
		},
		{
			name:       "hash in single quotes not a comment",
			line:       "'#hash-in-single-quotes' {{ .VAR }}",
			matchStart: 25,
			want:       false,
		},
		{
			name:       "escaped quote inside double quotes does not reopen string",
			line:       `key: "val\"ue" # {{ .VAR }}`,
			matchStart: 17,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isInComment([]byte(tt.line), tt.matchStart))
		})
	}
}

func TestSubstituteBytes_ErrorPositions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErrs []SubstitutionError
	}{
		{
			name:  "error on first line",
			input: "host: {{ .HOST }}",
			wantErrs: []SubstitutionError{
				{Name: "HOST", Line: 1, Column: 7, LineText: "host: {{ .HOST }}", Err: ErrUndefinedVariable},
			},
		},
		{
			name:  "error on last line",
			input: "first: value\nsecond: {{ .VAR }}",
			wantErrs: []SubstitutionError{
				{Name: "VAR", Line: 2, Column: 9, LineText: "second: {{ .VAR }}", Err: ErrUndefinedVariable},
			},
		},
		{
			name:  "multiple errors each has correct position",
			input: "a: {{ .X }}\nb: {{ .Y }}\nc: {{ .Z }}",
			wantErrs: []SubstitutionError{
				{Name: "X", Line: 1, Column: 4, LineText: "a: {{ .X }}", Err: ErrUndefinedVariable},
				{Name: "Y", Line: 2, Column: 4, LineText: "b: {{ .Y }}", Err: ErrUndefinedVariable},
				{Name: "Z", Line: 3, Column: 4, LineText: "c: {{ .Z }}", Err: ErrUndefinedVariable},
			},
		},
	}

	sub := NewSubstitutor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := sub.SubstituteBytes([]byte(tt.input))
			assert.Equal(t, tt.wantErrs, errs)
		})
	}
}

// An error's line breaks where the YAML parser breaks lines, so a file with CR
// or CRLF endings gets the line the other diagnostics use, and only that
// line's text.
func TestSubstituteBytes_ErrorLinesBreakAtEveryLineEnding(t *testing.T) {
	t.Parallel()

	for name, lineBreak := range map[string]string{"LF": "\n", "CRLF": "\r\n", "CR": "\r"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input := strings.Join([]string{"a: 1", "b: 2", "port: {{ .PORT }}", "c: 3"}, lineBreak)
			_, errs := NewSubstitutor(mapResolver{}).SubstituteBytes([]byte(input))
			require.Equal(t, []SubstitutionError{
				{Name: "PORT", Line: 3, Column: 7, LineText: "port: {{ .PORT }}", Err: ErrUndefinedVariable},
			}, errs)
		})
	}
}
