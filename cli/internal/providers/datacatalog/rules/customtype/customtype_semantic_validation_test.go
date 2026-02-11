package customtype

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

func TestCustomTypeSemanticValid_AllRefsFound(t *testing.T) {
	t.Parallel()

	graph := funcs.GraphWith(
		"email", "property",
		"phone", "property",
		"signup_method", "property",
	)

	spec := localcatalog.CustomTypeSpec{
		Types: []localcatalog.CustomType{
			{
				LocalID: "contact_info",
				Name:    "Contact Info",
				Type:    "object",
				Properties: []localcatalog.CustomTypeProperty{
					{Ref: "#property:email"},
					{Ref: "#property:phone"},
				},
				Variants: localcatalog.Variants{
					{
						Type:          "discriminator",
						Discriminator: "#property:signup_method",
						Cases: []localcatalog.VariantCase{
							{
								DisplayName: "Email",
								Properties: []localcatalog.PropertyReference{
									{Ref: "#property:email"},
								},
							},
						},
					},
				},
			},
		},
	}

	results := funcs.ValidateReferences(spec, graph)
	assert.Empty(t, results, "all refs exist in graph — no errors expected")
}

func TestCustomTypeSemanticValid_PropertyRefNotFound(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()

	spec := localcatalog.CustomTypeSpec{
		Types: []localcatalog.CustomType{
			{
				LocalID: "contact_info",
				Name:    "Contact Info",
				Type:    "object",
				Properties: []localcatalog.CustomTypeProperty{
					{Ref: "#property:missing_prop"},
				},
			},
		},
	}

	results := funcs.ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/types/0/properties/0/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'missing_prop' not found")
}

func TestCustomTypeSemanticValid_VariantDiscriminatorNotFound(t *testing.T) {
	t.Parallel()

	graph := funcs.GraphWith("email", "property")

	spec := localcatalog.CustomTypeSpec{
		Types: []localcatalog.CustomType{
			{
				LocalID: "contact_info",
				Name:    "Contact Info",
				Type:    "object",
				Properties: []localcatalog.CustomTypeProperty{
					{Ref: "#property:email"},
				},
				Variants: localcatalog.Variants{
					{
						Type:          "discriminator",
						Discriminator: "#property:missing_discriminator",
						Cases: []localcatalog.VariantCase{
							{
								DisplayName: "Case 1",
								Properties: []localcatalog.PropertyReference{
									{Ref: "#property:email"},
								},
							},
						},
					},
				},
			},
		},
	}

	results := funcs.ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'missing_discriminator' not found")
}

func TestCustomTypeSemanticValid_VariantCasePropertyNotFound(t *testing.T) {
	t.Parallel()

	graph := funcs.GraphWith("signup_method", "property")

	spec := localcatalog.CustomTypeSpec{
		Types: []localcatalog.CustomType{
			{
				LocalID: "contact_info",
				Name:    "Contact Info",
				Type:    "object",
				Variants: localcatalog.Variants{
					{
						Type:          "discriminator",
						Discriminator: "#property:signup_method",
						Cases: []localcatalog.VariantCase{
							{
								DisplayName: "Email",
								Properties: []localcatalog.PropertyReference{
									{Ref: "#property:missing_case_prop"},
								},
							},
						},
					},
				},
			},
		},
	}

	results := funcs.ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/types/0/variants/0/cases/0/properties/0/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'missing_case_prop' not found")
}

func TestCustomTypeSemanticValid_VariantDefaultPropertyNotFound(t *testing.T) {
	t.Parallel()

	graph := funcs.GraphWith("signup_method", "property", "email", "property")

	spec := localcatalog.CustomTypeSpec{
		Types: []localcatalog.CustomType{
			{
				LocalID: "contact_info",
				Name:    "Contact Info",
				Type:    "object",
				Variants: localcatalog.Variants{
					{
						Type:          "discriminator",
						Discriminator: "#property:signup_method",
						Cases: []localcatalog.VariantCase{
							{
								DisplayName: "Email",
								Properties: []localcatalog.PropertyReference{
									{Ref: "#property:email"},
								},
							},
						},
						Default: []localcatalog.PropertyReference{
							{Ref: "#property:missing_default_prop"},
						},
					},
				},
			},
		},
	}

	results := funcs.ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/types/0/variants/0/default/0/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'missing_default_prop' not found")
}

func TestCustomTypeSemanticValid_NoVariants(t *testing.T) {
	t.Parallel()

	graph := funcs.GraphWith("email", "property")

	spec := localcatalog.CustomTypeSpec{
		Types: []localcatalog.CustomType{
			{
				LocalID: "simple_type",
				Name:    "Simple Type",
				Type:    "object",
				Properties: []localcatalog.CustomTypeProperty{
					{Ref: "#property:email"},
				},
			},
		},
	}

	results := funcs.ValidateReferences(spec, graph)
	assert.Empty(t, results)
}
