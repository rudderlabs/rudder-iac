package customtype

import (
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

// extractRefs extracts Reference fields from ValidationResults
func extractRefs(results []rules.ValidationResult) []string {
	refs := make([]string, len(results))
	for i, result := range results {
		refs[i] = result.Reference
	}
	return refs
}

// extractMsgs extracts Message fields from ValidationResults
func extractMsgs(results []rules.ValidationResult) []string {
	msgs := make([]string, len(results))
	for i, result := range results {
		msgs[i] = result.Message
	}
	return msgs
}

func TestNewCustomTypeSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewCustomTypeSpecSyntaxValidRule()

	assert.Equal(t, "datacatalog/custom-types/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "custom type spec syntax must be valid", rule.Description())
	assert.Equal(t, prules.LegacyVersionPatterns("custom-types"), rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid, "Rule should have valid examples")
	assert.NotEmpty(t, examples.Invalid, "Rule should have invalid examples")
}

func TestCustomTypeSpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.CustomTypeSpec
	}{
		{
			name: "minimal valid spec with required fields only",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_status",
						Name:    "UserStatus",
						Type:    "string",
					},
				},
			},
		},
		{
			name: "complete spec with all fields populated",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID:     "address",
						Name:        "Address",
						Description: "Physical address structure",
						Type:        "object",
						Config:      map[string]any{"format": "json"},
					},
					{
						LocalID:     "coordinates",
						Name:        "Coordinates",
						Description: "Geographic coordinates",
						Type:        "object",
					},
				},
			},
		},
		{
			name: "custom type with properties references",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_profile",
						Name:    "UserProfile",
						Type:    "object",
						Properties: []localcatalog.CustomTypeProperty{
							{Ref: "#/properties/user-traits/name", Required: true},
							{Ref: "#/properties/user-traits/email", Required: true},
							{Ref: "#/properties/user-traits/age", Required: false},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeSpec(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)
			assert.Empty(t, results, "Valid spec should not produce validation errors")
		})
	}
}

func TestCustomTypeSpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "missing types array",
			spec: localcatalog.CustomTypeSpec{
				Types: nil,
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types"},
			expectedMsgs:   []string{"'types' is required"},
		},
		{
			name: "custom type missing id",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						Name: "UserStatus",
						Type: "string",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "custom type missing name",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_status",
						Type:    "string",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/name"},
			expectedMsgs:   []string{"'name' is required"},
		},
		{
			name: "custom type missing type",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_status",
						Name:    "UserStatus",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/type"},
			expectedMsgs:   []string{"'type' is required"},
		},
		{
			name: "custom type missing all required fields",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						Description: "Some description",
					},
				},
			},
			expectedErrors: 3,
			expectedRefs:   []string{"/types/0/id", "/types/0/name", "/types/0/type"},
			expectedMsgs:   []string{"'id' is required", "'name' is required", "'type' is required"},
		},
		{
			name: "custom type with invalid type value",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "invalid_type",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/type"},
			expectedMsgs:   []string{"'type' is not valid: must be one of the following: string, number, integer, boolean, array, object, null"},
		},
		{
			name: "custom type with property missing $ref",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_profile",
						Name:    "UserProfile",
						Type:    "object",
						Properties: []localcatalog.CustomTypeProperty{
							{Required: true},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/properties/0/$ref"},
			expectedMsgs:   []string{"'$ref' is required"},
		},
		{
			name: "custom type with invalid property reference format",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_profile",
						Name:    "UserProfile",
						Type:    "object",
						Properties: []localcatalog.CustomTypeProperty{
							{Ref: "invalid-ref-format", Required: true},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/properties/0/$ref"},
			expectedMsgs:   []string{"'$ref' is not valid: must be of pattern #/properties/<group>/<id>"},
		},
		{
			name: "multiple custom types with errors at different indices",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{LocalID: "valid_type", Name: "ValidType", Type: "string"},
					{Name: "MissingID", Type: "number"},
					{LocalID: "missing_name", Type: "boolean"},
					{LocalID: "missing_type", Name: "MissingType"},
				},
			},
			expectedErrors: 3,
			expectedRefs:   []string{"/types/1/id", "/types/2/name", "/types/3/type"},
			expectedMsgs:   []string{"'id' is required", "'name' is required", "'type' is required"},
		},
		{
			name: "description too short",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID:     "status",
						Name:        "Status",
						Type:        "string",
						Description: "ab",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/description"},
			expectedMsgs:   []string{"'description' length must be greater than or equal to 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeSpec(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors, "Unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)

				assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "Validation error references don't match")
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Validation error messages don't match")
			}
		})
	}
}

