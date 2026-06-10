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
}

// New wraps a real, known value. The spec loader and provider spec-to-args
// conversion use it.
func New(v string) String { return String{v: v} }

// NewUnknown marks a secret whose real value we never hold. This is what
// MapRemoteToState constructs for a secret field, since backend APIs do not
// return secret values. An unknown secret always diffs (see Diff), so its
// resource is re-applied on every run.
func NewUnknown() String { return String{unknown: true} }

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

// MarshalJSON redacts. This is load-bearing for export, which json.Marshals a
// spec before turning it into YAML. Outgoing API request bodies use plain string
// fields populated via Reveal(), so the real value still reaches the wire.
func (s String) MarshalJSON() ([]byte, error) { return json.Marshal(s.masked()) }

// MarshalYAML redacts any direct YAML serialization of a String.
func (s String) MarshalYAML() (any, error) { return s.masked(), nil }

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
