package swift_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/swift"
	"github.com/rudderlabs/rudder-iac/typer/plan"
)

// Lists of enum (or object) custom types must be serialized element by
// element. Passing Swift values straight to the SDK makes the whole event fail
// to encode.
func TestGenerateSerializesListsOfCustomTypes(t *testing.T) {
	role := plan.CustomType{
		Name:   "user_role",
		Type:   plan.PrimitiveTypeString,
		Config: &plan.PropertyConfig{Enum: []any{"admin", "driver"}},
	}
	roleList := plan.CustomType{Name: "role_list", Type: plan.PrimitiveTypeArray, ItemType: role}
	roleGrid := plan.CustomType{Name: "role_grid", Type: plan.PrimitiveTypeArray, ItemType: roleList}
	email := plan.CustomType{Name: "email", Type: plan.PrimitiveTypeString}
	emailList := plan.CustomType{Name: "email_list", Type: plan.PrimitiveTypeArray, ItemType: email}

	props := map[string]plan.PropertySchema{
		// array property whose items are an enum custom type
		"user_roles": {Property: plan.Property{
			Name:      "user_roles",
			Types:     []plan.PropertyType{plan.PrimitiveTypeArray},
			ItemTypes: []plan.PropertyType{role},
		}, Required: true},
		// optional array property whose items are a list custom type
		"role_history": {Property: plan.Property{
			Name:      "role_history",
			Types:     []plan.PropertyType{plan.PrimitiveTypeArray},
			ItemTypes: []plan.PropertyType{roleList},
		}},
		// custom array types
		"roles":  {Property: plan.Property{Name: "roles", Types: []plan.PropertyType{roleList}}, Required: true},
		"grid":   {Property: plan.Property{Name: "grid", Types: []plan.PropertyType{roleGrid}}, Required: true},
		"emails": {Property: plan.Property{Name: "emails", Types: []plan.PropertyType{emailList}}, Required: true},
		// array property whose items are a primitive custom type
		"aliases": {Property: plan.Property{
			Name:      "aliases",
			Types:     []plan.PropertyType{plan.PrimitiveTypeArray},
			ItemTypes: []plan.PropertyType{email},
		}, Required: true},
	}
	tp := &plan.TrackingPlan{
		Name: "Lists",
		Rules: []plan.EventRule{{
			Event:   plan.Event{EventType: plan.EventTypeTrack, Name: "Roles Changed"},
			Section: plan.IdentitySectionProperties,
			Schema:  plan.ObjectSchema{Properties: props},
		}},
	}

	files, err := (&swift.Generator{}).Generate(tp, core.GenerateOptions{RudderCLIVersion: "1.0.0"}, swift.SwiftOptions{})
	require.NoError(t, err)
	require.Len(t, files, 1)
	out := files[0].Content

	assert.Contains(t, out, `"user_roles": userRoles.map { $0.rawValue }`)
	assert.Contains(t, out, `if let roleHistory = roleHistory { props["role_history"] = roleHistory.map { $0.map { $0.rawValue } } }`)
	assert.Contains(t, out, `"roles": roles.map { $0.rawValue }`)
	assert.Contains(t, out, `"grid": grid.map { $0.map { $0.rawValue } }`)
	// values that are already serializable are passed through unchanged
	assert.Contains(t, out, `"emails": emails`)
	assert.NotContains(t, out, `emails.map`)
	assert.Contains(t, out, `"aliases": aliases`)
	assert.NotContains(t, out, `aliases.map`)
}

func TestGeneratePublicImportOfSDK(t *testing.T) {
	tp := &plan.TrackingPlan{Name: "Empty"}
	files, err := (&swift.Generator{}).Generate(tp, core.GenerateOptions{RudderCLIVersion: "1.0.0"}, swift.SwiftOptions{})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Contains(t, files[0].Content, "#if compiler(>=6.0)\npublic import Foundation\npublic import RudderStackAnalytics\n#else\nimport Foundation\nimport RudderStackAnalytics\n#endif\n")
}
