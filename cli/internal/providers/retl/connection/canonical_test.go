package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsonMapperConfig is the smallest config the JSON mapper flow accepts; tests
// tweak the one field they are about and leave the rest alone.
func jsonMapperConfig() ConfigSpec {
	return ConfigSpec{
		SyncBehaviour: "upsert",
		Schedule:      ScheduleSpec{Type: "basic", EveryMinutes: ptr(30)},
		Identifiers:   []MappingSpec{{From: "id", To: "user_id"}},
		Mappings:      []MappingSpec{{From: "email", To: "traits.email"}},
	}
}

// objectMappingConfig is the smallest config the object mapping flow accepts:
// exactly one identifier and an object, and no constants or event.
func objectMappingConfig() ConfigSpec {
	return ConfigSpec{
		SyncBehaviour: "upsert",
		Schedule:      ScheduleSpec{Type: "manual"},
		Identifiers:   []MappingSpec{{From: "id", To: "user_id"}},
		Object:        ptr("Contact"),
	}
}

// completeSettings is the object the backend stores for a connection that
// changes one setting: the rest are filled in with their defaults.
func completeSettings(enabled bool, retention, snapshots int, retry bool) *SyncSettingsSpec {
	return &SyncSettingsSpec{
		SyncLogs:   &SyncLogsSpec{Enabled: ptr(enabled), RetentionDays: ptr(retention), SnapshotsToRetain: ptr(snapshots)},
		FailedKeys: &FailedKeysSpec{Retry: ptr(retry)},
	}
}

func TestCanonicalConfig(t *testing.T) {
	t.Parallel()

	emptyLists := jsonMapperConfig()
	emptyLists.Mappings = []MappingSpec{}
	emptyLists.Constants = []ConstantSpec{}

	omittedLists := jsonMapperConfig()
	omittedLists.Mappings = nil

	partialSettings := jsonMapperConfig()
	partialSettings.SyncSettings = &SyncSettingsSpec{SyncLogs: &SyncLogsSpec{RetentionDays: ptr(90)}}

	filledSettings := jsonMapperConfig()
	filledSettings.SyncSettings = completeSettings(true, 90, 5, true)

	explicitZeroes := jsonMapperConfig()
	explicitZeroes.SyncSettings = &SyncSettingsSpec{
		SyncLogs:   &SyncLogsSpec{Enabled: ptr(false), SnapshotsToRetain: ptr(0)},
		FailedKeys: &FailedKeysSpec{Retry: ptr(false)},
	}

	filledZeroes := jsonMapperConfig()
	filledZeroes.SyncSettings = completeSettings(false, 30, 0, false)

	defaultSettings := jsonMapperConfig()
	defaultSettings.SyncSettings = completeSettings(true, 30, 5, true)

	restatedDefault := jsonMapperConfig()
	restatedDefault.SyncSettings = &SyncSettingsSpec{SyncLogs: &SyncLogsSpec{Enabled: ptr(true)}}

	emptySettings := jsonMapperConfig()
	emptySettings.SyncSettings = &SyncSettingsSpec{}

	emptyObject := objectMappingConfig()
	emptyObject.Object = ptr("")

	withEvent := jsonMapperConfig()
	withEvent.Event = &EventSpec{Type: "track", NameColumn: "event_name"}

	tests := []struct {
		name   string
		config ConfigSpec
		want   ConfigSpec
	}{
		{name: "empty mappings and constants collapse to omission", config: emptyLists, want: omittedLists},
		{name: "a partial settings block is filled out field by field", config: partialSettings, want: filledSettings},
		{name: "an explicit false or zero is a setting, not an omission", config: explicitZeroes, want: filledZeroes},
		{name: "the complete default object collapses to omission", config: defaultSettings, want: jsonMapperConfig()},
		{name: "a partial block that only restates defaults collapses too", config: restatedDefault, want: jsonMapperConfig()},
		{name: "an empty settings block is the defaults, so it collapses", config: emptySettings, want: jsonMapperConfig()},
		{name: "an explicitly empty object is left for validation to reject", config: emptyObject, want: emptyObject},
		{name: "an omitted event stays omitted", config: jsonMapperConfig(), want: jsonMapperConfig()},
		{name: "a supplied event is preserved as written", config: withEvent, want: withEvent},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, canonicalConfig(tc.config))
		})
	}
}

