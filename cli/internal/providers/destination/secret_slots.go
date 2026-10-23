package destination

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

// secretSlot is the value of one destination secret key, as the spec is
// written before variable substitution.
type secretSlot struct {
	file string
	// destination names the spec in messages: its id, or its file when the id
	// is empty or holds a reference, since such ids do not tell specs apart.
	destination string
	// key is the secret key with list indices, such as "headers[0].to".
	key string
	// reference is the JSON pointer to the value in the spec file.
	reference string
	// variable is the variable the value references when the value is exactly
	// one reference with no default, and empty otherwise.
	variable string
	// defaulted is the variable the value references when the value is
	// exactly one reference with a default, and empty otherwise.
	defaulted string
	// suggested is the variable import would write for the key. It is empty
	// when the spec id is empty, holds a reference, or makes no valid variable
	// name.
	suggested string
}

// readSecretSlots returns the secret key values of the destination spec in
// source. A spec it cannot read has none, and the syntax rule reports it. That
// includes a spec whose type is unregistered or a reference, since the type
// decides which keys are secret. A key holding null, an empty string or a
// value read from each event is left out, since it holds no secret.
func readSecretSlots(registry *definitions.Registry, file string, source []byte) []secretSlot {
	replaced := varsubst.ReplaceReferences(source)

	spec, ok := readSourceSpec(replaced.Data)
	if !ok {
		return nil
	}
	secretKeys := specSecretKeys(registry, spec)
	if len(secretKeys) == 0 {
		return nil
	}

	// An id holding a reference reads as a placeholder here, which would leak
	// into the names and the messages, so it counts as no id.
	id := spec.id
	if replaced.HoldsReference(id) {
		id = ""
	}
	destination := fmt.Sprintf("destination %q", id)
	if id == "" {
		destination = fmt.Sprintf("the destination in %q", file)
	}

	// MaskSecrets rewrites each secret value as the reference import writes for
	// it, so masking a second copy of the config yields the suggested names
	// without restating the naming rule here. A missing id, which the syntax
	// rule reports, would give names with no prefix, so it gets none.
	masked, _ := readSourceSpec(replaced.Data)
	if id == "" || secret.MaskSecrets(masked.config, id, secretKeys) != nil {
		masked.config = nil
	}

	var slots []secretSlot
	for _, value := range secretValues(spec.config, masked.config, secretKeys) {
		if value.value == nil || value.value == "" || readsEvent(value.value) {
			continue
		}

		slot := secretSlot{
			file:        file,
			destination: destination,
			key:         value.key,
			reference:   "/spec/config" + value.reference,
			suggested:   referencedVariable(value.masked),
		}
		if s, ok := value.value.(string); ok {
			if ref, ok := replaced.Reference(s); ok {
				if ref.HasDefault {
					slot.defaulted = ref.Name
				} else {
					slot.variable = ref.Name
				}
			}
		}
		slots = append(slots, slot)
	}
	return slots
}

// specSecretKeys returns the secret keys of the definition the spec names.
// When definition_version names no registered version, because it is missing,
// a reference or wrong, the syntax rule reports that. The keys of every
// registered version of the type are used then, so a literal secret is still
// reported.
func specSecretKeys(registry *definitions.Registry, spec sourceSpec) []string {
	if def, err := registry.Get(spec.destinationType, spec.definitionVersion); err == nil {
		return def.SecretKeys()
	}

	versions, err := registry.Versions(spec.destinationType)
	if err != nil {
		return nil
	}
	var secretKeys []string
	for _, version := range versions {
		def, err := registry.Get(spec.destinationType, version)
		if err != nil {
			continue
		}
		for _, key := range def.SecretKeys() {
			if !slices.Contains(secretKeys, key) {
				secretKeys = append(secretKeys, key)
			}
		}
	}
	return secretKeys
}

// readsEvent reports whether value is one the destination reads from each
// event at delivery, so it holds no secret: a JSONPath such as $.messageId, or
// a template such as {{ message.userId || "" }} with an empty fallback. A
// template with a fallback is still reported, since the fallback is a literal.
func readsEvent(value any) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	if strings.HasPrefix(s, "$.") || strings.HasPrefix(s, "$[") {
		return true
	}
	if !definitions.IsTemplateConfigValue(s) {
		return false
	}

	// The template pattern takes the fallback after the last "||".
	fallback := strings.TrimSpace(strings.TrimSuffix(s[strings.LastIndex(s, "||")+len("||"):], "}}"))
	return fallback == "" || fallback == `""` || fallback == "''"
}

