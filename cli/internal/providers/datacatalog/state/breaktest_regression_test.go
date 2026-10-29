package state

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/stretchr/testify/assert"
)

func TestCustomTypeDiffUpstreamNormalizesConfigKeys(t *testing.T) {
	args := &CustomTypeArgs{
		Name: "email", Type: "string",
		Config: map[string]any{"min_length": 3, "max_length": 80},
	}
	upstream := &catalog.CustomType{
		Name: "email", Type: "string",
		Config: map[string]any{"minLength": 3, "maxLength": 80},
	}
	assert.False(t, args.DiffUpstream(upstream), "import followed by apply must converge")
}
