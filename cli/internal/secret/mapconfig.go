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
//
// Secret keys are dotted paths ("headers.to") where "." is purely a separator
// — there is no escape syntax. Upstream secret keys are camelCase identifiers,
// so a literal dot never occurs in a key; empty segments resolve to nothing.
//
// Ownership: the three helpers that produce a config (WrapKnownSecrets,
// WrapUnknownSecrets, RevealSecrets) copy it and never touch the caller's. That
// is not politeness — a nested path walks into inner containers that a caller's
// shallow copy still shares with its own source (destination's ApplyDefaults
// uses maps.Copy), so in-place mutation would reach back into the parsed spec.
// MaskSecrets is the exception: it reports an error rather than returning a
// config, and both call sites build the map themselves and already mutate it.

// WrapKnownSecrets returns a copy of config with each listed secret path that is
// present wrapped as a *String holding the known local value. Pointer form
// survives the differ's struct→map decode. Absent secrets are not invented —
// requiredness is owned by the caller's spec model (e.g. validate tags), so a
// provider that needs an always-present secret must seed the key before calling.
func WrapKnownSecrets(config map[string]any, secretKeys []string) map[string]any {
	return replaceSecrets(config, secretKeys, func(v any) any {
		raw, _ := v.(string)
		s := New(raw)
		return &s
	})
}

// WrapUnknownSecrets returns a copy of config with each listed secret path that
// is present marked as an unknown *String. Used when mapping remote state: APIs
// never return secret values, so a present-but-opaque key must always diff (see
// String.Diff). Absent keys stay absent — inventing them would force perpetual
// re-apply for conditional secrets that do not apply. A provider whose secret is
// unconditional must seed the key before calling (see accounts.MapRemoteToState).
func WrapUnknownSecrets(config map[string]any, secretKeys []string) map[string]any {
	return replaceSecrets(config, secretKeys, func(any) any {
		s := NewUnknown()
		return &s
	})
}

// RevealSecrets returns a copy of config with every listed secret path replaced
// by its Reveal() string. Run before marshalling to the wire so the real value is
// sent instead of a masked form. Keys absent from config are left alone.
func RevealSecrets(config map[string]any, secretKeys []string) map[string]any {
	return replaceSecrets(config, secretKeys, func(v any) any {
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

// MaskSecrets replaces each listed secret path present in config with a
// "{{ .VAR }}" reference derived from externalID. Only keys present in config are
// touched; absent secrets are not invented. Unlike the helpers above this mutates
// config in place, so the caller must own it.
func MaskSecrets(config map[string]any, externalID string, secretKeys []string) error {
	if config == nil || len(secretKeys) == 0 {
		return nil
	}
	prefix := strings.ToUpper(strings.ReplaceAll(externalID, "-", "_"))
	mask := func(owner map[string]any, key, varName string) error {
		token, err := marshalToken(NewUnknown(WithVariableName(varName)))
		if err != nil {
			return err
		}
		owner[key] = token
		return nil
	}
	for _, key := range secretKeys {
		if err := walkSecretPath(config, strings.Split(key, "."), prefix, mask); err != nil {
			return fmt.Errorf("masking secret key %q: %w", key, err)
		}
	}
	return nil
}

// replaceSecrets copies config and swaps every leaf the secret paths resolve to
// for replace's result. See the ownership note above for why it copies. With
// nothing to replace it returns config as-is — there is no mutation to isolate
// the caller from, so the copy would be pure cost.
func replaceSecrets(config map[string]any, secretKeys []string, replace func(any) any) map[string]any {
	if config == nil || len(secretKeys) == 0 {
		return config
	}
	out := cloneSecretConfig(config)
	visit := func(owner map[string]any, key, _ string) error {
		owner[key] = replace(owner[key])
		return nil
	}
	for _, key := range secretKeys {
		// visit never fails, so neither can the walk.
		_ = walkSecretPath(out, strings.Split(key, "."), "", visit)
	}
	return out
}

// secretVisitor is called once per leaf a secret path resolves to. owner[key] is
// the leaf; varName is its export variable name.
type secretVisitor func(owner map[string]any, key, varName string) error

// walkSecretPath resolves path against config and visits every leaf it reaches.
// A secret path carries no container information, so dispatch is on the runtime
// type at each step: map[string]any descends into the one nested object, []any
// and []map[string]any fan the remaining path out to every member — "headers.to"
// means the "to" of every header, matching upstream semantics. Non-container
// members are skipped, as are empty path segments.
//
// Both the mutating helpers and masking share this one traversal: two copies
// would have to be kept in sync by hand, and the two halves of a wrap→mask round
// trip silently stop matching the moment one of them learns a container shape
// the other does not.
func walkSecretPath(config map[string]any, path []string, varName string, visit secretVisitor) error {
	if len(path) == 0 || path[0] == "" {
		return nil
	}

	key := path[0]
	value, ok := config[key]
	if !ok {
		return nil
	}

	varName += "_" + strings.ToUpper(key)
	if len(path) == 1 {
		return visit(config, key, varName)
	}

	return walkNestedSecretPath(value, path[1:], varName, visit)
}

// walkNestedSecretPath continues the walk through whatever container value turns
// out to be. Slice members are indexed into the variable name so each exports its
// own reference; a map hop contributes no index.
func walkNestedSecretPath(value any, path []string, varName string, visit secretVisitor) error {
	switch v := value.(type) {
	case []any:
		for i, item := range v {
			if err := walkNestedSecretPath(item, path, varName+"_"+strconv.Itoa(i), visit); err != nil {
				return err
			}
		}
	case []map[string]any:
		for i := range v {
			if err := walkSecretPath(v[i], path, varName+"_"+strconv.Itoa(i), visit); err != nil {
				return err
			}
		}
	case map[string]any:
		return walkSecretPath(v, path, varName, visit)
	}
	return nil
}

func cloneSecretConfig(config map[string]any) map[string]any {
	out := make(map[string]any, len(config))
	for key, value := range config {
		out[key] = cloneSecretValue(value)
	}
	return out
}

// cloneSecretValue deep-copies exactly the container shapes walkSecretPath
// descends into. Anything else is shared, which is safe because the walk never
// reaches through it to mutate.
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
