package validate

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// fakeProject stands in for a loaded project. A real one renders its document
// to stdout during Load, which writes the same bytes here.
type fakeProject struct {
	project.Project
	loadErr error
	specs   map[string]*specs.Spec
	doc     string
	out     *bytes.Buffer
}

func (f *fakeProject) Load(string) error {
	f.out.WriteString(f.doc)
	return f.loadErr
}

func (f *fakeProject) Specs() map[string]*specs.Spec { return f.specs }

type validationFailure struct{}

func (validationFailure) Error() string        { return "semantic validation failed" }
func (validationFailure) Is(target error) bool { return target == project.ErrValidationFailed }

func TestRun(t *testing.T) {
	t.Parallel()

	legacy := map[string]*specs.Spec{"a.yaml": {Version: specs.SpecVersionV0_1}}

	tests := []struct {
		name       string
		json       bool
		loadErr    error
		specs      map[string]*specs.Spec
		wantErr    string
		wantSilent bool
		wantStdout string
		wantStderr string
	}{
		{
			name:       "json: a clean run leaves stdout to the document",
			json:       true,
			wantStdout: "{doc}",
		},
		{
			name:       "json: a legacy spec notice goes to stderr",
			json:       true,
			specs:      legacy,
			wantStdout: "{doc}",
			wantStderr: "v0.1 spec format is deprecated",
		},
		{
			name:       "json: a validation failure is not printed again",
			json:       true,
			loadErr:    validationFailure{},
			wantErr:    "validating project: semantic validation failed",
			wantSilent: true,
			wantStdout: "{doc}",
		},
		{
			name:       "json: an error the document does not hold stays on the error path",
			json:       true,
			loadErr:    errors.New("building resource graph: cycle"),
			wantErr:    "validating project: building resource graph: cycle",
			wantStdout: "{doc}",
		},
		{
			name:       "text: a validation failure is returned for printing",
			loadErr:    validationFailure{},
			wantErr:    "validating project: semantic validation failed",
			wantStdout: "{doc}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out, errOut bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&out)
			cmd.SetErr(&errOut)

			err := run(cmd, &fakeProject{loadErr: tt.loadErr, specs: tt.specs, doc: "{doc}", out: &out}, ".", tt.json)

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
				var silent *cmderrors.SilentError
				assert.Equal(t, tt.wantSilent, errors.As(err, &silent), fmt.Sprintf("silent=%v", tt.wantSilent))
			}
			assert.Equal(t, tt.wantStdout, out.String())
			assert.Contains(t, errOut.String(), tt.wantStderr)
			if tt.wantStderr == "" {
				assert.Empty(t, errOut.String())
			}
		})
	}
}
