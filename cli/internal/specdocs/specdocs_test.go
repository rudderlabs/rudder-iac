package specdocs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/specdocs"
)

// The contract prints suggestions as an array even when an error has none,
// and leaves the caller's error as it was.
func TestNewErrorDocPrintsSuggestionsAsAnArray(t *testing.T) {
	t.Parallel()

	e := &specdocs.Error{Code: specdocs.CodeInvalidID, Message: "bad id"}

	assert.Equal(t, `{
  "schemaVersion": 1,
  "cliVersion": "1.2.3",
  "error": {
    "code": "invalid_id",
    "message": "bad id",
    "suggestions": []
  }
}
`, string(encode(t, specdocs.NewErrorDoc("1.2.3", e))))
	assert.Nil(t, e.Suggestions)
}
