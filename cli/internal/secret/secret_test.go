package secret

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	realSecret   = "sk_live_abcd1234"
	maskedSecret = "****1234"
)

func TestString_Masking(t *testing.T) {
	tests := []struct {
		name string
		in   String
		want string
	}{
		{"long value shows last four", New(realSecret), maskedSecret},
		{"exactly eight shows last four", New("abcd1234"), "****1234"},
		{"short value is fully masked", New("hunter2"), "***"},
		{"empty value is masked", New(""), "***"},
		{"unknown secret", NewUnknown(), "(unknown)"},
		// Masking counts characters, not bytes: the hint is the last four whole
		// runes and the threshold counts runes, so multi-byte values are not split.
		{"multibyte value shows last four runes", New("1234café"), "****café"},
		{"few runes but many bytes is fully masked", New("éééé"), "***"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.String())
		})
	}
}

// TestString_RedactingSurfaces proves that every surface that prints, formats,
// logs, or serializes the value returns the masked form and never the real one.
func TestString_RedactingSurfaces(t *testing.T) {
	surfaces := []struct {
		name   string
		render func(t *testing.T, s String) string
	}{
		{"String", func(t *testing.T, s String) string { return s.String() }},
		{"GoString", func(t *testing.T, s String) string { return s.GoString() }},
		{"fmt %v", func(t *testing.T, s String) string { return fmt.Sprintf("%v", s) }},
		{"fmt %s", func(t *testing.T, s String) string { return fmt.Sprintf("%s", s) }},
		{"fmt %q", func(t *testing.T, s String) string { return fmt.Sprintf("%q", s) }},
		{"fmt %x", func(t *testing.T, s String) string { return fmt.Sprintf("%x", s) }},
		{"fmt %+v", func(t *testing.T, s String) string { return fmt.Sprintf("%+v", s) }},
		{"fmt %#v", func(t *testing.T, s String) string { return fmt.Sprintf("%#v", s) }},
		{"slog LogValue", func(t *testing.T, s String) string { return s.LogValue().String() }},
		{"MarshalJSON", func(t *testing.T, s String) string {
			b, err := json.Marshal(s)
			require.NoError(t, err)
			return string(b)
		}},
		{"MarshalYAML", func(t *testing.T, s String) string {
			b, err := yaml.Marshal(s)
			require.NoError(t, err)
			return string(b)
		}},
	}

	inputs := []struct {
		name       string
		in         String
		real       string // empty means there is no real value to leak
		wantMasked string
	}{
		{"real value", New(realSecret), realSecret, maskedSecret},
		{"unknown secret", NewUnknown(), "", "(unknown)"},
	}

	for _, in := range inputs {
		for _, sf := range surfaces {
			t.Run(in.name+"/"+sf.name, func(t *testing.T) {
				out := sf.render(t, in.in)
				assert.Contains(t, out, in.wantMasked)
				if in.real != "" {
					assert.NotContains(t, out, in.real)
				}
			})
		}
	}
}

func TestReveal(t *testing.T) {
	s := New(realSecret)
	// Reveal must return the real value even after redacting surfaces ran.
	_ = s.String()
	_ = fmt.Sprintf("%v %s %q %#v", s, s, s, s)
	assert.Equal(t, realSecret, s.Reveal())

	assert.Equal(t, "", NewUnknown().Reveal())
}

func TestUnmarshalYAML(t *testing.T) {
	var s String
	require.NoError(t, yaml.Unmarshal([]byte("sk_live_abcd1234\n"), &s))
	assert.Equal(t, New("sk_live_abcd1234"), s)
	assert.False(t, s.IsUnknown())
}

func TestUnmarshalYAML_InStruct(t *testing.T) {
	type spec struct {
		Name  string `yaml:"name"`
		Token String `yaml:"token"`
	}
	var got spec
	require.NoError(t, yaml.Unmarshal([]byte("name: main\ntoken: sk_live_abcd1234\n"), &got))
	assert.Equal(t, spec{Name: "main", Token: New("sk_live_abcd1234")}, got)
}

func TestUnmarshalJSON(t *testing.T) {
	var s String
	require.NoError(t, json.Unmarshal([]byte(`"sk_live_abcd1234"`), &s))
	assert.Equal(t, New("sk_live_abcd1234"), s)
	assert.False(t, s.IsUnknown())
}

// TestMarshalJSON_InStruct_Redacts mirrors the export path, which json.Marshals a
// spec struct before turning it into the exported YAML.
func TestMarshalJSON_InStruct_Redacts(t *testing.T) {
	type spec struct {
		Name  string `json:"name"`
		Token String `json:"token"`
	}
	b, err := json.Marshal(spec{Name: "main", Token: New(realSecret)})
	require.NoError(t, err)
	assert.NotContains(t, string(b), realSecret)
	assert.Contains(t, string(b), maskedSecret)
}

