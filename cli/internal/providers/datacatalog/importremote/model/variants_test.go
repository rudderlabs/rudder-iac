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
					"prop_event_type_id": "#/properties/common/event_type",
					"prop_user_id":       "#/properties/user/user_id",
					"prop_email":         "#/properties/user/email",
					"prop_source":        "#/properties/tracking/source",
					"prop_session_id":    "#/properties/session/session_id",
					"prop_timestamp":     "#/properties/common/timestamp",
					"prop_user_agent":    "#/properties/common/user_agent",
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
				Discriminator: "#/properties/common/event_type",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Sign Up Event",
						Match:       []any{"signup"},
						Description: "Properties for signup events",
						Properties: []localcatalog.PropertyReference{
							{Ref: "#/properties/user/user_id", Required: true},
							{Ref: "#/properties/user/email", Required: true},
							{Ref: "#/properties/tracking/source", Required: false},
						},
					},
					{
						DisplayName: "Login Event",
						Match:       []any{"login"},
						Description: "Properties for login events",
						Properties: []localcatalog.PropertyReference{
							{Ref: "#/properties/user/user_id", Required: true},
							{Ref: "#/properties/session/session_id", Required: false},
						},
					},
				},
				Default: []localcatalog.PropertyReference{
					{Ref: "#/properties/common/timestamp", Required: true},
					{Ref: "#/properties/common/user_agent", Required: false},
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
					"prop_platform":     "#/properties/device/platform",
					"prop_ios_version":  "#/properties/device/ios_version",
					"prop_country":      "#/properties/location/country",
					"prop_state":        "#/properties/location/state",
					"prop_zipcode":      "#/properties/location/zipcode",
					"prop_country_code": "#/properties/location/country_code",
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
				Discriminator: "#/properties/device/platform",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "iOS Platform",
						Match:       []any{"ios"},
						Properties: []localcatalog.PropertyReference{
							{Ref: "#/properties/device/ios_version", Required: true},
						},
					},
				},
				Default: []localcatalog.PropertyReference{},
			},
			{
				Type:          "conditional",
				Discriminator: "#/properties/location/country",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "US Users",
						Match:       []any{"US", "USA"},
						Properties: []localcatalog.PropertyReference{
							{Ref: "#/properties/location/state", Required: true},
							{Ref: "#/properties/location/zipcode", Required: false},
						},
					},
				},
				Default: []localcatalog.PropertyReference{
					{Ref: "#/properties/location/country_code", Required: true},
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
					return "#/properties/flags/feature_flag", nil
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
				Discriminator: "#/properties/flags/feature_flag",
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
					return "#/properties/valid", nil
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
					return "#/properties/valid", nil
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
					return "#/properties/valid", nil
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
					return "#/properties/valid", nil
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

func TestImportableVariantsV1_fromUpstream(t *testing.T) {
	t.Run("converts empty variants to nil", func(t *testing.T) {
		t.Parallel()
		mockRes := &mockResolver{}
		iv := &ImportableVariantsV1{}

		err := iv.fromUpstream(catalog.Variants{}, mockRes)

		require.NoError(t, err)
		require.Nil(t, iv.VariantsV1)
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
					"prop_event_type_id": "#/properties/common/event_type",
					"prop_user_id":       "#/properties/user/user_id",
					"prop_email":         "#/properties/user/email",
					"prop_source":        "#/properties/tracking/source",
					"prop_session_id":    "#/properties/session/session_id",
					"prop_timestamp":     "#/properties/common/timestamp",
					"prop_user_agent":    "#/properties/common/user_agent",
				}

				ref, ok := refMap[remoteID]
				if !ok {
					return "", fmt.Errorf("unknown property ID: %s", remoteID)
				}
				return ref, nil
			},
		}

		iv := &ImportableVariantsV1{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.NoError(t, err)

		expected := localcatalog.VariantsV1{
			{
				Type:          "conditional",
				Discriminator: "#/properties/common/event_type",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Sign Up Event",
						Match:       []any{"signup"},
						Description: "Properties for signup events",
						Properties: []localcatalog.PropertyReferenceV1{
							{Property: "#/properties/user/user_id", Required: true},
							{Property: "#/properties/user/email", Required: true},
							{Property: "#/properties/tracking/source", Required: false},
						},
					},
					{
						DisplayName: "Login Event",
						Match:       []any{"login"},
						Description: "Properties for login events",
						Properties: []localcatalog.PropertyReferenceV1{
							{Property: "#/properties/user/user_id", Required: true},
							{Property: "#/properties/session/session_id", Required: false},
						},
					},
				},
				Default: localcatalog.DefaultPropertiesV1{
					Properties: []localcatalog.PropertyReferenceV1{
						{Property: "#/properties/common/timestamp", Required: true},
						{Property: "#/properties/common/user_agent", Required: false},
					},
				},
			},
		}

		require.Equal(t, expected, iv.VariantsV1)
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
					"prop_platform":     "#/properties/device/platform",
					"prop_ios_version":  "#/properties/device/ios_version",
					"prop_country":      "#/properties/location/country",
					"prop_state":        "#/properties/location/state",
					"prop_zipcode":      "#/properties/location/zipcode",
					"prop_country_code": "#/properties/location/country_code",
				}

				ref, ok := refMap[remoteID]
				if !ok {
					return "", fmt.Errorf("unknown property ID: %s", remoteID)
				}
				return ref, nil
			},
		}

		iv := &ImportableVariantsV1{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.NoError(t, err)

		expected := localcatalog.VariantsV1{
			{
				Type:          "conditional",
				Discriminator: "#/properties/device/platform",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "iOS Platform",
						Match:       []any{"ios"},
						Properties: []localcatalog.PropertyReferenceV1{
							{Property: "#/properties/device/ios_version", Required: true},
						},
					},
				},
				Default: localcatalog.DefaultPropertiesV1{
					Properties: []localcatalog.PropertyReferenceV1{},
				},
			},
			{
				Type:          "conditional",
				Discriminator: "#/properties/location/country",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "US Users",
						Match:       []any{"US", "USA"},
						Properties: []localcatalog.PropertyReferenceV1{
							{Property: "#/properties/location/state", Required: true},
							{Property: "#/properties/location/zipcode", Required: false},
						},
					},
				},
				Default: localcatalog.DefaultPropertiesV1{
					Properties: []localcatalog.PropertyReferenceV1{
						{Property: "#/properties/location/country_code", Required: true},
					},
				},
			},
		}

		require.Equal(t, expected, iv.VariantsV1)
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
					return "#/properties/flags/feature_flag", nil
				}
				return "", fmt.Errorf("property not found: %s", remoteID)
			},
		}

		iv := &ImportableVariantsV1{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.NoError(t, err)

		expected := localcatalog.VariantsV1{
			{
				Type:          "conditional",
				Discriminator: "#/properties/flags/feature_flag",
				Cases:         []localcatalog.VariantCaseV1{},
				Default: localcatalog.DefaultPropertiesV1{
					Properties: []localcatalog.PropertyReferenceV1{},
				},
			},
		}

		require.Equal(t, expected, iv.VariantsV1)
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

		iv := &ImportableVariantsV1{}
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

		iv := &ImportableVariantsV1{}
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
					return "#/properties/valid", nil
				}
				return "", fmt.Errorf("property not found: %s", remoteID)
			},
		}

		iv := &ImportableVariantsV1{}
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
					return "#/properties/valid", nil
				}
				return "", nil
			},
		}

		iv := &ImportableVariantsV1{}
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
					return "#/properties/valid", nil
				}
				return "", fmt.Errorf("property not found: %s", remoteID)
			},
		}

		iv := &ImportableVariantsV1{}
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
					return "#/properties/valid", nil
				}
				return "", nil
			},
		}

		iv := &ImportableVariantsV1{}
		err := iv.fromUpstream(remoteVariants, mockRes)

		require.Error(t, err)
		require.Contains(t, err.Error(), "resolved reference is empty for property")
		require.Contains(t, err.Error(), "prop_empty_default")
		require.Contains(t, err.Error(), "variant default")
	})

}
