package provider_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyProvider_NotImplementedMethods(t *testing.T) {
	t.Parallel()

	p := &provider.EmptyProvider{}
	ctx := context.Background()
	data := resources.ResourceData{"k": "v"}

	tests := map[string]func() error{
		"Create": func() error {
			_, err := p.Create(ctx, "id", "type", data)
			return err
		},
		"CreateRaw": func() error {
			_, err := p.CreateRaw(ctx, resources.NewResource("id", "type", data, nil))
			return err
		},
		"Update": func() error {
			_, err := p.Update(ctx, "id", "type", data, data)
			return err
		},
		"UpdateRaw": func() error {
			_, err := p.UpdateRaw(ctx, resources.NewResource("id", "type", data, nil), map[string]any{}, map[string]any{})
			return err
		},
		"Import": func() error {
			_, err := p.Import(ctx, "id", "type", data, "remote")
			return err
		},
		"ImportRaw": func() error {
			_, err := p.ImportRaw(ctx, resources.NewResource("id", "type", data, nil), "remote")
			return err
		},
		"Delete": func() error {
			return p.Delete(ctx, "id", "type", data)
		},
		"DeleteRaw": func() error {
			return p.DeleteRaw(ctx, "id", "type", map[string]any{}, map[string]any{})
		},
		"LoadLegacySpec": func() error {
			return p.LoadLegacySpec("dummy.yaml", &specs.Spec{})
		},
		"MigrateSpec": func() error {
			_, err := p.MigrateSpec(&specs.Spec{})
			return err
		},
	}

	for name, run := range tests {
		t.Run(name, func(t *testing.T) {
			err := run()
			require.Error(t, err)
			assert.EqualError(t, err, "not implemented")
		})
	}
}

func TestEmptyProvider_DefaultNoOpMethods(t *testing.T) {
	t.Parallel()

	p := &provider.EmptyProvider{}
	err := p.ConsolidateSync(context.Background(), resources.NewGraph(), state.EmptyState())
	require.NoError(t, err)

	assert.Nil(t, p.SupportedMatchPatterns())
	assert.Equal(t, []any{}, asAnySlice(p.SyntacticRules()))
	assert.Equal(t, []any{}, asAnySlice(p.SemanticRules()))
}

func asAnySlice[T any](items []T) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}