func TestLogValue_Integration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	logger.Info("creating source", "token", New(realSecret))
	out := buf.String()
	assert.Contains(t, out, maskedSecret)
	assert.NotContains(t, out, realSecret)
}

func TestEquality(t *testing.T) {
	assert.True(t, New("a") == New("a"))
	assert.False(t, New("a") == New("b"))
	assert.False(t, New("a") == NewUnknown())
	assert.True(t, NewUnknown() == NewUnknown())
}

func TestIsZero(t *testing.T) {
	assert.True(t, String{}.IsZero())
	assert.True(t, New("").IsZero())
	assert.False(t, New("x").IsZero())
	assert.False(t, NewUnknown().IsZero())
}

func TestIsUnknown(t *testing.T) {
	assert.False(t, New("x").IsUnknown())
	assert.False(t, New("").IsUnknown())
	assert.True(t, NewUnknown().IsUnknown())
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name string
		a, b String
		want bool
	}{
		{"equal values do not diff", New("hunter2"), New("hunter2"), false},
		{"different values diff", New("hunter2"), New("hunter3"), true},
		{"local value vs unknown remote always diffs", New("hunter2"), NewUnknown(), true},
		{"unknown remote vs local value always diffs", NewUnknown(), New("hunter2"), true},
		{"both unknown always diff", NewUnknown(), NewUnknown(), true},
		{"empty equals empty", New(""), New(""), false},
		{"empty vs set diffs", New(""), New("hunter2"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Diff(tt.b))
		})
	}
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

// With a variable name attached, the marshals emit the reference — that is the
// whole point: the token must survive into exported YAML where a plain secret
// would have been redacted into a useless literal.
func TestWithVariableName_MarshalsVariableReference(t *testing.T) {
	enableVarSubstitution(t)

	s := NewUnknown(WithVariableName("BOOK_THE_HOBBIT_ACCESS_KEY"))

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Equal(t, `"{{ .BOOK_THE_HOBBIT_ACCESS_KEY }}"`, string(jsonBytes))

	yamlVal, err := s.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, "{{ .BOOK_THE_HOBBIT_ACCESS_KEY }}", yamlVal)
}

// Even when a known value carries a name, only the reference is emitted by the
// marshals and every formatting surface keeps masking — the real value must
// never escape.
func TestWithVariableName_NeverLeaksValue(t *testing.T) {
	enableVarSubstitution(t)

	s := New("hunter2-but-long", WithVariableName("ACCESS_KEY"))

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.NotContains(t, string(jsonBytes), "hunter2")

	assert.NotContains(t, fmt.Sprintf("%v %s %q %#v", s, s, s, s), "hunter2")
	assert.Equal(t, "****long", s.String())
}

// With the gate off the option is a no-op, so the secret serializes as a
// masked literal — the pre-scaffolding behaviour.
func TestWithVariableName_GateOff(t *testing.T) {
	s := NewUnknown(WithVariableName("ACCESS_KEY"))
	assert.Equal(t, NewUnknown(), s)

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Equal(t, `"(unknown)"`, string(jsonBytes))
}

// Loading a spec value over a named secret produces a plain known secret; the
// variable name only ever exists on the export path.
func TestWithVariableName_UnmarshalResetsName(t *testing.T) {
	enableVarSubstitution(t)

	s := NewUnknown(WithVariableName("ACCESS_KEY"))
	require.NoError(t, yaml.Unmarshal([]byte(`"real-value"`), &s))
	assert.Equal(t, New("real-value"), s)

	s = NewUnknown(WithVariableName("ACCESS_KEY"))
	require.NoError(t, json.Unmarshal([]byte(`"real-value"`), &s))
	assert.Equal(t, New("real-value"), s)
}

// The name is stored verbatim — validation happens at marshal time.
func TestWithVariableName_UsedVerbatim(t *testing.T) {
	enableVarSubstitution(t)

	assert.Equal(t,
		String{varName: "Book_Access_Key_2"},
		New("", WithVariableName("Book_Access_Key_2")),
	)
}

// A name outside the substitutor's variable grammar fails the marshals, so a
// bad name (derived from user-controlled external IDs) errors when the export
// spec is generated — not two steps later when apply rejects the token.
func TestWithVariableName_InvalidNameFailsMarshal(t *testing.T) {
	enableVarSubstitution(t)

	s := NewUnknown(WithVariableName("9-starts-with-digit"))

	_, err := json.Marshal(s)
	require.ErrorContains(t, err, "9-starts-with-digit")

	_, err = s.MarshalYAML()
	require.ErrorContains(t, err, "9-starts-with-digit")
}
