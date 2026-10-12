package varsubst

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A generated slot is '{{ .VAR }}', and the value is inserted as is. A lone
// quote in the value used to end the scalar and surface as a YAML error far
// from the variable. Now the substitution says which variable and why.
func TestSubstituteBytes_SingleQuotedSlotRejectsALoneQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		value   string
		want    string
		wantErr bool
	}{
		{name: "lone quote in a single-quoted slot", input: "password: '{{ .PW }}'\n", value: "it's-a-secret", wantErr: true},
		{name: "three quotes in a row", input: "password: '{{ .PW }}'\n", value: "a'''b", wantErr: true},
		{name: "doubled quote is a valid escape", input: "password: '{{ .PW }}'\n", value: "it''s-a-secret", want: "password: 'it''s-a-secret'\n"},
		{name: "no quote", input: "password: '{{ .PW }}'\n", value: "plain", want: "password: 'plain'\n"},
		{name: "double-quoted slot is not checked", input: "password: \"{{ .PW }}\"\n", value: "it's-a-secret", want: "password: \"it's-a-secret\"\n"},
		{name: "bare slot is not checked", input: "password: {{ .PW }}\n", value: "it's-a-secret", want: "password: it's-a-secret\n"},
		{name: "quote inside a double-quoted string that precedes the slot", input: "a: \"it's\" # '{{ .PW }}'\n", value: "x'y", want: "a: \"it's\" # '{{ .PW }}'\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, errs := NewSubstitutor(mapResolver{"PW": tt.value}).SubstituteBytes([]byte(tt.input))

			if tt.wantErr {
				require.Len(t, errs, 1)
				assert.ErrorIs(t, &errs[0], ErrUnescapedSingleQuote)
				assert.Equal(t, "PW", errs[0].Name)
				return
			}
			require.Empty(t, errs)
			assert.Equal(t, tt.want, string(got))
		})
	}
}
