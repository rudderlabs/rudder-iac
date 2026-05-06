package source

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceSpec_AllowsOCamlType(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	enabled := true

	spec := SourceSpec{
		LocalID:          "test-source",
		Name:             "Test Source",
		SourceDefinition: "ocaml",
		Enabled:          &enabled,
	}

	err := validate.Struct(spec)
	require.NoError(t, err)
	assert.Contains(t, sourceDefinitions, "ocaml")
}
