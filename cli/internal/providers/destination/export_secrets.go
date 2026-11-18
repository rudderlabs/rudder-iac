package destination

import (
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
)

// absentSecretKeys returns the top-level secret keys missing from config as
// the API returned it. The API leaves some secrets out of every response, and
// only those are candidates for a reference: one it returns as null or "" is
// unset, and export drops it (DEX-661).
func absentSecretKeys(registered *definitions.RegisteredDefinition, config map[string]any) map[string]bool {
	absent := map[string]bool{}
	for _, key := range registered.SecretKeys() {
		if strings.Contains(key, ".") {
			continue
		}
		if _, present := config[key]; !present {
			absent[key] = true
		}
	}
	return absent
}

// requiredSecretKeys returns the absent secret keys the exported config needs,
// so import writes a reference for each and the spec validates once the var
// file is filled in. A key is needed when the definition's own validation
// requires it for this config (Google Sheets credentials; S3 access keys only
// while role_based_auth is false), or when connecting a source requires it in
// the mode that source connects in (V-C5). The latter covers the source types
// the config names under connection_mode, and a warehouse source, whose mode
// the config may leave out. A secret the config can do without stays out.
func requiredSecretKeys(registered *definitions.RegisteredDefinition, config map[string]any, absent map[string]bool) []string {
	if len(absent) == 0 {
		return nil
	}

	var required []string
	need := func(key string) {
		if absent[key] && !slices.Contains(required, key) {
			required = append(required, key)
		}
	}

	for _, configErr := range registered.ValidateConfig(config) {
		need(strings.TrimPrefix(configErr.Path, "/"))
	}

	block, _ := config["connection_mode"].(map[string]any)
	sourceTypes := make([]string, 0, len(block)+1)
	for sourceType := range block {
		sourceTypes = append(sourceTypes, sourceType)
	}
	if slices.Contains(registered.SupportedSourceTypes(), common.SourceTypeWarehouse) {
		sourceTypes = append(sourceTypes, common.SourceTypeWarehouse)
	}
	for _, sourceType := range sourceTypes {
		for _, key := range registered.ConnectionRequiredKeys(sourceType, registered.ConnectMode(sourceType, config)) {
			need(key)
		}
	}

	slices.Sort(required)
	return required
}
