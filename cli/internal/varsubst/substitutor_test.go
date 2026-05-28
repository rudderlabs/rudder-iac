package varsubst

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
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
			name:      "missing dot prefix reports error",
			resolvers: []Resolver{mapResolver{"VAR": "value"}},
			input:     "key: {{ VAR }}",
			wantData:  "key: {{ VAR }}",
			wantErrs: []SubstitutionError{
				{Name: "VAR", Line: 1, Column: 6, LineText: "key: {{ VAR }}", Err: ErrInvalidVarSyntax},
			},
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

func TestToDiagnostics(t *testing.T) {
	errs := []SubstitutionError{
		{
			Name:     "DB_PASSWORD",
			Line:     8,
			Column:   15,
			LineText: "    password: {{ .DB_PASSWORD }}",
			Err:      ErrUndefinedVariable,
		},
		{
			Name:     "9PORT",
			Line:     5,
			Column:   11,
			LineText: "    port: {{ .9PORT }}",
			Err:      ErrInvalidVarSyntax,
		},
		{
			Name:     "VAR",
			Line:     10,
			Column:   11,
			LineText: "    value: {{ VAR }}",
			Err:      ErrInvalidVarSyntax,
		},
	}

	got := ToDiagnostics("specs/dest.yaml", errs)

	assert.Equal(t, validation.Diagnostics{
		{
			RuleID:   "project/var-substitution",
			Severity: rules.Error,
			Message:  `undefined variable "DB_PASSWORD"`,
			File:     "specs/dest.yaml",
			Position: pathindex.Position{
				Line:     8,
				Column:   15,
				LineText: "    password: {{ .DB_PASSWORD }}",
			},
		},
		{
			RuleID:   "project/var-substitution",
			Severity: rules.Error,
			Message:  `invalid variable syntax "9PORT"`,
			File:     "specs/dest.yaml",
			Position: pathindex.Position{
				Line:     5,
				Column:   11,
				LineText: "    port: {{ .9PORT }}",
			},
		},
		{
			RuleID:   "project/var-substitution",
			Severity: rules.Error,
			Message:  `invalid variable syntax "VAR"`,
			File:     "specs/dest.yaml",
			Position: pathindex.Position{
				Line:     10,
				Column:   11,
				LineText: "    value: {{ VAR }}",
			},
		},
	}, got)
}
