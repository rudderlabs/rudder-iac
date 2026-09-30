package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Short keys are labels such as "api" or "dev" and stay readable; longer
// keys may be real and keep 4 characters at each end at most.
func TestMaskWriteKey(t *testing.T) {
	t.Parallel()
	for key, want := range map[string]string{
		"":                      "",
		"api":                   "api",
		"dev":                   "dev",
		"worker01":              "worker01",
		"svc-order1":            "svc-...",
		"2Nd5xQkRealLookingKey": "2Nd5...gKey",
	} {
		require.Equal(t, want, MaskWriteKey(key), key)
	}
}
