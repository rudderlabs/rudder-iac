package handler_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The example writer handler hands back every remote writer regardless of the
// filter, which makes it the right fixture for asserting what BaseHandler does
// with a managed resource in each mode.
func TestBaseHandler_LoadImportable_IdentityByFilter(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*backend.Backend, string, string) {
		t.Helper()

		b := backend.NewBackend()
		managed, err := b.CreateWriter("Ursula Le Guin", "le-guin")
		require.NoError(t, err)
		unmanaged, err := b.CreateWriter("Terry Pratchett", "")
		require.NoError(t, err)
		return b, managed.ID, unmanaged.ID
	}

	t.Run("without the filter every resource is named afresh", func(t *testing.T) {
		b, managedID, unmanagedID := setup(t)
		h := writer.NewHandler(b)

		got, err := h.LoadImportable(context.Background(), namer.NewExternalIdNamer(namer.NewKebabCase()))
		require.NoError(t, err)

		all := got.GetAll(writer.HandlerMetadata.ResourceType)
		assert.Equal(t, "ursula-le-guin", all[managedID].ExternalID,
			"a plain import treats everything it writes as new, so it does not adopt the upstream identifier")
		assert.Equal(t, "terry-pratchett", all[unmanagedID].ExternalID)
	})

	t.Run("with IncludeManaged the upstream identifier is kept", func(t *testing.T) {
		b, managedID, unmanagedID := setup(t)
		h := writer.NewHandler(b)

		got, err := h.LoadImportable(context.Background(), namer.NewExternalIdNamer(namer.NewKebabCase()),
			resources.ImportableFilter{IncludeManaged: true})
		require.NoError(t, err)

		all := got.GetAll(writer.HandlerMetadata.ResourceType)
		assert.Equal(t, "le-guin", all[managedID].ExternalID)
		assert.Equal(t, "#writer:le-guin", all[managedID].Reference,
			"the reference has to follow the kept identifier, or cross-spec references break")
		assert.Equal(t, "terry-pratchett", all[unmanagedID].ExternalID)
	})

	t.Run("a generated identifier never collides with a kept one", func(t *testing.T) {
		b := backend.NewBackend()
		// The unmanaged writer's name kebab-cases to exactly the managed
		// writer's upstream externalId.
		managed, err := b.CreateWriter("Anonymous", "le-guin")
		require.NoError(t, err)
		unmanaged, err := b.CreateWriter("Le Guin", "")
		require.NoError(t, err)

		h := writer.NewHandler(b)
		got, err := h.LoadImportable(context.Background(), namer.NewExternalIdNamer(namer.NewKebabCase()),
			resources.ImportableFilter{IncludeManaged: true})
		require.NoError(t, err)

		all := got.GetAll(writer.HandlerMetadata.ResourceType)
		assert.Equal(t, "le-guin", all[managed.ID].ExternalID)
		assert.NotEqual(t, "le-guin", all[unmanaged.ID].ExternalID)
	})
}
