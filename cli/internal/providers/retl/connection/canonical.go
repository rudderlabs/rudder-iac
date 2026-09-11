package connection

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/samber/lo"
)

// The sync settings the backend fills in, field by field, for anything a
// request leaves out — DEFAULT_SYNC_SETTINGS in config-backend
// src/modules/retl/api-gateway/connection-config/constants.ts. A create always
// stores the merged object, so a spec that omits a setting and the response
// that echoes the default describe the same connection.
const (
	defaultSyncLogsEnabled       = true
	defaultSyncLogsRetentionDays = 30
	defaultSnapshotsToRetain     = 5
	defaultFailedKeysRetry       = true
)

// canonicalConfig rewrites a config into the single form the spec and the API
// response both settle on, so an unchanged connection does not re-diff on
// every apply. It never writes through the pointers or slices it is handed:
// the config it normalizes belongs to the caller.
func canonicalConfig(config ConfigSpec) ConfigSpec {
	// The API drops empty mapping and constant lists from its responses, so a
	// spec that spells one out as [] has to lose it too.
	if len(config.Mappings) == 0 {
		config.Mappings = nil
	}
	if len(config.Constants) == 0 {
		config.Constants = nil
	}
	config.SyncSettings = canonicalSyncSettings(config.SyncSettings)
	// Object stays untouched. An explicitly empty object is a spec error
	// DEX-829 reports, not something to normalize out of sight; and events are
	// preserved as supplied, because the backend invents no event default the
	// spec would have to mirror.
	return config
}

// canonicalSyncSettings mirrors the backend merge: every missing field takes
// its default. Omitting the block produces exactly the fully defaulted object,
// so that object collapses back to omission instead of reading as drift.
func canonicalSyncSettings(settings *SyncSettingsSpec) *SyncSettingsSpec {
	if settings == nil {
		return nil
	}

	filled := filledSyncSettings(settings)
	if *filled.SyncLogs.Enabled == defaultSyncLogsEnabled &&
		*filled.SyncLogs.RetentionDays == defaultSyncLogsRetentionDays &&
		*filled.SyncLogs.SnapshotsToRetain == defaultSnapshotsToRetain &&
		*filled.FailedKeys.Retry == defaultFailedKeysRetry {
		return nil
	}
	return filled
}

// filledSyncSettings applies the backend's field-by-field defaulting. Every
// field comes back in a fresh pointer, so the caller's nested objects are
// neither shared nor written through.
func filledSyncSettings(settings *SyncSettingsSpec) *SyncSettingsSpec {
	var (
		logs   SyncLogsSpec
		failed FailedKeysSpec
	)
	if settings != nil {
		if settings.SyncLogs != nil {
			logs = *settings.SyncLogs
		}
		if settings.FailedKeys != nil {
			failed = *settings.FailedKeys
		}
	}

	return &SyncSettingsSpec{
		SyncLogs: &SyncLogsSpec{
			Enabled:           lo.ToPtr(lo.FromPtrOr(logs.Enabled, defaultSyncLogsEnabled)),
			RetentionDays:     lo.ToPtr(lo.FromPtrOr(logs.RetentionDays, defaultSyncLogsRetentionDays)),
			SnapshotsToRetain: lo.ToPtr(lo.FromPtrOr(logs.SnapshotsToRetain, defaultSnapshotsToRetain)),
		},
		FailedKeys: &FailedKeysSpec{Retry: lo.ToPtr(lo.FromPtrOr(failed.Retry, defaultFailedKeysRetry))},
	}
}

// configToData renders a config as the graph's config map. Going through JSON
// gives the local and the remote path identical maps, and canonicalizing here
// is the single choke point for everything that reaches the graph or state.
func configToData(config ConfigSpec) (map[string]any, error) {
	encoded, err := json.Marshal(canonicalConfig(config))
	if err != nil {
		return nil, fmt.Errorf("encoding connection config: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(encoded, &data); err != nil {
		return nil, fmt.Errorf("decoding connection config: %w", err)
	}
	return data, nil
}

// configFromData reads back what configToData wrote. It takes any because the
// value arrives out of a resource data map, where the wrong type is one of the
// malformed inputs it has to report rather than assert on.
func configFromData(data any) (ConfigSpec, error) {
	raw, ok := data.(map[string]any)
	if !ok {
		return ConfigSpec{}, fmt.Errorf("connection config: expected a map, got %T", data)
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return ConfigSpec{}, fmt.Errorf("encoding connection config data: %w", err)
	}

	// A key the spec does not know is state this conversion would drop on the
	// floor, so it is reported instead of silently losing the configuration.
	var config ConfigSpec
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return ConfigSpec{}, fmt.Errorf("decoding connection config data: %w", err)
	}
	return config, nil
}
