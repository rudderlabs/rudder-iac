package delete_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
)

// These tests exercise resourceops.ValidateType as used by the delete command.
// The delete command delegates type validation to the shared helper rather than
// maintaining its own copy — these tests confirm the shared contract holds.

func TestValidateType_KnownType_ReturnsNil(t *testing.T) {
	t.Parallel()

	types := []string{"event-stream-source", "event-stream-destination"}

	require.NoError(t, resourceops.ValidateType(types, "event-stream-source"))
	require.NoError(t, resourceops.ValidateType(types, "event-stream-destination"))
}

func TestValidateType_UnknownType_ListsValidTypes(t *testing.T) {
	t.Parallel()

	types := []string{"event-stream-source", "event-stream-destination"}

	err := resourceops.ValidateType(types, "bogus-type")
	require.Error(t, err)

	msg := err.Error()
	assert.Contains(t, msg, "bogus-type")
	assert.True(t, strings.Contains(msg, "event-stream-source") || strings.Contains(msg, "event-stream-destination"),
		"error should list at least one valid type, got: %s", msg)
}
