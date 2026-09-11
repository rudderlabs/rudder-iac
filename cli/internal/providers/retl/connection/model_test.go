package connection

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

// decode mirrors what the handler does with a spec body: strict decoding, so
// an unknown field is an error rather than a silently dropped setting.
func decode(t *testing.T, raw map[string]any) (ConnectionsSpec, error) {
	t.Helper()

	var spec ConnectionsSpec
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &spec,
	})
	require.NoError(t, err)
	return spec, decoder.Decode(raw)
}

func TestConnectionsSpecDecode(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"connections": []any{
			map[string]any{
				"id":          "users-to-webhook",
				"source":      "#retl-source-sql-model:users",
				"destination": "#destination:webhook",
				"config": map[string]any{
					"sync_behaviour": "upsert",
					"schedule": map[string]any{
						"type":          "basic",
						"every_minutes": 30,
					},
					"identifiers": []any{
						map[string]any{"from": "user_id", "to": "user_id"},
					},
					"mappings": []any{
						map[string]any{"from": "email", "to": "traits.email"},
					},
					"constants": []any{
						map[string]any{"key": "source", "value": "warehouse"},
					},
					"event": map[string]any{
						"type":        "track",
						"name":        "signup",
						"name_column": "event_name",
					},
					"cursor_column": "updated_at",
					"sync_settings": map[string]any{
						// Explicit false and zero must survive decoding: they
						// mean something different from an omitted field.
						"sync_logs": map[string]any{
							"enabled":             false,
							"retention_days":      0,
							"snapshots_to_retain": 0,
						},
						"failed_keys": map[string]any{"retry": false},
					},
				},
			},
			map[string]any{
				"id":          "users-to-amplitude",
				"source":      "#retl-source-sql-model:users",
				"destination": "#destination:amplitude",
				"enabled":     false,
				"config": map[string]any{
					"sync_behaviour": "mirror",
					"schedule": map[string]any{
						"type":            "cron",
						"cron_expression": "*/15 * * * *",
					},
					"identifiers": []any{
						map[string]any{"from": "user_id", "to": "user_id"},
					},
					"object": "users",
				},
			},
		},
	}

	spec, err := decode(t, raw)
	require.NoError(t, err)

	assert.Equal(t, ConnectionsSpec{
		Connections: []ConnectionSpec{
			{
				LocalID:     "users-to-webhook",
				Source:      "#retl-source-sql-model:users",
				Destination: "#destination:webhook",
				Config: ConfigSpec{
					SyncBehaviour: "upsert",
					Schedule:      ScheduleSpec{Type: "basic", EveryMinutes: ptr(30)},
					Identifiers:   []MappingSpec{{From: "user_id", To: "user_id"}},
					Mappings:      []MappingSpec{{From: "email", To: "traits.email"}},
					Constants:     []ConstantSpec{{Key: "source", Value: "warehouse"}},
					Event:         &EventSpec{Type: "track", Name: "signup", NameColumn: "event_name"},
					CursorColumn:  "updated_at",
					SyncSettings: &SyncSettingsSpec{
						SyncLogs: &SyncLogsSpec{
							Enabled:           ptr(false),
							RetentionDays:     ptr(0),
							SnapshotsToRetain: ptr(0),
						},
						FailedKeys: &FailedKeysSpec{Retry: ptr(false)},
					},
				},
			},
			{
				LocalID:     "users-to-amplitude",
				Source:      "#retl-source-sql-model:users",
				Destination: "#destination:amplitude",
				Enabled:     ptr(false),
				Config: ConfigSpec{
					SyncBehaviour: "mirror",
					Schedule:      ScheduleSpec{Type: "cron", CronExpression: "*/15 * * * *"},
					Identifiers:   []MappingSpec{{From: "user_id", To: "user_id"}},
					Object:        ptr("users"),
				},
			},
		},
	}, spec)
}

func TestConnectionsSpecDecodeObject(t *testing.T) {
	t.Parallel()

	entry := func(config map[string]any) map[string]any {
		return map[string]any{
			"connections": []any{
				map[string]any{
					"id":          "users-to-amplitude",
					"source":      "#retl-source-sql-model:users",
					"destination": "#destination:amplitude",
					"config":      config,
				},
			},
		}
	}

	t.Run("absent", func(t *testing.T) {
		t.Parallel()
		spec, err := decode(t, entry(map[string]any{"sync_behaviour": "upsert"}))
		require.NoError(t, err)
		assert.Nil(t, spec.Connections[0].Config.Object)
	})

	t.Run("explicitly empty", func(t *testing.T) {
		t.Parallel()
		spec, err := decode(t, entry(map[string]any{"sync_behaviour": "upsert", "object": ""}))
		require.NoError(t, err)
		assert.Equal(t, ptr(""), spec.Connections[0].Config.Object)
	})
}

