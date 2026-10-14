package hello

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdHello(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantOut    string
		wantErrMsg string
	}{
		{name: "lowercase claude", args: []string{"claude"}, wantOut: "Hello, Claude!\n"},
		{name: "capitalised claude", args: []string{"Claude"}, wantOut: "Hello, Claude!\n"},
		{name: "unsupported name", args: []string{"bob"}, wantErrMsg: `unsupported name "bob": only "claude" is supported`},
		{name: "no args", args: []string{}, wantErrMsg: "accepts 1 arg(s), received 0"},
		{name: "too many args", args: []string{"claude", "bob"}, wantErrMsg: "accepts 1 arg(s), received 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCmdHello()
			cmd.SetOut(&out)
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.String())
		})
	}
}
