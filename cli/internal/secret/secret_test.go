package secret

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

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
		{"remote redacted", NewRedacted(), "(remote-redacted)"},
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
		{"remote redacted", NewRedacted(), "", "(remote-redacted)"},
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

	assert.Equal(t, "", NewRedacted().Reveal())
}

func TestUnmarshalYAML(t *testing.T) {
	var s String
	require.NoError(t, yaml.Unmarshal([]byte("sk_live_abcd1234\n"), &s))
	assert.Equal(t, New("sk_live_abcd1234"), s)
	assert.False(t, s.IsRedacted())
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
	assert.False(t, s.IsRedacted())
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
	assert.False(t, New("a") == NewRedacted())
	assert.True(t, NewRedacted() == NewRedacted())
}

func TestIsZero(t *testing.T) {
	assert.True(t, String{}.IsZero())
	assert.True(t, New("").IsZero())
	assert.False(t, New("x").IsZero())
	assert.False(t, NewRedacted().IsZero())
}

func TestIsRedacted(t *testing.T) {
	assert.False(t, New("x").IsRedacted())
	assert.False(t, New("").IsRedacted())
	assert.True(t, NewRedacted().IsRedacted())
}
