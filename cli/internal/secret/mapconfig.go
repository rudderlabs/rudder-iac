package secret

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

// This file holds the map-config secret helpers shared by every provider whose
// config is a map[string]any with a known set of secret keys (destinations,
// accounts, …). They were originally private to the destination handler; lifting
// them here is the framework-level DRY that lets a new provider get
// destination-grade secret handling for free — no per-provider reflection, no
// struct-tag machinery.

// WrapKnownSecrets wraps each listed secret key that is already present in
// config as a *String holding the known local value. Pointer form survives the
// differ's struct→map decode. Absent secrets are not invented — requiredness is
// owned by the caller's spec model (e.g. validate tags), so a provider that
// needs an always-present secret must seed the key before calling.
func WrapKnownSecrets(config map[string]any, secretKeys []string) map[string]any {
	if config == nil || len(secretKeys) == 0 {
		return config
	}
	for _, key := range secretKeys {
		v, ok := config[key]
		if !ok {
			continue
		}
		raw := ""
		if s, ok := v.(string); ok {
			raw = s
		}
		s := New(raw)
		config[key] = &s
	}
	return config
}

// WrapUnknownSecrets marks each listed secret key that is already present in
// config as an unknown *String. Used when mapping remote state: APIs never
// return secret values, so a present-but-opaque key must always diff (see
// String.Diff). Absent keys stay absent — inventing them would force perpetual
// re-apply for conditional secrets that do not apply. A provider whose secret is
// unconditional must seed the key before calling (see accounts.MapRemoteToState).
func WrapUnknownSecrets(config map[string]any, secretKeys []string) map[string]any {
	if config == nil || len(secretKeys) == 0 {
		return config
	}
	for _, key := range secretKeys {
		if _, ok := config[key]; !ok {
			continue
		}
		s := NewUnknown()
		config[key] = &s
	}
	return config
}

// RevealSecrets returns a shallow copy of config with every listed secret key
// replaced by its Reveal() string. Run before marshalling to the wire so the
// real value is sent instead of a masked form. Keys absent from config are left
// alone.
func RevealSecrets(config map[string]any, secretKeys []string) map[string]any {
	if config == nil || len(secretKeys) == 0 {
		return config
	}
	out := maps.Clone(config)
	for _, key := range secretKeys {
		v, ok := out[key]
		if !ok {
			continue
		}
		switch s := v.(type) {
		case *String:
			if s == nil {
				out[key] = ""
				continue
			}
			out[key] = s.Reveal()
		case String:
			out[key] = s.Reveal()
		}
	}
	return out
}

// MaskSecrets replaces each listed secret key present in config with a masked
// token derived from externalID — a "{{ .VAR }}" reference under the variable
// substitution gate, otherwise a masked literal. Only keys present in config are
// touched; absent secrets are not invented.
func MaskSecrets(config map[string]any, externalID string, secretKeys []string) error {
	if config == nil || len(secretKeys) == 0 {
		return nil
	}
	prefix := strings.ToUpper(strings.ReplaceAll(externalID, "-", "_"))
	for _, key := range secretKeys {
		if _, ok := config[key]; !ok {
			continue
		}
		varName := fmt.Sprintf("%s_%s", prefix, strings.ToUpper(key))
		token, err := marshalToken(NewUnknown(WithVariableName(varName)))
		if err != nil {
			return fmt.Errorf("masking secret key %q: %w", key, err)
		}
		config[key] = token
	}
	return nil
}

// marshalToken JSON-marshals a String to its export string form (variable
// reference or masked literal).
func marshalToken(s String) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	var token string
	if err := json.Unmarshal(b, &token); err != nil {
		return "", err
	}
	return token, nil
}
