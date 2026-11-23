package project

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A validation failure the JSON renderer already wrote out is both: it is
// ErrReported, so the caller does not print it again, and ErrValidationFailed,
// so the exit code is still the validation one. Joining the two wrappers must
// not let either mask the other.
func TestReportedValidationFailureMatchesBothSentinels(t *testing.T) {
	err := reported{cause: newValidationFailure("syntax validation failed")}

	assert.ErrorIs(t, err, ErrReported)
	assert.ErrorIs(t, err, ErrValidationFailed)
	assert.EqualError(t, err, "syntax validation failed")

	// A reported error that is not a validation failure stays one.
	other := reported{cause: errors.New("missing --location")}
	assert.ErrorIs(t, other, ErrReported)
	assert.NotErrorIs(t, other, ErrValidationFailed)
}
