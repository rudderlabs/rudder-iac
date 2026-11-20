package destination

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
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
	// ref is the reference the value is, when the value is exactly one
	// reference with nothing around it, and nil otherwise.
	ref *varsubst.Reference
	// suggested is the variable import would write for the key. It is empty
	// when the spec id is empty, holds a reference, or makes no valid variable
	// name.
	suggested string
}

// referencesVariable reports whether the value is exactly one reference with
// no default, the only value a secret key may hold.
func (s secretSlot) referencesVariable() bool {
	return s.ref != nil && !s.ref.HasDefault
}

// readSecretSlots returns the secret key values of the destination spec ctx
// holds as written, read from ctx.Spec, the copy NewSourceContext parsed with
// placeholders. A spec it cannot read has none, and the syntax rule reports
// it. That includes a spec whose type is unregistered or a reference, since
// the type decides which keys are secret. A key holding null, an empty string
// or a value read from each event is left out, since it holds no secret.
func readSecretSlots(registry *definitions.Registry, ctx *vrules.ValidationContext) []secretSlot {
	if ctx.Spec == nil {
		return nil
	}
	var (
		file     = ctx.FilePath
		replaced = ctx.References
		spec     = readSourceSpec(ctx.Spec)
	)
	secretKeys, jsonPathKeys := specSecretKeys(registry, spec)
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

	var slots []secretSlot
	secret.WalkSecrets(spec.config, secretKeys, func(segments []string, value any) {
		if value == nil || value == "" || readsEvent(value, slices.Contains(jsonPathKeys, secretKeyPath(segments))) {
			return
		}

		slot := secretSlot{
			file:        file,
			destination: destination,
			key:         secretKeyName(segments),
			reference:   "/spec/config/" + strings.Join(segments, "/"),
			suggested:   suggestedVariable(id, segments),
		}
		// A value that is not a string is no placeholder.
		s, _ := value.(string)
		if ref, ok := replaced.Reference(s); ok {
			slot.ref = &ref
		}
		slots = append(slots, slot)
	})
	return slots
}

// secretKeyName writes the segments that lead to a secret value as messages
// name the key, with list indexes in brackets, such as "headers[0].to".
// Registration requires secret keys in snake_case, so a segment that is a
// number is a list index.
func secretKeyName(segments []string) string {
	var b strings.Builder
	for _, segment := range segments {
		if _, err := strconv.Atoi(segment); err == nil {
			fmt.Fprintf(&b, "[%s]", segment)
			continue
		}
		if b.Len() > 0 {
			b.WriteString(".")
		}
		b.WriteString(segment)
	}
	return b.String()
}

// secretKeyPath writes the segments that lead to a secret value as the
// definition lists the key, without list indexes, such as "headers.from".
func secretKeyPath(segments []string) string {
	keys := make([]string, 0, len(segments))
	for _, segment := range segments {
		if _, err := strconv.Atoi(segment); err != nil {
			keys = append(keys, segment)
		}
	}
	return strings.Join(keys, ".")
}

// suggestedVariable is the variable import writes for the secret value
// segments lead to in the spec with id. A missing id, which the syntax rule
// reports, would give a name with no prefix, and an id that makes no valid
// variable name would make import fail, so neither gets a suggestion.
func suggestedVariable(id string, segments []string) string {
	if id == "" {
		return ""
	}
	name := secret.VariableName(id, segments...)
	if !varsubst.IsValidVariableName(name) {
		return ""
	}
	return name
}

// specSecretKeys returns the secret keys of the definition the spec names,
// and those of them whose value may be a JSONPath read from each event.
// When definition_version names no registered version, because it is missing,
// a reference or wrong, the syntax rule reports that. The keys of every
// registered version of the type are used then, so a literal secret is still
// reported.
func specSecretKeys(registry *definitions.Registry, spec sourceSpec) (secretKeys, jsonPathKeys []string) {
	if def, err := registry.Get(spec.destinationType, spec.definitionVersion); err == nil {
		return def.SecretKeys(), def.JSONPathSecretKeys()
	}

	versions, err := registry.Versions(spec.destinationType)
	if err != nil {
		return nil, nil
	}
	for _, version := range versions {
		def, err := registry.Get(spec.destinationType, version)
		if err != nil {
			continue
		}
		secretKeys = appendNew(secretKeys, def.SecretKeys())
		jsonPathKeys = appendNew(jsonPathKeys, def.JSONPathSecretKeys())
	}
	return secretKeys, jsonPathKeys
}

// appendNew appends the keys that keys does not hold yet.
func appendNew(keys, more []string) []string {
	for _, key := range more {
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys
}

// readsEvent reports whether value is one the destination reads from each
// event at delivery, so it holds no secret: a JSONPath such as $.messageId in
// a key that reads one, or a template such as {{ message.userId || "" }} with
// an empty fallback. A JSONPath in a key that does not read one, such as a
// basic auth password, is a literal and is reported, and so is a template with
// a fallback, since the fallback is a literal.
func readsEvent(value any, readsJSONPath bool) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	if readsJSONPath && (strings.HasPrefix(s, "$.") || strings.HasPrefix(s, "$[")) {
		return true
	}
	// Only a value that is one template reads from the event. The template
	// pattern spans from the first {{ to the last }}, so with two templates
	// it would read the fallback of the last one alone and pass over a
	// literal fallback in an earlier one.
	if strings.Count(s, "{{") != 1 || strings.Count(s, "}}") != 1 {
		return false
	}
	fallback, ok := definitions.TemplateFallback(s)
	if !ok {
		return false
	}
	fallback = strings.TrimSpace(fallback)
	return fallback == "" || fallback == `""` || fallback == "''"
}

// sourceSpec holds the fields of a destination spec that locate its secrets.
type sourceSpec struct {
	id                string
	destinationType   string
	definitionVersion int64
	config            map[string]any
}

// readSourceSpec reads the fields of a spec whose references were replaced by
// placeholders. Any field may hold a placeholder where the spec expects
// another type, so each is read on its own instead of decoding the whole spec.
func readSourceSpec(spec map[string]any) sourceSpec {
	var (
		id, _                = spec["id"].(string)
		destinationType, _   = spec["type"].(string)
		definitionVersion, _ = spec["definition_version"].(int)
		config, _            = spec["config"].(map[string]any)
	)
	return sourceSpec{
		id:                id,
		destinationType:   destinationType,
		definitionVersion: int64(definitionVersion),
		config:            config,
	}
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
