package funcs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

// --- Test structs ---

type simpleRefSpec struct {
	Ref string `json:"$ref" validate:"required,pattern=legacy_property_ref"`
}

type multiRefSpec struct {
	EventRef    string `json:"event_ref" validate:"required,pattern=legacy_event_ref"`
	PropertyRef string `json:"prop_ref" validate:"required,pattern=legacy_property_ref"`
}

type nestedSpec struct {
	Name  string          `json:"name"`
	Inner *simpleRefSpec  `json:"inner"`
	Items []simpleRefSpec `json:"items"`
}

type pointerStringSpec struct {
	CategoryRef *string `json:"category" validate:"omitempty,pattern=legacy_category_ref"`
}

type sliceOfPointersSpec struct {
	Rules []*simpleRefSpec `json:"rules"`
}

type recursiveSpec struct {
	Ref      string           `json:"$ref" validate:"required,pattern=legacy_property_ref"`
	Children []*recursiveSpec `json:"children,omitempty"`
}

type noRefSpec struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

// --- ValidateReferences tests ---

func TestValidateReferences_SimpleRef_Found(t *testing.T) {
	t.Parallel()

	graph := GraphWith("user_id", "property")
	spec := simpleRefSpec{Ref: "#property:user_id"}

	results := ValidateReferences(spec, graph)
	assert.Empty(t, results)
}

func TestValidateReferences_SimpleRef_NotFound(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	spec := simpleRefSpec{Ref: "#property:user_id"}

	results := ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'user_id' not found")
}

func TestValidateReferences_MultipleRefs_MixedResults(t *testing.T) {
	t.Parallel()

	graph := GraphWith("signup", "event")
	spec := multiRefSpec{
		EventRef:    "#event:signup",
		PropertyRef: "#property:email",
	}

	results := ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/prop_ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'email' not found")
}

func TestValidateReferences_NestedStruct(t *testing.T) {
	t.Parallel()

	graph := GraphWith("name", "property")
	spec := nestedSpec{
		Name:  "test",
		Inner: &simpleRefSpec{Ref: "#property:name"},
		Items: []simpleRefSpec{
			{Ref: "#property:name"},
			{Ref: "#property:missing"},
		},
	}

	results := ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/items/1/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'missing' not found")
}

func TestValidateReferences_NilPointerField(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	spec := nestedSpec{
		Name:  "test",
		Inner: nil,
	}

	results := ValidateReferences(spec, graph)
	assert.Empty(t, results)
}

func TestValidateReferences_PointerString_Found(t *testing.T) {
	t.Parallel()

	graph := GraphWith("page_events", "category")
	catRef := "#category:page_events"
	spec := pointerStringSpec{CategoryRef: &catRef}

	results := ValidateReferences(spec, graph)
	assert.Empty(t, results)
}

func TestValidateReferences_PointerString_NotFound(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	catRef := "#category:nonexistent"
	spec := pointerStringSpec{CategoryRef: &catRef}

	results := ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/category", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced category 'nonexistent' not found")
}

func TestValidateReferences_PointerString_NilSkipped(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	spec := pointerStringSpec{CategoryRef: nil}

	results := ValidateReferences(spec, graph)
	assert.Empty(t, results)
}

func TestValidateReferences_SliceOfPointers(t *testing.T) {
	t.Parallel()

	graph := GraphWith("email", "property")
	spec := sliceOfPointersSpec{
		Rules: []*simpleRefSpec{
			{Ref: "#property:email"},
			nil,
			{Ref: "#property:missing"},
		},
	}

	results := ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/rules/2/$ref", results[0].Reference)
}

func TestValidateReferences_RecursiveStruct(t *testing.T) {
	t.Parallel()

	graph := GraphWith("parent", "property", "child", "property")
	spec := recursiveSpec{
		Ref: "#property:parent",
		Children: []*recursiveSpec{
			{
				Ref: "#property:child",
				Children: []*recursiveSpec{
					{Ref: "#property:grandchild"},
				},
			},
		},
	}

	results := ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/children/0/children/0/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'grandchild' not found")
}

func TestValidateReferences_NoRefTags(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	spec := noRefSpec{Name: "test", Description: "desc"}

	results := ValidateReferences(spec, graph)
	assert.Empty(t, results)
}

func TestValidateReferences_EmptyRefSkipped(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	spec := simpleRefSpec{Ref: ""}

	results := ValidateReferences(spec, graph)
	assert.Empty(t, results, "empty ref should be skipped")
}

func TestValidateReferences_TopLevelSlice(t *testing.T) {
	t.Parallel()

	graph := GraphWith("email", "property")
	specs := []simpleRefSpec{
		{Ref: "#property:email"},
		{Ref: "#property:missing"},
	}

	results := ValidateReferences(specs, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/1/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'missing' not found")
}

func TestValidateReferences_NonRefStringSkipped(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	spec := simpleRefSpec{Ref: "not-a-reference"}

	results := ValidateReferences(spec, graph)
	assert.Empty(t, results, "non-ref strings (no # prefix) should be skipped")
}

func TestValidateReferences_InvalidURNFormat(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	spec := simpleRefSpec{Ref: "#missing-colon"}

	results := ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "failed to parse reference")
}

// --- Helper function tests ---

func TestExtractLegacyPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{"property ref", "required,pattern=legacy_property_ref", "legacy_property_ref"},
		{"event ref", "required,pattern=legacy_event_ref", "legacy_event_ref"},
		{"category ref", "omitempty,pattern=legacy_category_ref", "legacy_category_ref"},
		{"non-legacy pattern", "required,pattern=display_name", ""},
		{"no pattern", "required,gte=3", ""},
		{"empty tag", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractLegacyPattern(tt.tag))
		})
	}
}

func TestParseURNRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		ref          string
		expectedType string
		expectedID   string
		wantErr      bool
	}{
		{"property ref", "#property:user_id", "property", "user_id", false},
		{"event ref", "#event:signup", "event", "signup", false},
		{"category ref", "#category:page_events", "category", "page_events", false},
		{"custom type ref", "#custom-type:Address", "custom-type", "Address", false},
		{"tracking plan ref", "#tracking-plan:my_tp", "tracking-plan", "my_tp", false},
		{"missing colon", "#property", "", "", true},
		{"empty kind", "#:user_id", "", "", true},
		{"empty id", "#property:", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceType, localID, err := ParseURNRef(tt.ref)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedType, resourceType)
				assert.Equal(t, tt.expectedID, localID)
			}
		})
	}
}
