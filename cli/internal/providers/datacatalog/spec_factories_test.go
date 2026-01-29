package datacatalog

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertySpecFactory_Defaults(t *testing.T) {
	t.Parallel()

	factory := &PropertySpecFactory{}
	assert.Equal(t, "properties", factory.SpecFieldName())
	assert.Equal(t, "properties", factory.Kind())
	assert.NotEmpty(t, factory.Examples().Valid)
	assert.NotEmpty(t, factory.Examples().Invalid)

	_, ok := factory.NewSpec().(*localcatalog.PropertySpec)
	assert.True(t, ok, "NewSpec should create instance of *localcatalog.PropertySpec")
}

func TestProvider_SpecFactories(t *testing.T) {
	provider := New(nil)
	factories := provider.SpecFactories()

	require.NotEmpty(t, factories, "Provider should return spec factories")
}
