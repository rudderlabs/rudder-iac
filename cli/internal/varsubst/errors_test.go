package varsubst

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstitutionError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *SubstitutionError
		wantMsg string
	}{
		{
			name: "with variable name",
			err: &SubstitutionError{
				Name:     "DB_PASSWORD",
				Line:     5,
				Column:   12,
				LineText: "  password: {{ .DB_PASSWORD }}",
				Err:      ErrUndefinedVariable,
			},
			wantMsg: "line 5, column 12: undefined variable: DB_PASSWORD",
		},
		{
			name: "without variable name",
			err: &SubstitutionError{
				Line:     3,
				Column:   8,
				LineText: "  host: {{ .123BAD }}",
				Err:      ErrInvalidVarSyntax,
			},
			wantMsg: "line 3, column 8: invalid variable syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestSubstitutionError_Unwrap(t *testing.T) {
	err := &SubstitutionError{
		Name: "HOST",
		Line: 1,
		Err:  ErrUndefinedVariable,
	}

	assert.True(t, errors.Is(err, ErrUndefinedVariable))
	assert.False(t, errors.Is(err, ErrInvalidVarSyntax))
}
