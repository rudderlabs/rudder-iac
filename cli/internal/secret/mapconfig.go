package secret

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// This file holds the map-config secret helpers shared by every provider whose
// config is a map[string]any with a known set of secret keys (destinations,
// accounts, …). They were originally private to the destination handler; lifting
// them here is the framework-level DRY that lets a new provider get
// destination-grade secret handling for free — no per-provider reflection, no
// struct-tag machinery.

// WrapKnownSecrets wraps each listed secret path that is already present in
// config as a *String holding the known local value. Pointer form survives the
// differ's struct→map decode. Absent secrets are not invented — requiredness is
// owned by the caller's spec model (e.g. validate tags), so a provider that
// needs an always-present secret must seed the key before calling.
func WrapKnownSecrets(config map[string]any, secretKeys []string) map[string]any {
	if config == nil || len(secretKeys) == 0 {
		return config
	}
	for _, key := range secretKeys {
		wrapKnownSecret(config, strings.Split(key, "."))
	}
	return config
}

// WrapUnknownSecrets marks each listed secret path that is already present in
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
		wrapUnknownSecret(config, strings.Split(key, "."))
	}
	return config
}

// RevealSecrets returns a copy of config with every listed secret path replaced
// by its Reveal() string. Run before marshalling to the wire so the real value is
// sent instead of a masked form. Keys absent from config are left alone.
func RevealSecrets(config map[string]any, secretKeys []string) map[string]any {
	if config == nil || len(secretKeys) == 0 {
		return config
	}
	out := cloneSecretConfig(config)
	for _, key := range secretKeys {
		revealSecret(out, strings.Split(key, "."))
	}
	return out
}

// MaskSecrets replaces each listed secret path present in config with a masked
// token derived from externalID — a "{{ .VAR }}" reference under the variable
// substitution gate, otherwise a masked literal. Only keys present in config are
// touched; absent secrets are not invented.
func MaskSecrets(config map[string]any, externalID string, secretKeys []string) error {
	if config == nil || len(secretKeys) == 0 {
		return nil
	}
	prefix := strings.ToUpper(strings.ReplaceAll(externalID, "-", "_"))
	for _, key := range secretKeys {
		path := strings.Split(key, ".")
		if !secretPathExists(config, path) {
			continue
		}

		if err := maskSecret(config, path, prefix); err != nil {
			return fmt.Errorf("masking secret key %q: %w", key, err)
		}
	}
	return nil
}

func wrapKnownSecret(config map[string]any, path []string) {
	setSecretValue(config, path, func(v any) any {
		raw := ""
		if s, ok := v.(string); ok {
			raw = s
		}
		s := New(raw)
		return &s
	})
}

func wrapUnknownSecret(config map[string]any, path []string) {
	setSecretValue(config, path, func(any) any {
		s := NewUnknown()
		return &s
	})
}

func revealSecret(config map[string]any, path []string) {
	setSecretValue(config, path, func(v any) any {
		switch s := v.(type) {
		case *String:
			if s == nil {
				return ""
			}
			return s.Reveal()
		case String:
			return s.Reveal()
		default:
			return v
		}
	})
}

func maskSecret(config map[string]any, path []string, prefix string) error {
	return maskSecretValue(config, path, prefix, nil)
}

func maskSecretValue(config map[string]any, path []string, prefix string, nameParts []string) error {
	if len(path) == 0 || path[0] == "" {
		return nil
	}

	key := path[0]
	value, ok := config[key]
	if !ok {
		return nil
	}

	nextNameParts := appendNamePart(nameParts, key)
	if len(path) == 1 {
		varName := secretVariableName(prefix, strings.Join(nextNameParts, "."))
		token, err := marshalToken(NewUnknown(WithVariableName(varName)))
		if err != nil {
			return err
		}
		config[key] = token
		return nil
	}

	return maskNestedSecretValue(value, path[1:], prefix, nextNameParts)
}

func maskNestedSecretValue(value any, path []string, prefix string, nameParts []string) error {
	switch v := value.(type) {
	case []any:
		for i, item := range v {
			if err := maskNestedSecretValue(item, path, prefix, appendNamePart(nameParts, strconv.Itoa(i))); err != nil {
				return err
			}
		}
	case []map[string]any:
		for i := range v {
			if err := maskSecretValue(v[i], path, prefix, appendNamePart(nameParts, strconv.Itoa(i))); err != nil {
				return err
			}
		}
	case map[string]any:
		return maskSecretValue(v, path, prefix, nameParts)
	}
	return nil
}

func appendNamePart(parts []string, part string) []string {
	out := make([]string, 0, len(parts)+1)
	out = append(out, parts...)
	out = append(out, part)
	return out
}

func setSecretValue(config map[string]any, path []string, replace func(any) any) {
	if len(path) == 0 || path[0] == "" {
		return
	}

	key := path[0]
	value, ok := config[key]
	if !ok {
		return
	}

	if len(path) == 1 {
		config[key] = replace(value)
		return
	}

	setNestedSecretValue(value, path[1:], replace)
}

func setNestedSecretValue(value any, path []string, replace func(any) any) {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			setNestedSecretValue(item, path, replace)
		}
	case []map[string]any:
		for i := range v {
			setSecretValue(v[i], path, replace)
		}
	case map[string]any:
		setSecretValue(v, path, replace)
	}
}

func secretPathExists(config map[string]any, path []string) bool {
	if len(path) == 0 || path[0] == "" {
		return false
	}

	value, ok := config[path[0]]
	if !ok {
		return false
	}

	if len(path) == 1 {
		return true
	}

	return nestedSecretPathExists(value, path[1:])
}

func nestedSecretPathExists(value any, path []string) bool {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if nestedSecretPathExists(item, path) {
				return true
			}
		}
	case []map[string]any:
		for i := range v {
			if secretPathExists(v[i], path) {
				return true
			}
		}
	case map[string]any:
		return secretPathExists(v, path)
	}
	return false
}

func cloneSecretConfig(config map[string]any) map[string]any {
	out := make(map[string]any, len(config))
	for key, value := range config {
		out[key] = cloneSecretValue(value)
	}
	return out
}

func cloneSecretValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneSecretConfig(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneSecretValue(item)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(v))
		for i, item := range v {
			out[i] = cloneSecretConfig(item)
		}
		return out
	default:
		return value
	}
}

func secretVariableName(prefix, key string) string {
	key = strings.NewReplacer(".", "_", "-", "_").Replace(key)
	return fmt.Sprintf("%s_%s", prefix, strings.ToUpper(key))
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
