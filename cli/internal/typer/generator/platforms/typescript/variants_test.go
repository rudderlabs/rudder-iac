package typescript

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildVariantGroup_StringDiscriminator(t *testing.T) {
	baseSchema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"kind": {Property: plan.Property{Name: "kind", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		},
	}
	variant := &plan.Variant{
		Discriminator: "kind",
		Cases: []plan.VariantCase{
			{
				DisplayName: "Alpha",
				Match:       []any{"alpha"},
				Description: "The alpha case",
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"score": {Property: plan.Property{Name: "score", Types: []plan.PropertyType{plan.PrimitiveTypeInteger}}, Required: true},
				}},
			},
			{
				DisplayName: "Beta",
				Match:       []any{"beta"},
				Description: "The beta case",
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}},
			},
		},
		DefaultSchema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"fallback": {Property: plan.Property{Name: "fallback", Types: []plan.PropertyType{plan.PrimitiveTypeBoolean}}},
		}},
	}

	group, err := buildVariantGroup("TestType", "Test variant", baseSchema, variant, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSVariantGroup{
		CaseInterfaces: []TSInterface{
			{
				Name:    "TestTypeCaseAlpha",
				Comment: "The alpha case",
				Properties: []TSInterfaceProperty{
					{Name: "kind", Type: `"alpha"`},
					{Name: "score", Type: "number"},
				},
			},
			{
				Name:    "TestTypeCaseBeta",
				Comment: "The beta case",
				Properties: []TSInterfaceProperty{
					{Name: "kind", Type: `"beta"`},
				},
			},
			{
				Name:    "TestTypeDefault",
				Comment: "Default case",
				Properties: []TSInterfaceProperty{
					{Name: "fallback", Type: "boolean", Optional: true},
					{Name: "kind", Type: `Exclude<string, "alpha" | "beta">`},
				},
			},
		},
		UnionAlias: TSTypeAlias{
			Alias:   "TestType",
			Type:    "TestTypeCaseAlpha | TestTypeCaseBeta | TestTypeDefault",
			Comment: "Test variant",
		},
	}, group)
}

func TestBuildVariantGroup_BooleanDiscriminator(t *testing.T) {
	activeProp := plan.Property{Name: "active", Types: []plan.PropertyType{plan.PrimitiveTypeBoolean}}
	baseSchema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"active": {Property: activeProp, Required: true},
		},
	}
	variant := &plan.Variant{
		Discriminator: "active",
		Cases: []plan.VariantCase{
			{
				Match:       []any{true},
				Description: "Active user",
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"email": {Property: plan.Property{Name: "email", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
				}},
			},
			{
				Match:       []any{false},
				Description: "Inactive user",
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"reason": {Property: plan.Property{Name: "reason", Types: []plan.PropertyType{plan.PrimitiveTypeString}}},
				}},
			},
		},
	}

	group, err := buildVariantGroup("UserAccess", "User access", baseSchema, variant, newTestRegistry())
	require.NoError(t, err)

	ifaceByName := make(map[string]TSInterface)
	for _, iface := range group.CaseInterfaces {
		ifaceByName[iface.Name] = iface
	}

	assert.Equal(t, TSInterface{
		Name:    "UserAccessCaseTrue",
		Comment: "Active user",
		Properties: []TSInterfaceProperty{
			{Name: "active", Type: "true"},
			{Name: "email", Type: "string"},
		},
	}, ifaceByName["UserAccessCaseTrue"])

	assert.Equal(t, TSInterface{
		Name:    "UserAccessCaseFalse",
		Comment: "Inactive user",
		Properties: []TSInterfaceProperty{
			{Name: "active", Type: "false"},
			{Name: "reason", Type: "string", Optional: true},
		},
	}, ifaceByName["UserAccessCaseFalse"])

	assert.Equal(t, TSInterface{
		Name:    "UserAccessDefault",
		Comment: "Default case",
		Properties: []TSInterfaceProperty{
			{Name: "active", Type: "Exclude<boolean, true | false>"},
		},
	}, ifaceByName["UserAccessDefault"], "default discriminator excludes covered values — both cases cover boolean exhaustively, so this resolves to never")

	assert.Equal(t, TSTypeAlias{
		Alias:   "UserAccess",
		Type:    "UserAccessCaseFalse | UserAccessCaseTrue | UserAccessDefault",
		Comment: "User access",
	}, group.UnionAlias)
}

func TestBuildVariantGroup_MultipleMatchValues(t *testing.T) {
	baseSchema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"mode": {Property: plan.Property{Name: "mode", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		},
	}
	variant := &plan.Variant{
		Discriminator: "mode",
		Cases: []plan.VariantCase{
			{
				Match:       []any{"fast", "turbo"},
				Description: "Fast modes",
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}},
			},
		},
	}

	group, err := buildVariantGroup("Config", "Config", baseSchema, variant, newTestRegistry())
	require.NoError(t, err)

	ifaceByName := make(map[string]TSInterface)
	for _, iface := range group.CaseInterfaces {
		ifaceByName[iface.Name] = iface
	}

	assert.Equal(t, TSInterface{
		Name:    "ConfigCaseFast",
		Comment: "Fast modes",
		Properties: []TSInterfaceProperty{
			{Name: "mode", Type: `"fast"`},
		},
	}, ifaceByName["ConfigCaseFast"])

	assert.Equal(t, TSInterface{
		Name:    "ConfigCaseTurbo",
		Comment: "Fast modes",
		Properties: []TSInterfaceProperty{
			{Name: "mode", Type: `"turbo"`},
		},
	}, ifaceByName["ConfigCaseTurbo"])

	assert.Contains(t, ifaceByName, "ConfigDefault")
}