func TestCustomTypeSpecSyntaxValidRule_VariantReferencePaths(t *testing.T) {
	t.Parallel()

	t.Run("nil variants is valid", func(t *testing.T) {
		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID:  "address",
					Name:     "Address",
					Type:     "object",
					Variants: nil,
				},
			},
		}

		results := validateCustomTypeSpec(
			localcatalog.KindCustomTypes,
			specs.SpecVersionV0_1,
			map[string]any{},
			spec,
		)
		assert.Empty(t, results, "Nil variants should produce no errors")
	})

	t.Run("empty variants is valid", func(t *testing.T) {
		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID:  "address",
					Name:     "Address",
					Type:     "object",
					Variants: localcatalog.Variants{},
				},
			},
		}

		results := validateCustomTypeSpec(
			localcatalog.KindCustomTypes,
			specs.SpecVersionV0_1,
			map[string]any{},
			spec,
		)
		assert.Empty(t, results, "Empty variants should produce no errors")
	})

	t.Run("valid variant produces no errors", func(t *testing.T) {
		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "address",
					Name:    "Address",
					Type:    "object",
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#/properties/address-props/country",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "US Address",
									Match:       []any{"US"},
									Properties: []localcatalog.PropertyReference{
										{Ref: "#/properties/address-props/zip_code", Required: true},
									},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSpec(
			localcatalog.KindCustomTypes,
			specs.SpecVersionV0_1,
			map[string]any{},
			spec,
		)
		assert.Empty(t, results, "Valid variant should produce no errors")
	})

	t.Run("more than one variant is invalid", func(t *testing.T) {
		validVariant := localcatalog.Variant{
			Type:          "discriminator",
			Discriminator: "#/properties/address-props/country",
			Cases: []localcatalog.VariantCase{
				{
					DisplayName: "Case1",
					Match:       []any{"value"},
					Properties: []localcatalog.PropertyReference{
						{Ref: "#/properties/address-props/zip_code", Required: true},
					},
				},
			},
		}

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID:  "address",
					Name:     "Address",
					Type:     "object",
					Variants: localcatalog.Variants{validVariant, validVariant},
				},
			},
		}

		results := validateCustomTypeSpec(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.NotEmpty(t, results)
		assert.Contains(t, extractRefs(results), "/types/0/variants")
		assert.Contains(t, extractMsgs(results), "'variants' length must be less than or equal to 1")
	})

	t.Run("invalid variant generates correct reference paths", func(t *testing.T) {
		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "address",
					Name:    "Address",
					Type:    "object",
					Variants: localcatalog.Variants{
						{
							Type:          "wrong_type",
							Discriminator: "bad_ref",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Case1",
									Match:       []any{"value"},
									Properties: []localcatalog.PropertyReference{
										{Ref: "bad_prop_ref"},
									},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSpec(
			localcatalog.KindCustomTypes,
			specs.SpecVersionV0_1,
			map[string]any{},
			spec,
		)

		expectedRefs := []string{
			"/types/0/variants/0/type",
			"/types/0/variants/0/discriminator",
			"/types/0/variants/0/cases/0/properties/0/$ref",
		}
		expectedMsgs := []string{
			"'type' must equal 'discriminator'",
			"'discriminator' is not valid: must be of pattern #/properties/<group>/<id>",
			"'$ref' is not valid: must be of pattern #/properties/<group>/<id>",
		}

		assert.Len(t, results, 3)
		assert.ElementsMatch(t, expectedRefs, extractRefs(results))
		assert.ElementsMatch(t, expectedMsgs, extractMsgs(results))
	})
}

func TestCustomTypeSpecSyntaxValidRule_VariantsOnlyForObjectType(t *testing.T) {
	t.Parallel()

	t.Run("variants on object type is valid", func(t *testing.T) {
		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "address",
					Name:    "Address",
					Type:    "object",
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#/properties/address-props/country",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "US Address",
									Match:       []any{"US"},
									Properties: []localcatalog.PropertyReference{
										{Ref: "#/properties/address-props/zip_code", Required: true},
									},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSpec(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.Empty(t, results, "Variants on object type should produce no errors")
	})

	t.Run("variants on non object type is invalid", func(t *testing.T) {
		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "status",
					Name:    "Status",
					Type:    "string",
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#/properties/props/field",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Case1",
									Match:       []any{"value"},
									Properties: []localcatalog.PropertyReference{
										{Ref: "#/properties/props/prop1", Required: true},
									},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSpec(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.NotEmpty(t, results)

		refs := extractRefs(results)
		msgs := extractMsgs(results)
		assert.Contains(t, refs, "/types/0/variants")
		assert.Contains(t, msgs, "'variants' is not allowed unless 'type object'")
	})

	t.Run("no variants on non-object type is valid", func(t *testing.T) {
		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "status",
					Name:    "Status",
					Type:    "string",
				},
			},
		}

		results := validateCustomTypeSpec(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.Empty(t, results, "No variants on non-object type should be valid")
	})
}

func TestCustomTypeSpecSyntaxValidRule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "empty types array is considered valid by go-validator",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "custom type with all fields empty",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{},
				},
			},
			expectedErrors: 3,
			expectedRefs:   []string{"/types/0/id", "/types/0/name", "/types/0/type"},
			expectedMsgs:   []string{"'id' is required", "'name' is required", "'type' is required"},
		},
		{
			name: "custom type with empty properties array",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID:    "user_profile",
						Name:       "UserProfile",
						Type:       "object",
						Properties: []localcatalog.CustomTypeProperty{},
					},
				},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "description at exact minimum length (3 chars)",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID:     "status",
						Name:        "Status",
						Type:        "string",
						Description: "abc",
					},
				},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "multiple properties with one invalid reference",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_profile",
						Name:    "UserProfile",
						Type:    "object",
						Properties: []localcatalog.CustomTypeProperty{
							{Ref: "#/properties/user-traits/name", Required: true},
							{Ref: "invalid-ref", Required: true},
							{Ref: "#/properties/user-traits/age", Required: false},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/properties/1/$ref"},
			expectedMsgs:   []string{"'$ref' is not valid: must be of pattern #/properties/<group>/<id>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeSpec(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors, "Unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)

				assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "Validation error references don't match")
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Validation error messages don't match")
			}
		})
	}
}
