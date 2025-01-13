package entity_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertyValidationRules_DuplicateKeys(t *testing.T) {
	t.Parallel()

	valid := &localcatalog.DataCatalog{
		Properties: map[localcatalog.EntityGroup][]*localcatalog.Property{
			"group_1": {
				{
					LocalID: "property_1",
					Name:    "property_1",
					Type:    "string",
					Config: map[string]interface{}{
						"enums": []string{"a", "b", "c"},
					},
				},
				{
					LocalID: "property_2",
					Name:    "property_1", // same name but different type
					Type:    "number",
					Config: map[string]interface{}{
						"minValue": 0,
						"maxValue": 100,
					},
				},
			},
		},
	}

	invalid := &localcatalog.DataCatalog{
		Properties: map[localcatalog.EntityGroup][]*localcatalog.Property{
			"group_1": {
				{
					LocalID: "property_1",
					Name:    "property_1",
					Type:    "string",
				},
				{
					LocalID: "property_2",
					Name:    "property_1", // duplicate name and but different type
					Type:    "int",
				},
			},
			"group_2": {
				{
					LocalID: "property_1", // duplicate id
					Name:    "property_1", // duplicate name and type
					Type:    "string",
				},
			},
		},
	}

	t.Run("no duplicate keys", func(t *testing.T) {
		t.Parallel()

		rule := entity.PropertyDuplicateKeysRule{}

		err := rule.Validate("#/properties/group_1/property_1", valid.Properties["group_1"][0], valid)
		require.Nil(t, err)

		err = rule.Validate("#/properties/group_1/property_2", valid.Properties["group_1"][1], valid)
		require.Nil(t, err)
	})

	t.Run("duplicate keys", func(t *testing.T) {
		t.Parallel()

		rule := entity.PropertyDuplicateKeysRule{}

		errs := rule.Validate("#/properties/group_1/property_1", invalid.Properties["group_1"][0], invalid)
		require.Len(t, errs, 2)
		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/properties/group_1/property_1",
			Err:        entity.ErrDuplicateByID,
			EntityType: entity.Property,
		})
		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/properties/group_1/property_1",
			Err:        entity.ErrDuplicateByNameType,
			EntityType: entity.Property,
		})

		errs = rule.Validate("#/properties/group_1/property_2", invalid.Properties["group_1"][1], invalid)
		require.Len(t, errs, 0)

		errs = rule.Validate("#/properties/group_2/property_1", invalid.Properties["group_2"][0], invalid)
		require.Len(t, errs, 2)
		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/properties/group_2/property_1",
			Err:        entity.ErrDuplicateByID,
			EntityType: entity.Property,
		})
		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/properties/group_2/property_1",
			Err:        entity.ErrDuplicateByNameType,
			EntityType: entity.Property,
		})
	})
}

func TestPropertyValidationRules_RequiredKeys(t *testing.T) {
	t.Parallel()

	valid := &localcatalog.DataCatalog{
		Properties: map[localcatalog.EntityGroup][]*localcatalog.Property{
			"group_1": {
				{
					LocalID: "property_1",
					Name:    "property_1",
					Type:    "string",
				},
			},
			"group_2": {
				{
					LocalID: "property_2",
					Name:    "property_2",
					Type:    "string",
				},
			},
		},
	}

	invalid := &localcatalog.DataCatalog{
		Properties: map[localcatalog.EntityGroup][]*localcatalog.Property{
			"group_1": {
				{
					LocalID: "", // missing local_id
					Name:    "property_1",
					Type:    "string",
				},
			},
			"group_2": {
				{
					LocalID: "property_2",
					Name:    "", // missing name
					Type:    "string",
				},
				{
					LocalID: "property_3",
					Name:    "property_3",
					Type:    "", // missing type
				},
			},
		},
	}

	t.Run("no missing required keys", func(t *testing.T) {
		t.Parallel()

		rule := entity.PropertyRequiredKeysRule{}

		err := rule.Validate("#/properties/group_1/property_1", valid.Properties["group_1"][0], valid)
		require.Nil(t, err)

		err = rule.Validate("#/properties/group_2/property_2", valid.Properties["group_2"][0], valid)
		require.Nil(t, err)
	})

	t.Run("missing required keys", func(t *testing.T) {
		t.Parallel()

		rule := entity.PropertyRequiredKeysRule{}

		errs := rule.Validate("#/properties/group_1/", invalid.Properties["group_1"][0], invalid)
		require.Len(t, errs, 1)
		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/properties/group_1/",
			Err:        entity.ErrMissingRequiredKeysID,
			EntityType: entity.Property,
		})

		errs = rule.Validate("#/properties/group_2/property_2", invalid.Properties["group_2"][0], invalid)
		require.Len(t, errs, 1)
		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/properties/group_2/property_2",
			Err:        entity.ErrMissingRequiredKeysName,
			EntityType: entity.Property,
		})

		errs = rule.Validate("#/properties/group_2/property_3", invalid.Properties["group_2"][1], invalid)
		require.Len(t, errs, 1)
		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/properties/group_2/property_3",
			Err:        entity.ErrInvalidRequiredKeysPropertyType,
			EntityType: entity.Property,
		})
	})

}
