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
// the mode that source connects in (V-C5). Connect-time needs follow actual
// importable event-stream and warehouse connections, not stored mode keys
// (which can outlive connections). A secret the config can do without stays out.
func requiredSecretKeys(registered *definitions.RegisteredDefinition, config, connectConfig map[string]any, absent map[string]bool, warehouseConnected bool, eventStreamSourceTypes map[string]bool) []string {
	if len(absent) == 0 {
		return nil
	}

	var required []string
	need := func(key string) {
		if absent[key] && !slices.Contains(required, key) {
			required = append(required, key)
		}
	}

	// Local config validation applies even when no connection is exported.
	// A stored mode may itself require a secret (e.g. Braze rest_api_key),
	// independent of the connect-time requirements of importable links.
	for _, configErr := range registered.ValidateConfig(config) {
		need(strings.TrimPrefix(configErr.Path, "/"))
	}

	// An explicitly invalid or unrecognized mode must not fall through
	// to cloud. Config validation reports it; no valid mode's secrets
	// can be inferred from that entry.
	needConnectionSecrets := func(sourceType string) {
		for _, key := range registered.ConnectionRequiredKeys(sourceType, registered.ConnectMode(sourceType, connectConfig)) {
			need(key)
		}
	}
	for sourceType := range eventStreamSourceTypes {
		needConnectionSecrets(sourceType)
	}
	if warehouseConnected && slices.Contains(registered.SupportedSourceTypes(), common.SourceTypeWarehouse) {
		needConnectionSecrets(common.SourceTypeWarehouse)
	}

	slices.Sort(required)
	return required
}
