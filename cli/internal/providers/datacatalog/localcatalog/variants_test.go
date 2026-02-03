package localcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariantV1_FromV0(t *testing.T) {
	t.Run("converts variant with cases and default", func(t *testing.T) {
		t.Parallel()
		v0 := Variant{
			Type:          "conditional",
			Discriminator: "#property:event_type",
			Cases: []VariantCase{
				{
					DisplayName: "Sign Up Event",
					Match:       []any{"signup"},
					Description: "Properties for signup events",
					Properties: []PropertyReference{
						{Ref: "#property:user_id", Required: true},
						{Ref: "#property:email", Required: true},
					},
				},
				{
					DisplayName: "Login Event",
					Match:       []any{"login"},
					Properties: []PropertyReference{
						{Ref: "#property:session_id", Required: false},
					},
				},
			},
			Default: []PropertyReference{
				{Ref: "#property:timestamp", Required: true},
			},
		}

		var v1 VariantV1
		err := v1.FromV0(v0)

		require.NoError(t, err)
		assert.Equal(t, "conditional", v1.Type)
		assert.Equal(t, "#property:event_type", v1.Discriminator)

		require.Len(t, v1.Cases, 2)
		assert.Equal(t, "Sign Up Event", v1.Cases[0].DisplayName)
		assert.Equal(t, []any{"signup"}, v1.Cases[0].Match)
		assert.Equal(t, "Properties for signup events", v1.Cases[0].Description)
		require.Len(t, v1.Cases[0].Properties, 2)
		assert.Equal(t, "#property:user_id", v1.Cases[0].Properties[0].Property)
		assert.True(t, v1.Cases[0].Properties[0].Required)
		assert.Equal(t, "#property:email", v1.Cases[0].Properties[1].Property)
		assert.True(t, v1.Cases[0].Properties[1].Required)

		assert.Equal(t, "Login Event", v1.Cases[1].DisplayName)
		require.Len(t, v1.Cases[1].Properties, 1)
		assert.Equal(t, "#property:session_id", v1.Cases[1].Properties[0].Property)
		assert.False(t, v1.Cases[1].Properties[0].Required)

		require.Len(t, v1.Default, 1)
		assert.Equal(t, "#property:timestamp", v1.Default.Properties[0].Property)
		assert.True(t, v1.Default.Properties[0].Required)
	})
}

func TestVariantsV1_FromV0(t *testing.T) {

	t.Run("converts variant array to variant array v1", func(t *testing.T) {
		t.Parallel()
		v0 := Variants{
			{
				Type:          "conditional",
				Discriminator: "#property:platform",
				Cases: []VariantCase{
					{
						DisplayName: "iOS",
						Match:       []any{"ios"},
						Properties: []PropertyReference{
							{Ref: "#property:ios_version", Required: true},
						},
					},
				},
				Default: []PropertyReference{},
			},
			{
				Type:          "conditional",
				Discriminator: "#property:country",
				Cases: []VariantCase{
					{
						DisplayName: "US",
						Match:       []any{"US", "USA"},
						Properties: []PropertyReference{
							{Ref: "#property:state", Required: true},
						},
					},
				},
				Default: []PropertyReference{
					{Ref: "#property:country_code", Required: true},
				},
			},
		}

		var v1 VariantsV1
		err := v1.FromV0(v0)

		require.NoError(t, err)
		require.Len(t, v1, 2)

		assert.Equal(t, "#property:platform", v1[0].Discriminator)
		require.Len(t, v1[0].Cases, 1)
		assert.Equal(t, "iOS", v1[0].Cases[0].DisplayName)
		assert.Empty(t, v1[0].Default)

		assert.Equal(t, "#property:country", v1[1].Discriminator)
		require.Len(t, v1[1].Cases, 1)
		assert.Equal(t, "US", v1[1].Cases[0].DisplayName)
		require.Len(t, v1[1].Default, 1)
		assert.Equal(t, "#property:country_code", v1[1].Default.Properties[0].Property)
	})
}
