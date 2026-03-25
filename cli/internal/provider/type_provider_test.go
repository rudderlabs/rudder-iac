package provider_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
)

func TestEmptyProvider_SupportedMatchPatterns(t *testing.T) {
	t.Parallel()

	var p provider.EmptyProvider
	assert.Nil(t, p.SupportedMatchPatterns())
}

func TestBaseProvider_SupportedMatchPatterns(t *testing.T) {
	t.Parallel()

	bp := provider.NewBaseProvider(nil)
	require.NotNil(t, bp)
	assert.Nil(t, bp.SupportedMatchPatterns())
}
