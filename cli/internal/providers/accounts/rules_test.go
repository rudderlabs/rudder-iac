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
