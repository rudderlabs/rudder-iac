package export

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sourceItem struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	AccessToken secret.String `json:"accessToken"`
}

type nestedConfig struct {
	Password secret.String `json:"password"`
}

type exportSpec struct {
	Sources   []sourceItem             `json:"sources"`
	Config    nestedConfig             `json:"config"`
	Optional  *secret.String           `json:"optional,omitempty"`
	ByName    map[string]secret.String `json:"byName,omitempty"`
	Freeform  map[string]any           `json:"freeform,omitempty"`
	PlainName string                   `json:"plainName"`
}

func enableVarSubstitution(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", true)
	viper.Set("flags.enableVarSubstitution", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})
}

func TestScaffoldSecretRefs(t *testing.T) {
	optional := secret.NewUnknown()
	spec := &exportSpec{
		Sources: []sourceItem{
			{ID: "src-main", Name: "Main", AccessToken: secret.NewUnknown()},
			{Name: "no id, falls back to index", AccessToken: secret.NewUnknown()},
		},
		Config:   nestedConfig{Password: secret.NewUnknown()},
		Optional: &optional,
		ByName:   map[string]secret.String{"writeKey": secret.NewUnknown()},
		Freeform: map[string]any{"apiKey": secret.NewUnknown(), "plain": "untouched"},
	}

	scaffoldSecretRefs(spec, varPathPrefix("retl/sources.yaml"))

	assert.Equal(t, &exportSpec{
		Sources: []sourceItem{
			{ID: "src-main", Name: "Main", AccessToken: secret.NewRef("{{ .RETL_SOURCES_SOURCES_SRC_MAIN_ACCESS_TOKEN }}")},
			{Name: "no id, falls back to index", AccessToken: secret.NewRef("{{ .RETL_SOURCES_SOURCES_1_ACCESS_TOKEN }}")},
		},
		Config:   nestedConfig{Password: secret.NewRef("{{ .RETL_SOURCES_CONFIG_PASSWORD }}")},
		Optional: ptr(secret.NewRef("{{ .RETL_SOURCES_OPTIONAL }}")),
		ByName:   map[string]secret.String{"writeKey": secret.NewRef("{{ .RETL_SOURCES_BY_NAME_WRITE_KEY }}")},
		Freeform: map[string]any{"apiKey": secret.NewRef("{{ .RETL_SOURCES_FREEFORM_API_KEY }}"), "plain": "untouched"},
	}, spec)
}

func TestScaffoldSecretRefs_KnownValuesAlsoTokenized(t *testing.T) {
	// Even a known value must never be serialized on export — it would be
	// masked into a useless literal anyway. Every secret becomes a reference.
	spec := &exportSpec{Config: nestedConfig{Password: secret.New("real")}}
	scaffoldSecretRefs(spec, varPathPrefix("spec.yaml"))
	assert.Equal(t, secret.NewRef("{{ .SPEC_CONFIG_PASSWORD }}"), spec.Config.Password)
}

func TestVarName(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want string
	}{
		{"camelCase splits", []string{"accessKey"}, "ACCESS_KEY"},
		{"kebab and dirs", []string{"retl", "sql-models", "my-source"}, "RETL_SQL_MODELS_MY_SOURCE"},
		{"special chars collapse", []string{"a..b", "c d"}, "A_B_C_D"},
		{"leading digit gets prefix", []string{"1984", "accessKey"}, "_1984_ACCESS_KEY"},
		{"empty components dropped", []string{"-", "id"}, "ID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, varName(tt.path))
		})
	}
}

func TestToMap_GateOn_EmitsVariableReferences(t *testing.T) {
	enableVarSubstitution(t)

	data := &SpecExportData[exportSpec]{
		RelativePath: "books/books.yaml",
		Data: &exportSpec{
			Sources:   []sourceItem{{ID: "book-a", Name: "Book A", AccessToken: secret.NewUnknown()}},
			PlainName: "stays plain",
		},
	}

	m, err := data.ToMap()
	require.NoError(t, err)

	sources := m["sources"].([]any)
	source := sources[0].(map[string]any)
	assert.Equal(t, "{{ .BOOKS_BOOKS_SOURCES_BOOK_A_ACCESS_TOKEN }}", source["accessToken"])
	assert.Equal(t, "stays plain", m["plainName"])
}

func TestToMap_GateOff_KeepsMaskedLiteral(t *testing.T) {
	data := &SpecExportData[exportSpec]{
		RelativePath: "books/books.yaml",
		Data: &exportSpec{
			Sources: []sourceItem{{ID: "book-a", Name: "Book A", AccessToken: secret.NewUnknown()}},
		},
	}

	m, err := data.ToMap()
	require.NoError(t, err)

	sources := m["sources"].([]any)
	source := sources[0].(map[string]any)
	assert.Equal(t, "(unknown)", source["accessToken"])
}

func ptr[T any](v T) *T { return &v }