// The config a handler hands over is owned by the spec loader, which keeps
// using it after the graph is built; normalizing must not reach into it.
func TestCanonicalConfigLeavesTheCallerCopyAlone(t *testing.T) {
	t.Parallel()

	build := func() ConfigSpec {
		config := jsonMapperConfig()
		config.Constants = []ConstantSpec{}
		config.SyncSettings = &SyncSettingsSpec{SyncLogs: &SyncLogsSpec{RetentionDays: ptr(90)}}
		return config
	}

	config := build()
	canonicalConfig(config)
	assert.Equal(t, build(), config)
}

func TestConfigToData(t *testing.T) {
	t.Parallel()

	config := jsonMapperConfig()
	config.Constants = []ConstantSpec{{Key: "source", Value: "warehouse"}}
	config.Event = &EventSpec{Type: "track", Name: "signup"}
	config.CursorColumn = "updated_at"
	config.SyncSettings = &SyncSettingsSpec{SyncLogs: &SyncLogsSpec{Enabled: ptr(false), SnapshotsToRetain: ptr(0)}}

	data, err := configToData(config)
	require.NoError(t, err)

	// Every number is float64 and every key snake_case, whichever path built
	// the config — that is what keeps a spec and its remote state comparable.
	// An explicit false or zero has to survive omitempty on the way out.
	assert.Equal(t, map[string]any{
		"sync_behaviour": "upsert",
		"schedule":       map[string]any{"type": "basic", "every_minutes": float64(30)},
		"identifiers":    []any{map[string]any{"from": "id", "to": "user_id"}},
		"mappings":       []any{map[string]any{"from": "email", "to": "traits.email"}},
		"constants":      []any{map[string]any{"key": "source", "value": "warehouse"}},
		"event":          map[string]any{"type": "track", "name": "signup"},
		"cursor_column":  "updated_at",
		"sync_settings": map[string]any{
			"sync_logs":   map[string]any{"enabled": false, "retention_days": float64(30), "snapshots_to_retain": float64(0)},
			"failed_keys": map[string]any{"retry": true},
		},
	}, data)
}

func TestConfigToDataOmitsAbsentOptionals(t *testing.T) {
	t.Parallel()

	config := jsonMapperConfig()
	config.Schedule = ScheduleSpec{Type: "manual"}
	config.Constants = []ConstantSpec{}

	data, err := configToData(config)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"sync_behaviour": "upsert",
		"schedule":       map[string]any{"type": "manual"},
		"identifiers":    []any{map[string]any{"from": "id", "to": "user_id"}},
		"mappings":       []any{map[string]any{"from": "email", "to": "traits.email"}},
	}, data)
}

func TestConfigDataRoundTrip(t *testing.T) {
	t.Parallel()

	config := jsonMapperConfig()
	config.Schedule = ScheduleSpec{Type: "cron", CronExpression: "0 * * * *"}
	config.Object = ptr("Contact")
	config.CursorColumn = "updated_at"
	config.SyncSettings = &SyncSettingsSpec{FailedKeys: &FailedKeysSpec{Retry: ptr(false)}}

	data, err := configToData(config)
	require.NoError(t, err)

	decoded, err := configFromData(data)
	require.NoError(t, err)
	assert.Equal(t, canonicalConfig(config), decoded)
}

func TestConfigFromDataErrors(t *testing.T) {
	t.Parallel()

	t.Run("a missing config is not a map", func(t *testing.T) {
		t.Parallel()

		_, err := configFromData(nil)
		assert.ErrorContains(t, err, "expected a map, got <nil>")
	})

	t.Run("a value of the wrong shape is reported, not dropped", func(t *testing.T) {
		t.Parallel()

		_, err := configFromData(map[string]any{"schedule": "every minute"})
		assert.ErrorContains(t, err, "decoding connection config data")
	})

	t.Run("a key the spec does not know is reported, not dropped", func(t *testing.T) {
		t.Parallel()

		_, err := configFromData(map[string]any{"sync_behaviour": "upsert", "destination_config": map[string]any{"audience_id": "aud-1"}})
		assert.ErrorContains(t, err, "decoding connection config data")
	})

	t.Run("a value that cannot be encoded is reported", func(t *testing.T) {
		t.Parallel()

		_, err := configFromData(map[string]any{"cursor_column": make(chan int)})
		assert.ErrorContains(t, err, "encoding connection config data")
	})
}
