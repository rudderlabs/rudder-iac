package trackingplan

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

// validateVariant validates a single Variant struct directly,
// decoupled from the TrackingPlan or CustomType parent context.
func validateVariant(variant localcatalog.Variant) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(variant, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "/variants",
			Message:   err.Error(),
		}}
	}
	return funcs.ParseValidationErrors(validationErrors)
}

func TestVariantsSyntaxValid_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		variant localcatalog.Variant
	}{
		{
			name: "single case",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/page_type",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Search",
						Description: "Search page",
						Properties: []localcatalog.PropertyReference{
							{Ref: "#/properties/group/search_term", Required: true},
						},
					},
				},
			},
		},
		{
			name: "multiple cases and default",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/event_category",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Ecommerce",
						Description: "E-commerce events",
						Properties: []localcatalog.PropertyReference{
							{Ref: "#/properties/group/product_id"},
							{Ref: "#/properties/group/price", Required: true},
						},
					},
					{
						DisplayName: "Content",
						Properties: []localcatalog.PropertyReference{
							{Ref: "#/properties/group/article_id"},
						},
					},
				},
				Default: []localcatalog.PropertyReference{
					{Ref: "#/properties/group/user_id", Required: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateVariant(tt.variant)
			assert.Empty(t, results, "Valid variant should produce no errors")
		})
	}
}

func TestVariantsSyntaxValid_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		variant      localcatalog.Variant
		expectedRefs []string
		expectedMsgs []string
	}{
		{
			name: "type not discriminator",
			variant: localcatalog.Variant{
				Type:          "invalid_type",
				Discriminator: "#/properties/group/field",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Case1",
						Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
					},
				},
			},
			expectedRefs: []string{"/type"},
			expectedMsgs: []string{"'type' must equal 'discriminator'"},
		},
		{
			name: "discriminator empty",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Case1",
						Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
					},
				},
			},
			expectedRefs: []string{"/discriminator"},
			expectedMsgs: []string{"'discriminator' is required"},
		},
		{
			name: "discriminator invalid format",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "not_a_reference",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Case1",
						Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
					},
				},
			},
			expectedRefs: []string{"/discriminator"},
			expectedMsgs: []string{"'discriminator' is not valid: must be of pattern #/properties/<group>/<id>"},
		},
		{
			name: "cases array empty",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/field",
				Cases:         []localcatalog.VariantCase{},
			},
			expectedRefs: []string{"/cases"},
			expectedMsgs: []string{"'cases' length must be greater than or equal to 1"},
		},
		{
			name: "case missing display_name",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/field",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "",
						Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
					},
				},
			},
			expectedRefs: []string{"/cases/0/display_name"},
			expectedMsgs: []string{"'display_name' is required"},
		},
		{
			name: "case properties empty",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/field",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Case1",
						Properties:  []localcatalog.PropertyReference{},
					},
				},
			},
			expectedRefs: []string{"/cases/0/properties"},
			expectedMsgs: []string{"'properties' length must be greater than or equal to 1"},
		},
		{
			name: "property reference missing $ref",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/field",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Case1",
						Properties:  []localcatalog.PropertyReference{{Ref: ""}},
					},
				},
			},
			expectedRefs: []string{"/cases/0/properties/0/$ref"},
			expectedMsgs: []string{"'$ref' is required"},
		},
		{
			name: "property reference invalid format",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/field",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Case1",
						Properties:  []localcatalog.PropertyReference{{Ref: "invalid_ref"}},
					},
				},
			},
			expectedRefs: []string{"/cases/0/properties/0/$ref"},
			expectedMsgs: []string{"'$ref' is not valid: must be of pattern #/properties/<group>/<id>"},
		},
		{
			name: "default property reference missing $ref",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/field",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Case1",
						Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
					},
				},
				Default: []localcatalog.PropertyReference{{Ref: ""}},
			},
			expectedRefs: []string{"/default/0/$ref"},
			expectedMsgs: []string{"'$ref' is required"},
		},
		{
			name: "default property reference invalid format",
			variant: localcatalog.Variant{
				Type:          "discriminator",
				Discriminator: "#/properties/group/field",
				Cases: []localcatalog.VariantCase{
					{
						DisplayName: "Case1",
						Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
					},
				},
				Default: []localcatalog.PropertyReference{{Ref: "invalid"}},
			},
			expectedRefs: []string{"/default/0/$ref"},
			expectedMsgs: []string{"'$ref' is not valid: must be of pattern #/properties/<group>/<id>"},
		},
		{
			name: "multiple errors in same variant",
			variant: localcatalog.Variant{
				Type:          "wrong",
				Discriminator: "",
				Cases:         []localcatalog.VariantCase{},
			},
			expectedRefs: []string{
				"/type",
				"/discriminator",
				"/cases",
			},
			expectedMsgs: []string{
				"'type' must equal 'discriminator'",
				"'discriminator' is required",
				"'cases' length must be greater than or equal to 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateVariant(tt.variant)

			assert.Len(t, results, len(tt.expectedRefs), "Expected %d errors, got %d", len(tt.expectedRefs), len(results))

			actualRefs := make([]string, len(results))
			actualMsgs := make([]string, len(results))
			for i, r := range results {
				actualRefs[i] = r.Reference
				actualMsgs[i] = r.Message
			}

			assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "References don't match")
			assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Messages don't match")
		})
	}
}
