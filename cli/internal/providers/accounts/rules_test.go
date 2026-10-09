package accounts

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func validatePostgres(config map[string]any, mutate func(spec map[string]any)) []rules.ValidationResult {
	spec := map[string]any{
		"id": "pg", "name": "Postgres", "account_definition_name": "SOURCE_POSTGRES", "config": config,
	}
	if mutate != nil {
		mutate(spec)
	}
	return NewSpecSyntaxValidRule().Validate(&rules.ValidationContext{
		Kind: AccountSpecKind, Version: "rudder/v1", Spec: spec,
	})
}

func references(results []rules.ValidationResult) []string {
	var refs []string
	for _, r := range results {
		refs = append(refs, r.Reference)
	}
	return refs
}

func pgConfig() map[string]any {
	return map[string]any{"host": "h", "dbname": "d", "user": "u", "port": 5432, "sslMode": "require", "password": "p"}
}

func TestSpecSyntaxValid_AcceptsCompleteSpec(t *testing.T) {
	assert.Empty(t, validatePostgres(pgConfig(), nil))
}

// DEX-995: a spec without id or name used to plan an empty "account:" URN.
func TestSpecSyntaxValid_RequiresIDAndName(t *testing.T) {
	assert.Equal(t, []string{"/spec/id"}, references(validatePostgres(pgConfig(), func(s map[string]any) { delete(s, "id") })))
	assert.Equal(t, []string{"/spec/name"}, references(validatePostgres(pgConfig(), func(s map[string]any) { delete(s, "name") })))
	assert.ElementsMatch(t, []string{"/spec/id", "/spec/name"}, references(validatePostgres(pgConfig(), func(s map[string]any) {
		delete(s, "id")
		delete(s, "name")
	})))
}

// DEX-994: a missing required config key used to surface only as a raw backend
// 400 during apply.
func TestSpecSyntaxValid_RequiresDefinitionConfig(t *testing.T) {
	config := pgConfig()
	delete(config, "host")
	delete(config, "password")

	results := validatePostgres(config, nil)

	assert.ElementsMatch(t, []string{"/spec/config/host", "/spec/config/password"}, references(results))
	assert.Contains(t, results[0].Message, "SOURCE_POSTGRES")
}

func TestSpecSyntaxValid_SnowflakeSecretFollowsAuthMode(t *testing.T) {
	validate := func(config map[string]any) []string {
		return references(NewSpecSyntaxValidRule().Validate(&rules.ValidationContext{
			Kind: AccountSpecKind, Version: "rudder/v1",
			Spec: map[string]any{"id": "s", "name": "S", "account_definition_name": "SOURCE_SNOWFLAKE", "config": config},
		}))
	}
	base := func(mode string, extra map[string]any) map[string]any {
		c := map[string]any{"account": "a", "dbname": "d", "warehouse": "w", "user": "u", "authenticationType": mode}
		for k, v := range extra {
			c[k] = v
		}
		return c
	}

	assert.Empty(t, validate(base("keyPair", map[string]any{"privateKey": "k"})))
	assert.Equal(t, []string{"/spec/config/privateKey"}, validate(base("keyPair", map[string]any{"password": "p"})))
	assert.Equal(t, []string{"/spec/config/password"}, validate(base("password", nil)))
	// No mode: only the discriminator is reported, not a guessed secret.
	noMode := base("", nil)
	delete(noMode, "authenticationType")
	assert.Equal(t, []string{"/spec/config/authenticationType"}, validate(noMode))
}

func TestMissingRequiredConfig_UnregisteredDefinition(t *testing.T) {
	assert.Empty(t, missingRequiredConfig("SOURCE_UNKNOWN", map[string]any{}))
}

