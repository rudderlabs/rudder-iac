package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	s "github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A *PropertyRef held as a map value or slice element reaches the walk as an
// interface value. It has to resolve there too, because a destination's account
// reference lives inside its config map.
func TestDereferenceByReflection_RefInsideAnyContainers(t *testing.T) {
	t.Parallel()

	state := s.EmptyState()
	state.AddResource(&s.ResourceState{
		ID:     "google-ads-prod",
		Type:   "account",
		Output: map[string]any{"id": "remote-acct-1"},
	})
	urn := resources.URN("google-ads-prod", "account")

	container := &struct {
		Config map[string]any
		List   []any
	}{
		Config: map[string]any{
			"rudder_account_id": &resources.PropertyRef{URN: urn, Property: "id"},
			"untouched":         "literal",
		},
		List: []any{&resources.PropertyRef{URN: urn, Property: "id"}},
	}

	require.NoError(t, s.DereferenceByReflection(container, state))

	assert.Equal(t, &resources.PropertyRef{
		URN: urn, Property: "id", IsResolved: true, Value: "remote-acct-1",
	}, container.Config["rudder_account_id"])
	assert.Equal(t, "literal", container.Config["untouched"])
	assert.Equal(t, &resources.PropertyRef{
		URN: urn, Property: "id", IsResolved: true, Value: "remote-acct-1",
	}, container.List[0])
}

// A nil interface value must not panic the walk.
func TestDereferenceByReflection_NilInsideMap(t *testing.T) {
	t.Parallel()

	container := &struct{ Config map[string]any }{Config: map[string]any{"empty": nil}}
	require.NoError(t, s.DereferenceByReflection(container, s.EmptyState()))
	assert.Nil(t, container.Config["empty"])
}

// A ref inside a map whose target is absent from state fails, the same way a
// ref on a typed field does.
func TestDereferenceByReflection_MissingTargetInsideMap(t *testing.T) {
	t.Parallel()

	container := &struct{ Config map[string]any }{
		Config: map[string]any{
			"rudder_account_id": &resources.PropertyRef{
				URN: resources.URN("absent", "account"), Property: "id",
			},
		},
	}
	err := s.DereferenceByReflection(container, s.EmptyState())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}
