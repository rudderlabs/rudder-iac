package resources_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestImportableFilterOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, resources.ImportableFilter{}, resources.ImportableFilterOf(nil),
		"no filter means import's behaviour: unmanaged only")
	assert.Equal(t, resources.ImportableFilter{IncludeManaged: true},
		resources.ImportableFilterOf([]resources.ImportableFilter{{IncludeManaged: true}}))
}

func TestImportableFilterUnmanagedOnly(t *testing.T) {
	t.Parallel()

	unmanaged := resources.ImportableFilter{}.UnmanagedOnly()
	if assert.NotNil(t, unmanaged, "an API filter must be sent when only unmanaged resources are wanted") {
		assert.False(t, *unmanaged)
	}

	assert.Nil(t, resources.ImportableFilter{IncludeManaged: true}.UnmanagedOnly(),
		"no API filter at all once managed resources are in scope")
}

func TestImportableFilterKeepID(t *testing.T) {
	t.Parallel()

	assert.Empty(t, resources.ImportableFilter{}.KeepID("upstream-id"),
		"a plain import names everything it writes, even a resource that came back managed")
	assert.Equal(t, "upstream-id", resources.ImportableFilter{IncludeManaged: true}.KeepID("upstream-id"))
	assert.Empty(t, resources.ImportableFilter{IncludeManaged: true}.KeepID(""))
}
