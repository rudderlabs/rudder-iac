package connection

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

// validRawSpec is the same connection as validConnection in the shape the rule
// actually receives: the raw map from the YAML, which is the only form that can
// carry a key the spec type does not have or a value of the wrong type.
func validRawSpec() map[string]any {
	return map[string]any{
		"connections": []any{
			map[string]any{
				"id":          "users-to-webhook",
				"source":      "#retl-source-sql-model:users-model",
				"destination": "#destination:my-http-destination",
				"config": map[string]any{
					"sync_behaviour": "upsert",
					"schedule":       map[string]any{"type": "basic", "every_minutes": 30},
					"identifiers":    []any{map[string]any{"from": "user_id", "to": "user_id"}},
					"mappings":       []any{map[string]any{"from": "email", "to": "traits.email"}},
				},
			},
		},
	}
}

func rawConnection(raw map[string]any) map[string]any {
	return raw["connections"].([]any)[0].(map[string]any)
}

func rawConfig(raw map[string]any) map[string]any {
	return rawConnection(raw)["config"].(map[string]any)
}

// specContext is the context the engine builds for a syntactic rule: kind and
// version decide which validator runs, the raw map is what it reads.
func specContext(raw map[string]any) *rules.ValidationContext {
	return &rules.ValidationContext{
		Kind:    retlConnection.ResourceKind,
		Version: specs.SpecVersionV1,
		Spec:    raw,
	}
}

// validConnection is the entry every spec case starts from: a JSON mapper
// connection on a basic schedule, which passes every check in this rule.
func validConnection() retlConnection.ConnectionSpec {
	return retlConnection.ConnectionSpec{
		LocalID:     "users-to-webhook",
		Source:      "#retl-source-sql-model:users-model",
		Destination: "#destination:my-http-destination",
		Config: retlConnection.ConfigSpec{
			SyncBehaviour: "upsert",
			Schedule:      retlConnection.ScheduleSpec{Type: "basic", EveryMinutes: lo.ToPtr(30)},
			Identifiers:   []retlConnection.MappingSpec{{From: "user_id", To: "user_id"}},
			Mappings:      []retlConnection.MappingSpec{{From: "email", To: "traits.email"}},
		},
	}
}

func TestConnectionSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewConnectionSpecSyntaxValidRule()

	assert.Equal(t, "retl/connection/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "retl connection spec syntax must be valid", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(retlConnection.ResourceKind), rule.AppliesTo())
}

func TestConnectionSpecSyntaxValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(c *retlConnection.ConnectionSpec)
		expected []rules.ValidationResult
	}{
		{
			name:     "valid JSON mapper connection",
			expected: []rules.ValidationResult{},
		},
		{
			name: "valid object mapping connection",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Object = lo.ToPtr("Account")
				c.Config.Mappings = nil
			},
			expected: []rules.ValidationResult{},
		},
		{
			name: "valid manual schedule",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "manual"}
			},
			expected: []rules.ValidationResult{},
		},
		{
			name: "valid cron schedule",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "cron", CronExpression: "0 */2 * * *"}
			},
			expected: []rules.ValidationResult{},
		},
		{
			name: "an unsupported cron dialect is the warning rule's",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "cron", CronExpression: "0 0 L * *"}
			},
			expected: []rules.ValidationResult{},
		},

		// Struct-level shape.
		{
			name:   "missing source",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Source = "" },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/source", Message: "'source' is required"},
			},
		},
		{
			name:   "unsupported sync_behaviour",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Config.SyncBehaviour = "replace" },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/sync_behaviour", Message: "'sync_behaviour' must be one of [upsert mirror full]"},
			},
		},
		{
			name:   "no identifiers",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Config.Identifiers = nil },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/identifiers", Message: "'identifiers' is required"},
			},
		},
		{
			name: "mapping entry missing its target",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Mappings = []retlConnection.MappingSpec{{From: "email"}}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/mappings/0/to", Message: "'to' is required"},
			},
		},

		// V-C2 / V-C6: endpoint references.
		{
			name:   "malformed source reference",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Source = "users-model" },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/source", Message: "'source' is invalid: must be of pattern #retl-source-sql-model:<id>"},
			},
		},
		{
			name:   "source reference of the event stream family",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Source = "#event-stream-source:my-js-source" },
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/source",
					Message:   "'source' must reference a rETL source (#retl-source-sql-model:<id>), got a 'event-stream-source' reference",
				},
			},
		},
		{
			name:   "malformed destination reference",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Destination = "#no-id-here" },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/destination", Message: "'destination' is invalid: must be of pattern #destination:<id>"},
			},
		},
		{
			name:   "destination reference of the wrong kind",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Destination = "#retl-source-sql-model:users-model" },
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/destination",
					Message:   "'destination' must reference a destination (#destination:<id>), got a 'retl-source-sql-model' reference",
				},
			},
		},

		// V-R3: schedule.
		{
			name: "basic schedule without every_minutes",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "basic"}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/schedule/every_minutes", Message: "'every_minutes' is required when 'type' is basic"},
			},
		},
		{
			name: "basic schedule below the five-minute floor",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "basic", EveryMinutes: lo.ToPtr(2)}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/schedule/every_minutes", Message: "'every_minutes' must be greater than or equal to 5"},
			},
		},
		{
			name: "basic schedule carrying a cron expression",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule.CronExpression = "0 * * * *"
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/schedule/cron_expression", Message: "'cron_expression' is not allowed when 'type' is basic"},
			},
		},
		{
			name: "cron schedule without an expression",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "cron"}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/schedule/cron_expression", Message: "'cron_expression' is required when 'type' is cron"},
			},
		},
		{
			name: "cron schedule carrying every_minutes",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "cron", EveryMinutes: lo.ToPtr(30), CronExpression: "0 * * * *"}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/schedule/every_minutes", Message: "'every_minutes' is not allowed when 'type' is cron"},
			},
		},
		{
			name: "cron expression that cannot be parsed",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "cron", CronExpression: "70 * * * *"}
			},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/config/schedule/cron_expression",
					Message:   `'cron_expression' is not valid: minute field "70": 70 is out of range 0-59`,
				},
			},
		},
		{
			name: "cron expression firing more often than every five minutes",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "cron", CronExpression: "*/2 * * * *"}
			},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/config/schedule/cron_expression",
					Message:   "'cron_expression' is not valid: consecutive syncs have a 2-minute gap; the minimum supported interval is 5 minutes",
				},
			},
		},
		{
			name: "manual schedule carrying both optional fields",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Schedule = retlConnection.ScheduleSpec{Type: "manual", EveryMinutes: lo.ToPtr(30), CronExpression: "0 * * * *"}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/schedule/every_minutes", Message: "'every_minutes' is not allowed when 'type' is manual"},
				{Reference: "/connections/0/config/schedule/cron_expression", Message: "'cron_expression' is not allowed when 'type' is manual"},
			},
		},

		// V-R6: column names.
		{
			name: "identifier reading a column that cannot be resolved",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Identifiers = []retlConnection.MappingSpec{{From: "1user id", To: "user_id"}}
			},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/config/identifiers/0/from",
					Message:   `'from' is not valid: must be a column name matching ^[A-Za-z_][\w$.]*$`,
				},
			},
		},
		{
			name: "mapping reading a column that cannot be resolved",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Mappings = []retlConnection.MappingSpec{{From: "e-mail", To: "traits.email"}}
			},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/config/mappings/0/from",
					Message:   `'from' is not valid: must be a column name matching ^[A-Za-z_][\w$.]*$`,
				},
			},
		},
		{
			name:   "cursor column that cannot be resolved",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Config.CursorColumn = "2updated" },
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/config/cursor_column",
					Message:   `'cursor_column' is not valid: must be a column name matching ^[A-Za-z_][\w$.]*$`,
				},
			},
		},
		{
			name: "event name column that cannot be resolved",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Event = &retlConnection.EventSpec{Type: "track", NameColumn: "event name"}
			},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/config/event/name_column",
					Message:   `'name_column' is not valid: must be a column name matching ^[A-Za-z_][\w$.]*$`,
				},
			},
		},

		// V-R7 / V-R8 / V-R9a / V-R10.
		{
			name: "cursor column outside an upsert sync",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.SyncBehaviour = "mirror"
				c.Config.CursorColumn = "updated_at"
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/cursor_column", Message: "'cursor_column' is not allowed when 'sync_behaviour' is mirror"},
			},
		},
		{
			name: "event naming both a literal and a column",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Event = &retlConnection.EventSpec{Type: "track", Name: "Signed Up", NameColumn: "event_name"}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/event/name", Message: "'name' and 'name_column' cannot be specified together"},
			},
		},
		{
			name: "constant claiming the reserved backend key",
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Constants = []retlConnection.ConstantSpec{
					{Key: "source", Value: "warehouse"},
					{Key: "context.mappedToDestination", Value: "true"},
				}
			},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/config/constants/1/key",
					Message:   `'key' is not valid: "context.mappedToDestination" is reserved by the backend`,
				},
			},
		},
		{
			name:   "object declared but empty",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Config.Object = lo.ToPtr("") },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/object", Message: "'object' must not be empty"},
			},
		},
		{
			name:   "object padded with whitespace",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Config.Object = lo.ToPtr(" Account ") },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/object", Message: "'object' must not have leading or trailing whitespace"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := validConnection()
			if tt.mutate != nil {
				tt.mutate(&c)
			}
			spec := retlConnection.ConnectionsSpec{Connections: []retlConnection.ConnectionSpec{c}}

			assert.Equal(t, tt.expected, validateConnectionsSpec(spec))
		})
	}
}

// TestConnectionSpecSyntaxValid_PerEntryReferences proves the per-entry checks
// report against the entry that carries the mistake rather than the first one.
func TestConnectionSpecSyntaxValid_PerEntryReferences(t *testing.T) {
	t.Parallel()

	second := validConnection()
	second.LocalID = "orders-to-webhook"
	second.Config.Schedule = retlConnection.ScheduleSpec{Type: "basic", EveryMinutes: lo.ToPtr(1)}

	spec := retlConnection.ConnectionsSpec{
		Connections: []retlConnection.ConnectionSpec{validConnection(), second},
	}

	assert.Equal(t, []rules.ValidationResult{{
		Reference: "/connections/1/config/schedule/every_minutes",
		Message:   "'every_minutes' must be greater than or equal to 5",
	}}, validateConnectionsSpec(spec))
}

