package accounts

import (
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
)

const (
	definitionBigQuery = "SOURCE_BIGQUERY"

	// authMethodWIF is BigQuery's workload identity federation mode, which has
	// no key file and therefore no credentials secret.
	authMethodWIF = "workloadIdentityFederation"
)

// validateBigQueryCredentials checks that the credentials secret is a service
// account key file. A {{ .VAR }} token is substituted into the YAML text before
// it is parsed, so a key file's JSON can arrive as a map (unquoted token), with
// its PEM line breaks folded into spaces (single-quoted token over a multi-line
// value), or as text that is not JSON. The backend accepts any non-empty string
// and the UI reports the damage only when credentials are validated (DEX-1022).
func validateBigQueryCredentials(config map[string]any) error {
	if mode, _ := config["authMethod"].(string); mode == authMethodWIF {
		return nil
	}
	raw, ok := config["credentials"]
	if !ok {
		return nil
	}

	text, ok := raw.(string)
	if !ok {
		return errors.New("credentials must be a string holding the service account key JSON; quote the variable with single quotes: credentials: '{{ .VAR }}'")
	}

	var key struct {
		PrivateKey  string `json:"private_key"`
		ClientEmail string `json:"client_email"`
	}
	if err := json.Unmarshal([]byte(text), &key); err != nil {
		return fmt.Errorf("credentials is not valid JSON (%w); it must be the service account key file content", err)
	}
	if key.ClientEmail == "" {
		return errors.New("credentials has no client_email; it must be a service account key file")
	}
	if block, _ := pem.Decode([]byte(key.PrivateKey)); block == nil {
		return errors.New("credentials private_key is not a PEM block; its line breaks were probably lost, keep the key file's \\n escapes and put the JSON on one line")
	}
	return nil
}