// The SOURCE_BIGQUERY schema requires credentials for a key file, and the three
// workload identity options for federation, where a key file is not sent.
func TestMissingRequiredConfig_BigQueryKeysFollowAuthMethod(t *testing.T) {
	federation := func(extra map[string]any) map[string]any {
		c := map[string]any{
			"project":                       "p",
			"authMethod":                    "workloadIdentityFederation",
			"workloadIdentityProjectNumber": "123",
			"workloadIdentityPoolId":        "pool",
			"workloadIdentityProviderId":    "provider",
		}
		for k, v := range extra {
			if v == nil {
				delete(c, k)
				continue
			}
			c[k] = v
		}
		return c
	}

	tests := []struct {
		name   string
		config map[string]any
		want   []string
	}{
		{"key file needs credentials", map[string]any{"project": "p"}, []string{"credentials"}},
		{"explicit key file method needs credentials", map[string]any{"project": "p", "authMethod": "serviceAccountKey"}, []string{"credentials"}},
		{"key file with credentials is complete", map[string]any{"project": "p", "credentials": "k"}, nil},
		{"complete federation needs no credentials", federation(nil), nil},
		{"federation needs the project", federation(map[string]any{"project": nil}), []string{"project"}},
		{"federation needs the project number", federation(map[string]any{"workloadIdentityProjectNumber": nil}), []string{"workloadIdentityProjectNumber"}},
		{"federation needs the pool id", federation(map[string]any{"workloadIdentityPoolId": nil}), []string{"workloadIdentityPoolId"}},
		{"federation needs the provider id", federation(map[string]any{"workloadIdentityProviderId": nil}), []string{"workloadIdentityProviderId"}},
		{"federation ignores the key file options of the other mode", federation(map[string]any{"credentials": ""}), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, missingRequiredConfig("SOURCE_BIGQUERY", tt.config))
		})
	}
}

// A null value is how "credentials:" with nothing after it parses, and the API
// treats it as unset.
func TestMissingRequiredConfig_NullCountsAsMissing(t *testing.T) {
	assert.Equal(t, []string{"credentials"}, missingRequiredConfig("SOURCE_BIGQUERY", map[string]any{"project": "p", "credentials": nil}))
	assert.Equal(t, []string{"project"}, missingRequiredConfig("SOURCE_BIGQUERY", map[string]any{"project": nil, "credentials": "k"}))
	// A null discriminator is an absent one, so the default mode applies.
	assert.Equal(t, []string{"credentials"}, missingRequiredConfig("SOURCE_BIGQUERY", map[string]any{"project": "p", "authMethod": nil}))
	assert.Equal(t, []string{"authenticationType"}, missingRequiredConfig("SOURCE_SNOWFLAKE", map[string]any{
		"account": "a", "dbname": "d", "warehouse": "w", "user": "u", "authenticationType": nil,
	}))
}

// A typo in the mode used to require nothing, so the spec passed validate and
// failed at the API (DEX-994).
func TestSpecSyntaxValid_ReportsUnknownAuthMode(t *testing.T) {
	validate := func(definition string, config map[string]any) []rules.ValidationResult {
		return NewSpecSyntaxValidRule().Validate(&rules.ValidationContext{
			Kind: AccountSpecKind, Version: "rudder/v1",
			Spec: map[string]any{"id": "a", "name": "A", "account_definition_name": definition, "config": config},
		})
	}

	bigquery := validate("SOURCE_BIGQUERY", map[string]any{"project": "p", "authMethod": "serviceAccount"})
	assert.Equal(t, []rules.ValidationResult{{
		Reference: "/spec/config/authMethod",
		Message:   "'authMethod' must be one of serviceAccountKey, workloadIdentityFederation for SOURCE_BIGQUERY accounts",
	}}, bigquery)

	snowflake := validate("SOURCE_SNOWFLAKE", map[string]any{
		"account": "a", "dbname": "d", "warehouse": "w", "user": "u", "authenticationType": "oauth",
	})
	assert.Equal(t, []string{"/spec/config/authenticationType"}, references(snowflake))
	assert.Contains(t, snowflake[0].Message, "must be one of keyPair, password")

	// A mode that is not a string is unknown too.
	assert.Equal(t, []string{"/spec/config/authMethod"}, references(validate("SOURCE_BIGQUERY", map[string]any{"project": "p", "authMethod": 1})))
}