// TestConnectionSpecSyntaxValid_StrictShape covers what only the raw map can
// express: a key the spec type does not carry, and a value whose type it cannot
// hold. Decoding into the spec struct would drop the first and fail the whole
// spec on the second, so the rule decodes the map itself.
func TestConnectionSpecSyntaxValid_StrictShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(raw map[string]any)
		expected []rules.ValidationResult
	}{
		{
			name:     "a spec the type carries in full",
			expected: []rules.ValidationResult{},
		},
		{
			name:   "an unknown field on the connection entry",
			mutate: func(raw map[string]any) { rawConnection(raw)["descriptoin"] = "users to webhook" },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/descriptoin", Message: `unknown field "descriptoin"`},
			},
		},
		{
			name:   "an unknown field under config",
			mutate: func(raw map[string]any) { rawConfig(raw)["cursor_colunm"] = "updated_at" },
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/cursor_colunm", Message: `unknown field "cursor_colunm"`},
			},
		},
		{
			name: "an unknown field on a mapping entry",
			mutate: func(raw map[string]any) {
				rawConfig(raw)["mappings"] = []any{map[string]any{"from": "email", "to": "traits.email", "fom": "email"}}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/mappings/0/fom", Message: `unknown field "fom"`},
			},
		},
		{
			name: "unknown fields at several depths are all reported",
			mutate: func(raw map[string]any) {
				rawConnection(raw)["typo"] = true
				rawConfig(raw)["schedule"] = map[string]any{"type": "manual", "evry_minutes": 30}
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/schedule/evry_minutes", Message: `unknown field "evry_minutes"`},
				{Reference: "/connections/0/typo", Message: `unknown field "typo"`},
			},
		},
		{
			name: "a misspelt field is reported as unknown and as the field it left unset",
			mutate: func(raw map[string]any) {
				config := rawConfig(raw)
				delete(config, "sync_behaviour")
				config["sync_behaivour"] = "upsert"
			},
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/config/sync_behaivour", Message: `unknown field "sync_behaivour"`},
				{Reference: "/connections/0/config/sync_behaviour", Message: "'sync_behaviour' is required"},
			},
		},
		{
			name: "a nested field of the wrong type",
			mutate: func(raw map[string]any) {
				rawConfig(raw)["schedule"] = map[string]any{"type": "basic", "every_minutes": "thirty"}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/schedule/every_minutes",
				Message:   "'every_minutes' is not valid: expected type 'int', got unconvertible type 'string'",
			}},
		},
		{
			name: "a fractional number for an integer field",
			mutate: func(raw map[string]any) {
				rawConfig(raw)["schedule"] = map[string]any{"type": "basic", "every_minutes": 7.5}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/schedule/every_minutes",
				Message:   "'every_minutes' is not valid: expected an integer, got 7.5",
			}},
		},
		{
			// The rule engine's JSON round-trip hands every YAML number over as a float64.
			name: "a whole number decoded as a float",
			mutate: func(raw map[string]any) {
				rawConfig(raw)["schedule"] = map[string]any{"type": "basic", "every_minutes": 30.0}
			},
			expected: []rules.ValidationResult{},
		},
		{
			name:   "a list field given a scalar",
			mutate: func(raw map[string]any) { rawConfig(raw)["mappings"] = "email" },
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/mappings",
				Message:   "'mappings' is not valid: source data must be an array or slice, got string",
			}},
		},
		{
			name:   "a connection entry that is not a map",
			mutate: func(raw map[string]any) { raw["connections"] = []any{"users-to-webhook"} },
			expected: []rules.ValidationResult{{
				Reference: "/connections/0",
				Message:   `'connections' is not valid: expected a map or struct, got "string"`,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := validRawSpec()
			if tt.mutate != nil {
				tt.mutate(raw)
			}

			assert.Equal(t, tt.expected, validateRawConnectionsSpec("", "", nil, raw))
		})
	}
}

// TestConnectionSpecSyntaxValidRule_Validate drives the rule through its public
// entry point, so the constructor's wiring and the engine's "/spec" prefixing
// are covered rather than only the validator behind them.
func TestConnectionSpecSyntaxValidRule_Validate(t *testing.T) {
	t.Parallel()

	raw := validRawSpec()
	rawConfig(raw)["schedule"] = map[string]any{"type": "basic", "every_minutes": 2, "bogus": true}

	assert.Equal(t, []rules.ValidationResult{
		{
			Reference: "/spec/connections/0/config/schedule/bogus",
			Message:   `unknown field "bogus"`,
		},
		{
			Reference: "/spec/connections/0/config/schedule/every_minutes",
			Message:   "'every_minutes' must be greater than or equal to 5",
		},
	}, NewConnectionSpecSyntaxValidRule().Validate(specContext(raw)))
}
