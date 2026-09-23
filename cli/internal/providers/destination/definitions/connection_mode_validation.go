package definitions

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// validateConnectionMode checks each connection_mode entry's value against
// the destination's own ConnectionModes for that source type — the same data
// ConnectionModes() reports as metadata, so there is one source of truth for
// both. The valid value set differs per source type per destination, so this
// cannot be expressed as a static struct tag; it is a dynamic pass over the
// raw config, mirroring validateConsentManagement in structure.
//
// Unlike consent_management, it does not itself reject an unsupported source
// type: sourceTypeKeyResults (rule_spec_syntax_valid.go) already does that
// generically for connection_mode, since the key was reserved for it before
// any destination modeled a config field — see sourceTypeConfigKeys. Doing it
// here too would just double-report through the rule pipeline. An unsupported
// source type here simply has no allowed-values list to check against, so its
// value is skipped rather than flagged as invalid.
//
// Dynamic values get no exemption: schema.json declares connectionMode as a
// plain enum per source type, and terraform's validators admit env./template
// forms only on pattern fields, never on enums. A template here would be
// stored verbatim and rejected upstream, so reject it locally instead.
func (d *RegisteredDefinition) validateConnectionMode(config map[string]any) []ConfigError {
	if _, ok := structFieldsByMapstructureTag(d.configType)["connection_mode"]; !ok {
		return nil
	}

	connectionMode, errors := connectionModeBlock(config)
	if connectionMode == nil {
		return errors
	}

	for _, sourceType := range slices.Sorted(maps.Keys(connectionMode)) {
		allowed, err := d.ConnectionModes(sourceType)
		if err != nil {
			continue
		}

		path := joinConfigPath("/connection_mode", sourceType)
		value, ok := connectionMode[sourceType].(string)
		if !ok {
			errors = append(errors, ConfigError{
				Path:    path,
				Message: fmt.Sprintf("'%s' connection mode must be a string", sourceType),
			})
			continue
		}
		if !slices.Contains(allowed, value) {
			errors = append(errors, ConfigError{
				Path: path,
				Message: fmt.Sprintf(
					"'%s' must be one of [%s]",
					sourceType, strings.Join(allowed, " "),
				),
			})
		}
	}
	return errors
}

func connectionModeBlock(config map[string]any) (map[string]any, []ConfigError) {
	raw, exists := config["connection_mode"]
	if !exists {
		return nil, nil
	}
	if raw == nil {
		return nil, []ConfigError{{
			Path:    "/connection_mode",
			Message: "'connection_mode' must be an object",
		}}
	}

	// A wrong-type-but-non-nil value is left to validateConfigModel's
	// mapstructure decode to report, matching consentManagementBlock.
	block, _ := raw.(map[string]any)
	return block, nil
}