// TestSpecValidateTags exercises the spec's validate tags through the rule
// engine: the tags are the contract for what a connection entry must carry,
// and the JSON Pointers they produce are the snake_case paths users are
// pointed at.
func TestSpecValidateTags(t *testing.T) {
	t.Parallel()

	spec := ConnectionsSpec{
		Connections: []ConnectionSpec{{
			Source: "#retl-source-sql-model:users",
			Config: ConfigSpec{
				SyncBehaviour: "append",
				Identifiers:   []MappingSpec{{From: "user_id"}},
			},
		}},
	}

	errs, err := rules.ValidateStruct(spec, "")
	require.NoError(t, err)

	assert.Equal(t, []rules.ValidationResult{
		{Reference: "/connections/0/id", Message: "'id' is required"},
		{Reference: "/connections/0/destination", Message: "'destination' is required"},
		{Reference: "/connections/0/config/sync_behaviour", Message: "'sync_behaviour' must be one of [upsert mirror full]"},
		{Reference: "/connections/0/config/schedule/type", Message: "'type' is required"},
		{Reference: "/connections/0/config/identifiers/0/to", Message: "'to' is required"},
	}, funcs.ParseValidationErrors(errs, nil))
}

// TestSpecValidateTagsIdentifiers pins the nonempty identifiers contract, which
// the required-and-oneof failures above cannot reach: an entry that omits
// identifiers entirely and one that sends an empty list must both be rejected
// at the identifiers path rather than at a member's. They report differently —
// required fires on the nil slice, min=1 on the empty one — so both messages
// are pinned.
func TestSpecValidateTagsIdentifiers(t *testing.T) {
	t.Parallel()

	entry := func(identifiers []MappingSpec) ConnectionsSpec {
		return ConnectionsSpec{
			Connections: []ConnectionSpec{{
				LocalID:     "users-to-webhook",
				Source:      "#retl-source-sql-model:users",
				Destination: "#destination:webhook",
				Config: ConfigSpec{
					SyncBehaviour: "upsert",
					Schedule:      ScheduleSpec{Type: "manual"},
					Identifiers:   identifiers,
				},
			}},
		}
	}

	tests := []struct {
		name        string
		identifiers []MappingSpec
		wantMessage string
	}{
		{name: "omitted", identifiers: nil, wantMessage: "'identifiers' is required"},
		{name: "empty", identifiers: []MappingSpec{}, wantMessage: "'identifiers' length must be greater than or equal to 1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			errs, err := rules.ValidateStruct(entry(test.identifiers), "")
			require.NoError(t, err)

			assert.Equal(t, []rules.ValidationResult{
				{Reference: "/connections/0/config/identifiers", Message: test.wantMessage},
			}, funcs.ParseValidationErrors(errs, nil))
		})
	}
}

// TestSpecValidateTagsOmittedConfig pins that config is required without
// carrying a validate tag of its own: go-playground ignores required on a
// non-pointer struct field, but descends into it, so the zero config is
// rejected by its own required members — naming the fields a user left out
// instead of the block. Making Config a pointer would silently drop all three.
func TestSpecValidateTagsOmittedConfig(t *testing.T) {
	t.Parallel()

	spec := ConnectionsSpec{
		Connections: []ConnectionSpec{{
			LocalID:     "users-to-webhook",
			Source:      "#retl-source-sql-model:users",
			Destination: "#destination:webhook",
		}},
	}

	errs, err := rules.ValidateStruct(spec, "")
	require.NoError(t, err)

	assert.Equal(t, []rules.ValidationResult{
		{Reference: "/connections/0/config/sync_behaviour", Message: "'sync_behaviour' is required"},
		{Reference: "/connections/0/config/schedule/type", Message: "'type' is required"},
		{Reference: "/connections/0/config/identifiers", Message: "'identifiers' is required"},
	}, funcs.ParseValidationErrors(errs, nil))
}

// TestGraphKeysMatchEventStreamConnection guards the keys both connection
// resource types share. The names repeat the event stream ones so the
// project-wide connection topology scan, which reads event stream connections
// only today, can fold rETL connections in without a second set of keys.
func TestGraphKeysMatchEventStreamConnection(t *testing.T) {
	t.Parallel()

	assert.Equal(t, esConnection.ConnectionsKey, ConnectionsKey)
	assert.Equal(t, esConnection.SourceKey, SourceKey)
	assert.Equal(t, esConnection.DestinationKey, DestinationKey)
	assert.Equal(t, esConnection.EnabledKey, EnabledKey)
	assert.Equal(t, esConnection.IDKey, IDKey)
	assert.Equal(t, esConnection.SourceIDKey, SourceIDKey)
	assert.Equal(t, esConnection.DestinationIDKey, DestinationIDKey)
}
