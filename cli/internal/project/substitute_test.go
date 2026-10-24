package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

type oneVariable struct{}

func (oneVariable) Resolve(string) (string, bool) { return "fake-value", true }

// A spec substitution leaves unchanged keeps its RawSpec, so the load reuses
// the parse the source rules made of it. A spec substitution changes gets a
// new RawSpec holding the substituted bytes.
func TestSubstituteSpecsKeepsUnchangedSpecs(t *testing.T) {
	t.Parallel()

	var (
		plain      = &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: properties\nspec:\n  id: x\n")}
		referenced = &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: properties\nspec:\n  id: '{{ .ID }}'\n")}
		p          = &project{substitutor: varsubst.NewSubstitutor(oneVariable{})}
	)

	substituted, diags, hasUndefined := p.substituteSpecs(map[string]*specs.RawSpec{
		"plain.yaml":      plain,
		"referenced.yaml": referenced,
	})
	require.Empty(t, diags)
	assert.False(t, hasUndefined)

	assert.Same(t, plain, substituted["plain.yaml"])
	assert.NotSame(t, referenced, substituted["referenced.yaml"])
	assert.Equal(t, "version: rudder/v1\nkind: properties\nspec:\n  id: 'fake-value'\n", string(substituted["referenced.yaml"].Data))
}