func TestBuildVariantGroup_NoDefaultSchema(t *testing.T) {
	baseSchema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"kind": {Property: plan.Property{Name: "kind", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		},
	}
	variant := &plan.Variant{
		Discriminator: "kind",
		Cases: []plan.VariantCase{{
			Match:       []any{"a"},
			Description: "Case A",
			Schema:      plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}},
		}},
	}

	group, err := buildVariantGroup("Foo", "", baseSchema, variant, newTestRegistry())
	require.NoError(t, err)

	ifaceByName := make(map[string]TSInterface)
	for _, iface := range group.CaseInterfaces {
		ifaceByName[iface.Name] = iface
	}

	assert.Equal(t, TSInterface{
		Name:    "FooDefault",
		Comment: "Default case",
		Properties: []TSInterfaceProperty{
			{Name: "kind", Type: `Exclude<string, "a">`},
		},
	}, ifaceByName["FooDefault"])
}

func TestBuildVariantGroup_CasePropertyOverridesRequired(t *testing.T) {
	baseSchema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"kind":  {Property: plan.Property{Name: "kind", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
			"label": {Property: plan.Property{Name: "label", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: false},
		},
	}
	variant := &plan.Variant{
		Discriminator: "kind",
		Cases: []plan.VariantCase{{
			Match: []any{"special"},
			Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"label": {Property: plan.Property{Name: "label", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
			}},
		}},
	}

	group, err := buildVariantGroup("Item", "", baseSchema, variant, newTestRegistry())
	require.NoError(t, err)

	ifaceByName := make(map[string]TSInterface)
	for _, iface := range group.CaseInterfaces {
		ifaceByName[iface.Name] = iface
	}

	assert.Equal(t, TSInterface{
		Name: "ItemCaseSpecial",
		Properties: []TSInterfaceProperty{
			{Name: "kind", Type: `"special"`},
			{Name: "label", Type: "string", Optional: false},
		},
	}, ifaceByName["ItemCaseSpecial"], "case upgrades base optional → required via OR")
}

func TestBuildVariantGroup_EnumDiscriminatorInDefault(t *testing.T) {
	enumProp := plan.Property{
		Name:  "device_type",
		Types: []plan.PropertyType{plan.PrimitiveTypeString},
		Config: &plan.PropertyConfig{
			Enum: []any{"mobile", "desktop"},
		},
	}
	baseSchema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"device_type": {Property: enumProp, Required: true},
		},
	}
	variant := &plan.Variant{
		Discriminator: "device_type",
		Cases: []plan.VariantCase{{
			Match:  []any{"mobile"},
			Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}},
		}},
	}

	nr := newTestRegistry()
	_, err := getOrRegisterPropertyEnumName(&enumProp, nr)
	require.NoError(t, err)

	group, err := buildVariantGroup("Event", "", baseSchema, variant, nr)
	require.NoError(t, err)

	ifaceByName := make(map[string]TSInterface)
	for _, iface := range group.CaseInterfaces {
		ifaceByName[iface.Name] = iface
	}

	assert.Equal(t, TSInterface{
		Name:    "EventDefault",
		Comment: "Default case",
		Properties: []TSInterfaceProperty{
			{Name: "deviceType", Type: `Exclude<PropertyDeviceType, "mobile">`},
		},
	}, ifaceByName["EventDefault"], "default narrows the enum alias to values no named case covers")

	assert.Equal(t, TSInterface{
		Name: "EventCaseMobile",
		Properties: []TSInterfaceProperty{
			{Name: "deviceType", Type: `"mobile"`},
		},
	}, ifaceByName["EventCaseMobile"], "named case uses literal")
}

func TestBuildVariantGroup_MissingDiscriminator(t *testing.T) {
	baseSchema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"name": {Property: plan.Property{Name: "name", Types: []plan.PropertyType{plan.PrimitiveTypeString}}},
		},
	}
	variant := &plan.Variant{
		Discriminator: "kind",
		Cases:         []plan.VariantCase{{Match: []any{"a"}, Schema: plan.ObjectSchema{}}},
	}

	group, err := buildVariantGroup("Foo", "", baseSchema, variant, newTestRegistry())
	require.NoError(t, err, "missing discriminator is silently accepted, matching Kotlin")
	assert.Equal(t, "FooCaseA | FooDefault", group.UnionAlias.Type)
}

func TestFormatMatchValueForName(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"search", "search"},
		{"mobile", "mobile"},
		{true, "True"},
		{false, "False"},
		{42, "42"},
		{3.14, "3.14"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatMatchValueForName(tt.input))
	}
}

func TestFormatVariantCaseName(t *testing.T) {
	tests := []struct {
		parent   string
		match    any
		expected string
	}{
		{"CustomTypePageContext", "search", "CustomTypePageContextCaseSearch"},
		{"CustomTypeUserAccess", true, "CustomTypeUserAccessCaseTrue"},
		{"CustomTypeUserAccess", false, "CustomTypeUserAccessCaseFalse"},
		{"EventWithVariants", "mobile", "EventWithVariantsCaseMobile"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatVariantCaseName(tt.parent, tt.match))
	}
}
