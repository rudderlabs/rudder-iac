package model

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/stretchr/testify/require"
)

func TestImportableVariants_fromUpstream(t *testing.T) {
	t.Run("converts empty variants to nil", func(t *testing.T) {
		t.Parallel()
		mockRes := &mockResolver{}
		iv := &ImportableVariants{}

		err := iv.fromUpstream(catalog.Variants{}, mockRes)

		require.NoError(t, err)
		require.Nil(t, iv.Variants)
	})

	t.Run("converts single variant with all fields", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_event_type_id",
				Cases: []catalog.VariantCase{
					{
						DisplayName: "Sign Up Event",
						Match:       []any{"signup"},
						Description: "Properties for signup events",
						Properties: []catalog.PropertyReference{
							{ID: "prop_user_id", Required: true},
							{ID: "prop_email", Required: true},
							{ID: "prop_source", Required: false},
						},
					},
					{
						DisplayName: "Login Event",
						Match:       []any{"login"},
						Description: "Properties for login events",
						Properties: []catalog.PropertyReference{
							{ID: "prop_user_id", Required: true},
							{ID: "prop_session_id", Required: false},
						},
					},
				},
				Default: []catalog.PropertyReference{
					{ID: "prop_timestamp", Required: true},
					{ID: "prop_user_agent", Required: false},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				require.Equal(t, types.PropertyResourceType, entityType)

				refMap := map[string]string{
					"prop_event_type_id": "#property:event_type",
					"prop_user_id":       "#property:user_id",
					"prop_email":         "#property:email",
					"prop_source":        "#property:source",
					"prop_session_id":    "#property:session_id",
					"prop_timestamp":     "#property:timestamp",
					"prop_user_agent":    "#property:user_agent",
				}

				ref, ok := refMap[remoteID]
				if !ok {
					return "", fmt.Errorf("unknown property ID: %s", remoteID)
				}
				return ref, nil
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.NoError(t, err)

		expected := []localcatalog.Variant{
			{
				Type:          "conditional",
				Discriminator: "#property:event_type",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Sign Up Event",
						Match:       []any{"signup"},
						Description: "Properties for signup events",
						Properties: []localcatalog.PropertyReference{
							{Ref: "#property:user_id", Required: true},
							{Ref: "#property:email", Required: true},
							{Ref: "#property:source", Required: false},
						},
					},
					{
						DisplayName: "Login Event",
						Match:       []any{"login"},
						Description: "Properties for login events",
						Properties: []localcatalog.PropertyReference{
							{Ref: "#property:user_id", Required: true},
							{Ref: "#property:session_id", Required: false},
						},
					},
				},
				Default: []localcatalog.PropertyReference{
					{Ref: "#property:timestamp", Required: true},
					{Ref: "#property:user_agent", Required: false},
				},
			},
		}

		if !reflect.DeepEqual(iv.Variants, expected) {
			t.Errorf("Variants mismatch.\nExpected: %+v\nGot: %+v", expected, iv.Variants)
		}
	})

	t.Run("converts multiple variants", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_platform",
				Cases: []catalog.VariantCase{
					{
						DisplayName: "iOS Platform",
						Match:       []any{"ios"},
						Properties: []catalog.PropertyReference{
							{ID: "prop_ios_version", Required: true},
						},
					},
				},
				Default: []catalog.PropertyReference{},
			},
			{
				Type:          "conditional",
				Discriminator: "prop_country",
				Cases: []catalog.VariantCase{
					{
						DisplayName: "US Users",
						Match:       []any{"US", "USA"},
						Properties: []catalog.PropertyReference{
							{ID: "prop_state", Required: true},
							{ID: "prop_zipcode", Required: false},
						},
					},
				},
				Default: []catalog.PropertyReference{
					{ID: "prop_country_code", Required: true},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				refMap := map[string]string{
					"prop_platform":     "#property:platform",
					"prop_ios_version":  "#property:ios_version",
					"prop_country":      "#property:country",
					"prop_state":        "#property:state",
					"prop_zipcode":      "#property:zipcode",
					"prop_country_code": "#property:country_code",
				}

				ref, ok := refMap[remoteID]
				if !ok {
					return "", fmt.Errorf("unknown property ID: %s", remoteID)
				}
				return ref, nil
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.NoError(t, err)

		expected := []localcatalog.Variant{
			{
				Type:          "conditional",
				Discriminator: "#property:platform",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "iOS Platform",
						Match:       []any{"ios"},
						Properties: []localcatalog.PropertyReference{
							{Ref: "#property:ios_version", Required: true},
						},
					},
				},
				Default: []localcatalog.PropertyReference{},
			},
			{
				Type:          "conditional",
				Discriminator: "#property:country",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "US Users",
						Match:       []any{"US", "USA"},
						Properties: []localcatalog.PropertyReference{
							{Ref: "#property:state", Required: true},
							{Ref: "#property:zipcode", Required: false},
						},
					},
				},
				Default: []localcatalog.PropertyReference{
					{Ref: "#property:country_code", Required: true},
				},
			},
		}

		if !reflect.DeepEqual(iv.Variants, expected) {
			t.Errorf("Variants mismatch.\nExpected: %+v\nGot: %+v", expected, iv.Variants)
		}
	})

	t.Run("converts variant with empty cases", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_feature_flag",
				Cases:         []catalog.VariantCase{},
				Default:       []catalog.PropertyReference{},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if remoteID == "prop_feature_flag" {
					return "#property:feature_flag", nil
				}
				return "", fmt.Errorf("property not found: %s", remoteID)
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.NoError(t, err)

		expected := []localcatalog.Variant{
			{
				Type:          "conditional",
				Discriminator: "#property:feature_flag",
				Cases:         []localcatalog.VariantCase{},
				Default:       []localcatalog.PropertyReference{},
			},
		}

		if !reflect.DeepEqual(iv.Variants, expected) {
			t.Errorf("Variants mismatch.\nExpected: %+v\nGot: %+v", expected, iv.Variants)
		}
	})

	t.Run("errors when discriminator resolution fails", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_invalid",
				Cases:         []catalog.VariantCase{},
				Default:       []catalog.PropertyReference{},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", fmt.Errorf("property not found: %s", remoteID)
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.Error(t, err)
		require.Contains(t, err.Error(), "resolving reference for discriminator")
		require.Contains(t, err.Error(), "prop_invalid")
	})

	t.Run("errors when discriminator resolves to empty string", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_empty",
				Cases:         []catalog.VariantCase{},
				Default:       []catalog.PropertyReference{},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", nil
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.Error(t, err)
		require.Contains(t, err.Error(), "resolved reference is empty for discriminator")
		require.Contains(t, err.Error(), "prop_empty")
	})

	t.Run("errors when case property resolution fails", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_valid",
				Cases: []catalog.VariantCase{
					{
						DisplayName: "Test Case",
						Match:       []any{"test"},
						Properties: []catalog.PropertyReference{
							{ID: "prop_invalid", Required: true},
						},
					},
				},
				Default: []catalog.PropertyReference{},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if remoteID == "prop_valid" {
					return "#property:valid", nil
				}
				return "", fmt.Errorf("property not found: %s", remoteID)
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.Error(t, err)
		require.Contains(t, err.Error(), "resolving reference for property")
		require.Contains(t, err.Error(), "prop_invalid")
		require.Contains(t, err.Error(), "Test Case")
	})

	t.Run("errors when case property resolves to empty string", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_valid",
				Cases: []catalog.VariantCase{
					{
						DisplayName: "Empty Prop Case",
						Match:       []any{"test"},
						Properties: []catalog.PropertyReference{
							{ID: "prop_empty", Required: true},
						},
					},
				},
				Default: []catalog.PropertyReference{},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if remoteID == "prop_valid" {
					return "#property:valid", nil
				}
				return "", nil
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.Error(t, err)
		require.Contains(t, err.Error(), "resolved reference is empty for property")
		require.Contains(t, err.Error(), "prop_empty")
		require.Contains(t, err.Error(), "Empty Prop Case")
	})

	t.Run("errors when default property resolution fails", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_valid",
				Cases:         []catalog.VariantCase{},
				Default: []catalog.PropertyReference{
					{ID: "prop_invalid_default", Required: true},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if remoteID == "prop_valid" {
					return "#property:valid", nil
				}
				return "", fmt.Errorf("property not found: %s", remoteID)
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.Error(t, err)
		require.Contains(t, err.Error(), "resolving reference for property")
		require.Contains(t, err.Error(), "prop_invalid_default")
		require.Contains(t, err.Error(), "variant default")
	})

	t.Run("errors when default property resolves to empty string", func(t *testing.T) {
		t.Parallel()
		remoteVariants := catalog.Variants{
			{
				Type:          "conditional",
				Discriminator: "prop_valid",
				Cases:         []catalog.VariantCase{},
				Default: []catalog.PropertyReference{
					{ID: "prop_empty_default", Required: true},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if remoteID == "prop_valid" {
					return "#property:valid", nil
				}
				return "", nil
			},
		}

		iv := &ImportableVariants{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.Error(t, err)
		require.Contains(t, err.Error(), "resolved reference is empty for property")
		require.Contains(t, err.Error(), "prop_empty_default")
		require.Contains(t, err.Error(), "variant default")
	})

}