// sourceSpec holds the fields of a destination spec that locate its secrets.
type sourceSpec struct {
	id                string
	destinationType   string
	definitionVersion int64
	config            map[string]any
}

// readSourceSpec reads a spec whose references were replaced by placeholders.
// Any field may hold a placeholder where the spec expects another type, so each
// is read on its own instead of decoding the whole spec.
func readSourceSpec(data []byte) (sourceSpec, bool) {
	parsed, err := specs.New(data)
	if err != nil {
		return sourceSpec{}, false
	}

	var (
		id, _                = parsed.Spec["id"].(string)
		destinationType, _   = parsed.Spec["type"].(string)
		definitionVersion, _ = parsed.Spec["definition_version"].(int)
		config, _            = parsed.Spec["config"].(map[string]any)
	)
	return sourceSpec{
		id:                id,
		destinationType:   destinationType,
		definitionVersion: int64(definitionVersion),
		config:            config,
	}, true
}

// secretValue is one value a secret key resolves to.
type secretValue struct {
	reference string // JSON pointer relative to config
	key       string
	value     any
	masked    any // the same value in the config MaskSecrets rewrote
}

// secretValues resolves each secret key against config the way the secret
// package does: a map descends into the named key and a list fans out to every
// member. masked is walked in step with config.
func secretValues(config, masked map[string]any, secretKeys []string) []secretValue {
	var values []secretValue
	for _, secretKey := range secretKeys {
		values = appendSecretValues(values, config, masked, strings.Split(secretKey, "."), "", "")
	}
	return values
}

func appendSecretValues(values []secretValue, config, masked map[string]any, path []string, reference, key string) []secretValue {
	if len(path) == 0 || path[0] == "" {
		return values
	}

	name := path[0]
	value, ok := config[name]
	if !ok {
		return values
	}

	reference += "/" + name
	if key != "" {
		key += "."
	}
	key += name

	if len(path) == 1 {
		return append(values, secretValue{reference: reference, key: key, value: value, masked: masked[name]})
	}
	return appendNestedSecretValues(values, value, masked[name], path[1:], reference, key)
}

func appendNestedSecretValues(values []secretValue, value, masked any, path []string, reference, key string) []secretValue {
	switch v := value.(type) {
	case []any:
		maskedItems, _ := masked.([]any)
		for i, item := range v {
			var maskedItem any
			if i < len(maskedItems) {
				maskedItem = maskedItems[i]
			}
			values = appendNestedSecretValues(values, item, maskedItem, path, fmt.Sprintf("%s/%d", reference, i), fmt.Sprintf("%s[%d]", key, i))
		}
	case map[string]any:
		maskedMap, _ := masked.(map[string]any)
		values = appendSecretValues(values, v, maskedMap, path, reference, key)
	}
	return values
}

// referencedVariable returns the variable a "{{ .NAME }}" reference names.
func referencedVariable(value any) string {
	token, _ := value.(string)
	refs := varsubst.ReplaceReferences([]byte(token)).References
	if len(refs) != 1 {
		return ""
	}
	return refs[0].Name
}

// showsSecret reports whether the line validate prints for a result at
// configPath can show a secret value. The line belongs to the path, or to its
// nearest ancestor in config when the path is missing. So the answer is yes
// when that printed path is a secret key, lies under one, or leads to one. A
// list item that holds a secret key leads to it, and its line shows whichever
// property YAML lists first, which config does not record.
func showsSecret(secretKeys []string, config map[string]any, configPath string) bool {
	keys := printedKeys(config, configPath)
	if len(keys) == 0 {
		return false
	}

	path := strings.Join(keys, ".")
	for _, secretKey := range secretKeys {
		if secretKey == path || strings.HasPrefix(path, secretKey+".") || strings.HasPrefix(secretKey, path+".") {
			return true
		}
	}
	return false
}

// printedKeys walks configPath through config as far as it exists, which is
// the path validate prints the line of. It returns the keys on the way without
// list indices, since secret keys name a list's members without one.
func printedKeys(config map[string]any, configPath string) []string {
	var (
		keys    []string
		current any = config
	)
	for _, segment := range strings.Split(strings.Trim(configPath, "/"), "/") {
		switch v := current.(type) {
		case map[string]any:
			value, ok := v[segment]
			if !ok {
				return keys
			}
			keys = append(keys, segment)
			current = value
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(v) {
				return keys
			}
			current = v[index]
		default:
			return keys
		}
	}
	return keys
}
