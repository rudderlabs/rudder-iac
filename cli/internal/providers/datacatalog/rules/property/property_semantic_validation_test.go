package property

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestPropertySemanticValid_CustomTypeRefFound(t *testing.T) {
	t.Parallel()

	graph := funcs.GraphWith("Address", "custom-type")

	spec := localcatalog.PropertySpec{
		Properties: []localcatalog.Property{
			{LocalID: "address", Name: "Address", Type: "#custom-type:Address"},
		},
	}

	results := validatePropertySemantic("properties", "rudder/v1", nil, spec, graph)
	assert.Empty(t, results, "custom type exists in graph — no errors expected")
}

func TestPropertySemanticValid_CustomTypeRefNotFound(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()

	spec := localcatalog.PropertySpec{
		Properties: []localcatalog.Property{
			{LocalID: "address", Name: "Address", Type: "#custom-type:NonexistentType"},
		},
	}

	results := validatePropertySemantic("properties", "rudder/v1", nil, spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/properties/0/type", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced custom-type 'NonexistentType' not found")
}

func TestPropertySemanticValid_PrimitiveTypeSkipped(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()

	spec := localcatalog.PropertySpec{
		Properties: []localcatalog.Property{
			{LocalID: "name", Name: "Name", Type: "string"},
			{LocalID: "age", Name: "Age", Type: "number"},
			{LocalID: "active", Name: "Active", Type: "boolean"},
			{LocalID: "tags", Name: "Tags", Type: "array"},
			{LocalID: "meta", Name: "Meta", Type: "object"},
		},
	}

	results := validatePropertySemantic("properties", "rudder/v1", nil, spec, graph)
	assert.Empty(t, results, "primitive types should not trigger ref lookup")
}

func TestPropertySemanticValid_MixedPrimitiveAndRef(t *testing.T) {
	t.Parallel()

	graph := funcs.GraphWith("Address", "custom-type")

	spec := localcatalog.PropertySpec{
		Properties: []localcatalog.Property{
			{LocalID: "name", Name: "Name", Type: "string"},
			{LocalID: "address", Name: "Address", Type: "#custom-type:Address"},
			{LocalID: "payment", Name: "Payment", Type: "#custom-type:MissingPaymentType"},
		},
	}

	results := validatePropertySemantic("properties", "rudder/v1", nil, spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/properties/2/type", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced custom-type 'MissingPaymentType' not found")
}

func TestPropertySemanticValid_EmptyTypeSkipped(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()

	spec := localcatalog.PropertySpec{
		Properties: []localcatalog.Property{
			{LocalID: "name", Name: "Name", Type: ""},
		},
	}

	results := validatePropertySemantic("properties", "rudder/v1", nil, spec, graph)
	assert.Empty(t, results, "empty type should be skipped")
}
