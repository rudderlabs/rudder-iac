// Package secret provides String, a self-redacting value type for sensitive
// data (API tokens, passwords, credentials). Every standard Go surface that
// prints, formats, logs, or serializes a String returns a masked form
// automatically; the real value escapes only through the explicit, greppable
// Reveal() accessor. Sensitivity becomes a property of the field's type, so a
// secret is masked once and stays masked at every surface it flows through.
package secret

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	unknownPlaceholder = "(unknown)"
	shortMask          = "***"
	// hintThreshold is the minimum length, in runes, at which masked() reveals the
	// last four runes as a spot-check hint; shorter values are masked whole ("***").
	// At exactly this length the tail is half the value, so 8 is the shortest secret
	// for which leaking a 4-rune tail is treated as acceptable.
	hintThreshold = 8
)

// String is an opaque, comparable holder for a sensitive value. The zero value
// is a valid "no secret yet". It has no exported fields, so the real value can
// only be obtained through Reveal().
type String struct {
	v string
	// unknown marks a secret whose real value we do not hold — typically because it
	// came from a remote API that never returns secrets. It lets callers distinguish
	// "no secret" from "a secret we cannot see", and it drives the always-re-apply
	// diff rule below.
	unknown bool
	// varName, when set via WithVariableName, names the substitution variable
	// that stands in for this secret during import scaffolding: the marshals
	// emit a "{{ .varName }}" reference instead of a masked literal, which the
	// user later resolves through a var file on apply.
	varName string
}

// Option configures a String at construction time.
type Option func(*String)

// WithVariableName attaches the substitution variable that stands in for the
// secret during import scaffolding, making the marshals emit a
// "{{ .name }}" reference instead of a masked literal. The provider building
// the export spec chooses the name from what it knows about the resource
// (e.g. resource type, external ID, field) — it is used verbatim, so it must
// satisfy the substitutor's variable grammar (^[A-Za-z_][A-Za-z0-9_]*$) —
// which is what keeps names deterministic and stable across re-imports.
//
// Scaffolding only works under the enableVarSubstitution experimental gate —
// without substitution the reference could never be resolved on apply — so
// with the gate off this option is a no-op and the secret exports as a masked
// literal, the pre-scaffolding behaviour.
func WithVariableName(name string) Option {
	return func(s *String) {
		if !config.GetConfig().ExperimentalFlags.EnableVarSubstitution {
			return
		}
		s.varName = name
	}
}

// New wraps a real, known value. The spec loader and provider spec-to-args
// conversion use it.
func New(v string, opts ...Option) String { return apply(String{v: v}, opts) }

// NewUnknown marks a secret whose real value we never hold. This is what
// MapRemoteToState constructs for a secret field, since backend APIs do not
// return secret values. An unknown secret always diffs (see Diff), so its
// resource is re-applied on every run.
func NewUnknown(opts ...Option) String { return apply(String{unknown: true}, opts) }

func apply(s String, opts []Option) String {
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// Reveal returns the real value. This is the only escape hatch and every call
// site is greppable, so revelations can be audited.
//
// On an unknown secret there is no real value to reveal, so this returns "".
// we may later harden Reveal to error rather than silently return "" for an unknown secret.
func (s String) Reveal() string { return s.v }

// IsZero reports whether there is no secret to send: an empty value that is not an
// unknown placeholder. Providers use it to decide whether to include the field in
// an outgoing request. New("") equals the zero value, so an empty secret is treated
// the same as "unset".
func (s String) IsZero() bool { return s.v == "" && !s.unknown }

// IsUnknown reports whether the value is a placeholder whose real value we do not
// hold (e.g. it came from a remote API that does not return secrets).
func (s String) IsUnknown() bool { return s.unknown }

// Diff reports whether a and b differ for plan purposes — true means the secret
// must be (re-)applied. An unknown secret always differs, even from another
// unknown: backend APIs never return secrets, so we can never confirm the remote
// matches the local value and must re-apply every run. Two known secrets differ
// only when their real values differ.
func (a String) Diff(b String) bool {
	if a.unknown || b.unknown {
		return true
	}
	return a.v != b.v
}

// masked is the single masked representation used by every formatting surface.
func (s String) masked() string {
	if s.unknown {
		return unknownPlaceholder
	}
	// Count and slice by rune so multi-byte values are measured by character and
	// the hint never splits a rune.
	if r := []rune(s.v); len(r) >= hintThreshold {
		return "****" + string(r[len(r)-4:])
	}
	return shortMask
}

// String implements fmt.Stringer, covering fmt.Print, log.Print, errors.New, and
// any other Stringer consumer.
func (s String) String() string { return s.masked() }

// Format implements fmt.Formatter so every verb (%v, %s, %q, %x, %+v, ...) routes
// through the masked form; without it, verbs like %q would reach the raw value
// via reflection.
func (s String) Format(f fmt.State, _ rune) { io.WriteString(f, s.masked()) }

// GoString implements fmt.GoStringer so %#v does not print the struct fields.
func (s String) GoString() string { return s.masked() }

// LogValue implements slog.LogValuer so structured logs mask automatically.
func (s String) LogValue() slog.Value { return slog.StringValue(s.masked()) }

// MarshalJSON redacts — unless a substitution variable is attached, in which
// case it emits the "{{ .VAR }}" reference: a remote never returns a secret's
// real value, so a masked literal in an exported spec would be useless, while
// a reference gives the user a var-file slot to supply it. This is
// load-bearing for export, which json.Marshals a spec before turning it into
// YAML. Outgoing API request bodies use plain string fields populated via
// Reveal(), so the real value still reaches the wire.
func (s String) MarshalJSON() ([]byte, error) {
	if s.varName != "" {
		return json.Marshal(s.token())
	}
	return json.Marshal(s.masked())
}

// MarshalYAML mirrors MarshalJSON for any direct YAML serialization of a String.
func (s String) MarshalYAML() (any, error) {
	if s.varName != "" {
		return s.token(), nil
	}
	return s.masked(), nil
}

func (s String) token() string { return fmt.Sprintf("{{ .%s }}", s.varName) }

// UnmarshalYAML reads a bare string from a spec, producing a known value.
func (s *String) UnmarshalYAML(node *yaml.Node) error {
	var raw string
	if err := node.Decode(&raw); err != nil {
		return fmt.Errorf("decoding secret: %w", err)
	}
	*s = New(raw)
	return nil
}

// UnmarshalJSON reads a bare string, keeping JSON spec paths symmetric with YAML.
func (s *String) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("decoding secret: %w", err)
	}
	*s = New(raw)
	return nil
}

// UnmarshalMapstructure implements mapstructure's Unmarshaler so every
// mapstructure decoder in the codebase accepts a secret field without
// per-decoder hook wiring. Spec maps carry secrets as bare strings (after YAML
// load and variable substitution); without this, mapstructure could not
// populate a secret.String, since the struct has no exported field to map onto.
func (s *String) UnmarshalMapstructure(input any) error {
	if existing, ok := input.(String); ok {
		*s = existing
		return nil
	}

	// reflect, not a type assertion, so named string types convert too.
	v := reflect.ValueOf(input)
	if v.Kind() != reflect.String {
		return fmt.Errorf("decoding secret: expected a string, got %T", input)
	}
	*s = New(v.String())
	return nil
}
