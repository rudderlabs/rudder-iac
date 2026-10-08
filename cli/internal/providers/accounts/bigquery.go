package accounts

import (
	"encoding/json"
	"encoding/pem"
	"errors"
	"maps"
)

// authMethodWIF is BigQuery's workload identity federation mode, which has no
// key file and therefore no credentials secret.
const authMethodWIF = "workloadIdentityFederation"

// normalizeBigQueryCredentials checks that the credentials secret is a service
// account key file and returns the config with it as the JSON string the
// backend parses.
//
// A {{ .VAR }} token is substituted into the YAML text before it is parsed, so
// the key file's JSON reaches the account in one of three shapes:
//   - a map, when the token is unquoted, which is how `import workspace` writes
//     it. The map is marshalled back to a string: YAML decoded the \n escapes
//     in private_key and Marshal escapes them again, so the key stays valid.
//   - a string, when the token is single-quoted.
//   - a string with its line breaks lost, when the var file value is
//     double-quoted and YAML turned \n into real newlines.
//
// The backend accepts any non-empty string and the UI reports damage only when
// credentials are validated (DEX-1022).
func normalizeBigQueryCredentials(config map[string]any) (map[string]any, error) {
	if mode, _ := config["authMethod"].(string); mode == authMethodWIF {
		return config, nil
	}
	raw, ok := config["credentials"]
	if !ok {
		return config, nil
	}

	var text string
	switch v := raw.(type) {
	case string:
		text = v
	case map[string]any:
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, errors.New("credentials could not be read as the service account key JSON")
		}
		text = string(encoded)
	default:
		return nil, errors.New("credentials must be the service account key JSON, as a string or a variable that holds it: credentials: '{{ .VAR }}'")
	}

	if err := checkServiceAccountKey(text); err != nil {
		return nil, err
	}

	normalized := maps.Clone(config)
	normalized["credentials"] = text
	return normalized, nil
}

// checkServiceAccountKey applies the rule sqlconnect-go and rudder-go-kit's
// CompatibleServiceAccountJSON apply: only a service_account key file works,
// and the other Google credential files are an easy mistake. Error text is
// fixed, never derived from the value, because the value is a secret.
func checkServiceAccountKey(text string) error {
	var key struct {
		Type        string `json:"type"`
		PrivateKey  string `json:"private_key"`
		ClientEmail string `json:"client_email"`
	}
	if err := json.Unmarshal([]byte(text), &key); err != nil {
		return errors.New("credentials is not valid JSON; it must be the service account key file content")
	}

	switch key.Type {
	case "service_account":
	case "authorized_user":
		return errors.New("credentials is a gcloud user credentials file (type authorized_user); use a service account key file")
	case "external_account":
		return errors.New("credentials is an external_account file; for workload identity federation set authMethod: workloadIdentityFederation and omit credentials")
	default:
		return errors.New("credentials is not a service account key file; its type must be service_account")
	}

	if key.ClientEmail == "" {
		return errors.New("credentials has no client_email; it must be a service account key file")
	}
	if block, _ := pem.Decode([]byte(key.PrivateKey)); block == nil {
		return errors.New("credentials private_key is not a PEM block; its line breaks were probably lost, because a double-quoted value in a var file turns the \\n escapes into real newlines, so single-quote the value there")
	}
	return nil
}
