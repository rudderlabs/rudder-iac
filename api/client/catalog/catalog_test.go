package catalog_test

import (
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOptionsToQuery(t *testing.T) {
	t.Run("empty options", func(t *testing.T) {
		opts := catalog.ListOptions{}
		assert.Equal(t, "", opts.ToQuery())
	})

	t.Run("has external id true", func(t *testing.T) {
		hasExternalID := true
		opts := catalog.ListOptions{HasExternalID: &hasExternalID}
		assert.Equal(t, "?hasExternalId=true", opts.ToQuery())
	})

	t.Run("has external id false", func(t *testing.T) {
		hasExternalID := false
		opts := catalog.ListOptions{HasExternalID: &hasExternalID}
		assert.Equal(t, "?hasExternalId=false", opts.ToQuery())
	})
}

func TestWithConcurrency(t *testing.T) {
	t.Run("returns error for zero", func(t *testing.T) {
		_, err := catalog.NewRudderDataCatalog(nil, catalog.WithConcurrency(0))
		require.Error(t, err)
		assert.ErrorContains(t, err, "applying option")
		assert.ErrorContains(t, err, "concurrency must be greater than 0")
	})

	t.Run("applies option", func(t *testing.T) {
		instance, err := catalog.NewRudderDataCatalog(nil, catalog.WithConcurrency(2))
		require.NoError(t, err)
		assert.NotNil(t, instance)
	})
}

func TestWithEventUpdateBatchSize(t *testing.T) {
	t.Run("returns error for zero", func(t *testing.T) {
		_, err := catalog.NewRudderDataCatalog(nil, catalog.WithEventUpdateBatchSize(0))
		require.Error(t, err)
		assert.ErrorContains(t, err, "applying option")
		assert.ErrorContains(t, err, "event update batch size must be greater than 0")
	})

	t.Run("applies option", func(t *testing.T) {
		instance, err := catalog.NewRudderDataCatalog(nil, catalog.WithEventUpdateBatchSize(5))
		require.NoError(t, err)
		assert.NotNil(t, instance)
	})
}

func TestCatalogErrors(t *testing.T) {
	t.Run("IsCatalogNotFoundError", func(t *testing.T) {
		err := &client.APIError{HTTPStatusCode: 400, Message: "catalog entry not found"}
		assert.True(t, catalog.IsCatalogNotFoundError(err))
		assert.False(t, catalog.IsCatalogNotFoundError(errors.New("other")))
		assert.False(t, catalog.IsCatalogNotFoundError(&client.APIError{HTTPStatusCode: 404, Message: "not found"}))
	})

	t.Run("IsCatalogAlreadyExistsError", func(t *testing.T) {
		err := &client.APIError{HTTPStatusCode: 400, Message: "catalog entry already exists"}
		assert.True(t, catalog.IsCatalogAlreadyExistsError(err))
		assert.False(t, catalog.IsCatalogAlreadyExistsError(errors.New("other")))
		assert.False(t, catalog.IsCatalogAlreadyExistsError(&client.APIError{HTTPStatusCode: 409, Message: "already exists"}))
	})
}
