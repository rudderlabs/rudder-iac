package destination

import (
	"fmt"
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
	file   string
	specID string
	// key is the secret key with list indices, such as "headers[0].to".
	key string
	// reference is the JSON pointer to the value in the spec file.
	reference string
	// variable is the variable the value references when the value is exactly
	// one reference with no default, and empty otherwise.
	variable string
	// suggested is the variable import would write for the key. It is empty
	// when the spec id makes no valid variable name.
	suggested string
}

// readSecretSlots returns the secret key values of the destination spec in
// source. A spec it cannot read, such as one with an unregistered type, has
// none, and the syntax rule reports it. A key holding null or an empty string
// is left out, since it holds no secret.
func readSecretSlots(registry *definitions.Registry, file string, source []byte) []secretSlot {
	replaced, refs := varsubst.ReplaceReferences(source)

	spec, ok := readSourceSpec(replaced)
	if !ok {
		return nil
	}
	def, err := registry.Get(spec.destinationType, spec.definitionVersion)
	if err != nil {
		return nil
	}
	secretKeys := def.SecretKeys()

	// MaskSecrets rewrites each secret value as the reference import writes for
	// it, so masking a second copy of the config yields the suggested names
	// without restating the naming rule here. A missing id, which the syntax
	// rule reports, would give names with no prefix, so it gets none.
	masked, _ := readSourceSpec(replaced)
	if spec.id == "" || secret.MaskSecrets(masked.config, spec.id, secretKeys) != nil {
		masked.config = nil
	}

	var slots []secretSlot
	for _, value := range secretValues(spec.config, masked.config, secretKeys) {
		if value.value == nil || value.value == "" {
			continue
		}

		slot := secretSlot{
			file:      file,
			specID:    spec.id,
			key:       value.key,
			reference: "/spec/config" + value.reference,
			suggested: referencedVariable(value.masked),
		}
		if s, ok := value.value.(string); ok {
			if ref, ok := varsubst.ReplacedReference(s, refs); ok && !ref.HasDefault {
				slot.variable = ref.Name
			}
		}
		slots = append(slots, slot)
	}
	return slots
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

	id, _ := parsed.Spec["id"].(string)
	destinationType, _ := parsed.Spec["type"].(string)
	definitionVersion, _ := parsed.Spec["definition_version"].(int)
	config, _ := parsed.Spec["config"].(map[string]any)
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
	_, refs := varsubst.ReplaceReferences([]byte(token))
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
