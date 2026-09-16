package connection

import (
	"encoding/json"
	"slices"
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// graphData builds the resource entry the syncer hands the lifecycle: endpoint
// refs already dereferenced to remote ids, plus the canonical config map.
func graphData(t *testing.T, config ConfigSpec) resources.ResourceData {
	t.Helper()

	data, err := configToMap(config)
	require.NoError(t, err)
	return resources.ResourceData{
		SourceKey:      "src-1",
		DestinationKey: "dst-1",
		EnabledKey:     true,
		ConfigKey:      data,
	}
}

// backendResponse replays what config-backend does with a create body: the
// assembler merges the identifiers into the stored mappings and fills sync
// settings field by field, and the response mapper splits them back out.
// Mirrors connection-config/assembler.ts and response-mapper.ts.
func backendResponse(request *retlClient.CreateRETLConnectionRequest) *retlClient.RETLConnection {
	conn := &retlClient.RETLConnection{
		ID:            "conn-1",
		SourceID:      request.SourceID,
		DestinationID: request.DestinationID,
		Enabled:       lo.FromPtr(request.Enabled),
		Schedule:      request.Schedule,
		SyncBehaviour: lo.FromPtr(request.SyncBehaviour),
		CursorColumn:  request.CursorColumn,
		// A warehouse source always stores the merged settings, whether or not
		// the request carried any.
		SyncSettings: mergedSyncSettings(request.SyncSettings),
	}

	if request.Object == "" {
		for _, mapping := range slices.Concat(request.Identifiers, request.Mappings) {
			if mapping.To == userIDTarget || mapping.To == anonymousIDTarget {
				conn.Identifiers = append(conn.Identifiers, mapping)
				continue
			}
			conn.Mappings = append(conn.Mappings, mapping)
		}
		if len(request.Constants) > 0 {
			conn.Constants = request.Constants
		}
		conn.Event = request.Event
		return conn
	}

	// Object mapping stores the single identifier as two synthetic mappings and
	// consumes both again on read; the object and the identifier target come
	// back out of the system constants. Constants and event never surface.
	identifier := request.Identifiers[0]
	stored := append([]retlClient.Mapping{
		{From: identifier.From, To: userIDTarget},
		{From: identifier.From, To: externalIDTarget},
	}, request.Mappings...)

	for _, mapping := range stored {
		if mapping.To == userIDTarget || mapping.To == externalIDTarget {
			continue
		}
		conn.Mappings = append(conn.Mappings, mapping)
	}
	conn.Identifiers = []retlClient.Mapping{{From: identifier.From, To: identifier.To}}
	conn.Object = request.Object
	return conn
}

func mergedSyncSettings(settings *retlClient.SyncSettings) *retlClient.SyncSettings {
	merged := &retlClient.SyncSettings{
		SyncLogsConfig: &retlClient.SyncLogsConfig{
			Enabled:            lo.ToPtr(true),
			LogRetentionInDays: lo.ToPtr(30),
			SnapshotsToRetain:  lo.ToPtr(5),
		},
		FailedKeysConfig: &retlClient.FailedKeysConfig{EnableFailedKeysRetry: lo.ToPtr(true)},
	}
	if settings == nil {
		return merged
	}
	if logs := settings.SyncLogsConfig; logs != nil {
		if logs.Enabled != nil {
			merged.SyncLogsConfig.Enabled = logs.Enabled
		}
		if logs.LogRetentionInDays != nil {
			merged.SyncLogsConfig.LogRetentionInDays = logs.LogRetentionInDays
		}
		if logs.SnapshotsToRetain != nil {
			merged.SyncLogsConfig.SnapshotsToRetain = logs.SnapshotsToRetain
		}
	}
	if failed := settings.FailedKeysConfig; failed != nil && failed.EnableFailedKeysRetry != nil {
		merged.FailedKeysConfig.EnableFailedKeysRetry = failed.EnableFailedKeysRetry
	}
	return merged
}

func TestEndpointID(t *testing.T) {
	t.Parallel()

	id, err := endpointIDFromData(resources.ResourceData{SourceKey: "src-1"}, SourceKey)
	require.NoError(t, err)
	assert.Equal(t, "src-1", id)

	// An unresolved reference is still a PropertyRef at this point; building a
	// request from it would silently point the connection at nothing.
	_, err = endpointIDFromData(resources.ResourceData{SourceKey: &resources.PropertyRef{}}, SourceKey)
	assert.ErrorContains(t, err, `"source" is not a resolved endpoint id`)

	_, err = endpointIDFromData(resources.ResourceData{SourceKey: ""}, SourceKey)
	assert.ErrorContains(t, err, `"source" is not a resolved endpoint id`)
}

func TestToCreateRequestErrors(t *testing.T) {
	t.Parallel()

	config, err := configToMap(jsonMapperConfig())
	require.NoError(t, err)

	tests := []struct {
		name    string
		data    resources.ResourceData
		wantErr string
	}{
		{
			name:    "an unresolved destination",
			data:    resources.ResourceData{SourceKey: "src-1", EnabledKey: true, ConfigKey: config},
			wantErr: `"destination" is not a resolved endpoint id`,
		},
		{
			name:    "a missing enabled flag",
			data:    resources.ResourceData{SourceKey: "src-1", DestinationKey: "dst-1", ConfigKey: config},
			wantErr: `"enabled" is not a bool`,
		},
		{
			name:    "a config that is not a map",
			data:    resources.ResourceData{SourceKey: "src-1", DestinationKey: "dst-1", EnabledKey: true, ConfigKey: "upsert"},
			wantErr: "expected a map, got string",
		},
		{
			name: "a config whose shape does not decode",
			data: resources.ResourceData{
				SourceKey: "src-1", DestinationKey: "dst-1", EnabledKey: true,
				ConfigKey: map[string]any{"identifiers": "id"},
			},
			wantErr: "decoding connection config data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := toCreateRequest(tc.data)
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestToCreateRequest(t *testing.T) {
	t.Parallel()

	config := jsonMapperConfig()
	config.Constants = []ConstantSpec{{Key: "source", Value: "warehouse"}}
	config.Event = &EventSpec{Type: "track", Name: "signup"}
	config.CursorColumn = "updated_at"
	config.SyncSettings = &SyncSettingsSpec{SyncLogs: &SyncLogsSpec{Enabled: ptr(false)}}

	request, err := toCreateRequest(graphData(t, config))
	require.NoError(t, err)

	assert.Equal(t, &retlClient.CreateRETLConnectionRequest{
		SourceID:      "src-1",
		DestinationID: "dst-1",
		Enabled:       lo.ToPtr(true),
		Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(30)},
		SyncBehaviour: lo.ToPtr(retlClient.SyncBehaviourUpsert),
		Identifiers:   []retlClient.Mapping{{From: "id", To: "user_id"}},
		Mappings:      []retlClient.Mapping{{From: "email", To: "traits.email"}},
		Constants:     []retlClient.Constant{{Key: "source", Value: "warehouse"}},
		Event:         &retlClient.Event{Type: retlClient.EventTypeTrack, Name: "signup"},
		CursorColumn:  "updated_at",
		SyncSettings: &retlClient.SyncSettings{
			SyncLogsConfig:   &retlClient.SyncLogsConfig{Enabled: lo.ToPtr(false), LogRetentionInDays: lo.ToPtr(30), SnapshotsToRetain: lo.ToPtr(5)},
			FailedKeysConfig: &retlClient.FailedKeysConfig{EnableFailedKeysRetry: lo.ToPtr(true)},
		},
	}, request)

	// The wire body is flat: no nested config field.
	body, err := json.Marshal(request)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"sourceId": "src-1",
		"destinationId": "dst-1",
		"enabled": true,
		"schedule": {"type": "basic", "everyMinutes": 30},
		"syncBehaviour": "upsert",
		"identifiers": [{"from": "id", "to": "user_id"}],
		"mappings": [{"from": "email", "to": "traits.email"}],
		"constants": [{"key": "source", "value": "warehouse"}],
		"event": {"type": "track", "name": "signup"},
		"cursorColumn": "updated_at",
		"syncSettings": {
			"syncLogsConfig": {"enabled": false, "logRetentionInDays": 30, "snapshotsToRetain": 5},
			"failedKeysConfig": {"enableFailedKeysRetry": true}
		}
	}`, string(body))
}

func TestToCreateRequestOmitsAbsentOptionals(t *testing.T) {
	t.Parallel()

	// Nothing optional is set: no event, object, cursor column or constants,
	// and sync settings that match what the server fills in by itself.
	config := jsonMapperConfig()
	config.SyncSettings = completeSettings(true, 30, 5, true)

	data := graphData(t, config)
	data[EnabledKey] = false

	request, err := toCreateRequest(data)
	require.NoError(t, err)

	body, err := json.Marshal(request)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"sourceId": "src-1",
		"destinationId": "dst-1",
		"enabled": false,
		"schedule": {"type": "basic", "everyMinutes": 30},
		"syncBehaviour": "upsert",
		"identifiers": [{"from": "id", "to": "user_id"}],
		"mappings": [{"from": "email", "to": "traits.email"}]
	}`, string(body))
}

func TestToCreateRequestObjectMapping(t *testing.T) {
	t.Parallel()

	config := objectMappingConfig()
	config.Mappings = []MappingSpec{}

	request, err := toCreateRequest(graphData(t, config))
	require.NoError(t, err)

	body, err := json.Marshal(request)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"sourceId": "src-1",
		"destinationId": "dst-1",
		"enabled": true,
		"schedule": {"type": "manual"},
		"syncBehaviour": "upsert",
		"identifiers": [{"from": "id", "to": "user_id"}],
		"object": "Contact"
	}`, string(body))
}

// Constants and events belong to the JSON mapper flow. Supplying either beside
// an object is invalid input, and dropping it to satisfy the API would lose it
// silently — so both paths report it instead.
func TestObjectMappingRejectsJSONMapperFields(t *testing.T) {
	t.Parallel()

	withConstants := objectMappingConfig()
	withConstants.Constants = []ConstantSpec{{Key: "source", Value: "warehouse"}}

	withEvent := objectMappingConfig()
	withEvent.Event = &EventSpec{Type: "track", Name: "signed_up"}

	tests := []struct {
		name    string
		config  ConfigSpec
		wantErr string
	}{
		{name: "constants", config: withConstants, wantErr: "constants are not supported with object mapping"},
		{name: "an event", config: withEvent, wantErr: "event is not supported with object mapping"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := graphData(t, tc.config)

			_, err := toCreateRequest(data)
			assert.ErrorContains(t, err, tc.wantErr)

			_, err = toUpdateRequest(data, graphData(t, objectMappingConfig()))
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestToUpdateRequest(t *testing.T) {
	t.Parallel()

	config := jsonMapperConfig()
	config.Constants = []ConstantSpec{{Key: "source", Value: "warehouse"}}
	config.SyncSettings = &SyncSettingsSpec{SyncLogs: &SyncLogsSpec{RetentionDays: ptr(90)}}

	request, err := toUpdateRequest(graphData(t, config), graphData(t, jsonMapperConfig()))
	require.NoError(t, err)

	assert.Equal(t, &retlClient.UpdateRETLConnectionRequest{
		Enabled:     lo.ToPtr(true),
		Schedule:    retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(30)},
		Identifiers: []retlClient.Mapping{{From: "id", To: "user_id"}},
		Mappings:    lo.ToPtr([]retlClient.Mapping{{From: "email", To: "traits.email"}}),
		Constants:   lo.ToPtr([]retlClient.Constant{{Key: "source", Value: "warehouse"}}),
		SyncSettings: &retlClient.SyncSettings{
			SyncLogsConfig:   &retlClient.SyncLogsConfig{Enabled: lo.ToPtr(true), LogRetentionInDays: lo.ToPtr(90), SnapshotsToRetain: lo.ToPtr(5)},
			FailedKeysConfig: &retlClient.FailedKeysConfig{EnableFailedKeysRetry: lo.ToPtr(true)},
		},
	}, request)

	// Immutable fields must be absent from the body, not merely unchanged: the
	// API rejects a PUT that names one at all.
	body, err := json.Marshal(request)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"enabled": true,
		"schedule": {"type": "basic", "everyMinutes": 30},
		"identifiers": [{"from": "id", "to": "user_id"}],
		"mappings": [{"from": "email", "to": "traits.email"}],
		"constants": [{"key": "source", "value": "warehouse"}],
		"syncSettings": {
			"syncLogsConfig": {"enabled": true, "logRetentionInDays": 90, "snapshotsToRetain": 5},
			"failedKeysConfig": {"enableFailedKeysRetry": true}
		}
	}`, string(body))
}

// Omitting a field on a PUT preserves what the server stored, so clearing a
// list has to travel as an explicit empty array.
func TestToUpdateRequestClearsListsExplicitly(t *testing.T) {
	t.Parallel()

	stored := jsonMapperConfig()
	stored.Constants = []ConstantSpec{{Key: "source", Value: "warehouse"}}

	config := jsonMapperConfig()
	config.Constants = nil

	request, err := toUpdateRequest(graphData(t, config), graphData(t, stored))
	require.NoError(t, err)

	body, err := json.Marshal(request)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"enabled": true,
		"schedule": {"type": "basic", "everyMinutes": 30},
		"identifiers": [{"from": "id", "to": "user_id"}],
		"mappings": [{"from": "email", "to": "traits.email"}],
		"constants": []
	}`, string(body))
}

// Object mapping is the flow whose mappings may legitimately go empty, and the
// one whose constants are immutable — an empty constants field included.
func TestToUpdateRequestObjectMapping(t *testing.T) {
	t.Parallel()

	stored := objectMappingConfig()
	stored.Mappings = []MappingSpec{{From: "email", To: "Email"}}

	request, err := toUpdateRequest(graphData(t, objectMappingConfig()), graphData(t, stored))
	require.NoError(t, err)

	body, err := json.Marshal(request)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"enabled": true,
		"schedule": {"type": "manual"},
		"identifiers": [{"from": "id", "to": "user_id"}],
		"mappings": []
	}`, string(body))
}

func TestToUpdateRequestSyncSettings(t *testing.T) {
	t.Parallel()

	nondefault := jsonMapperConfig()
	nondefault.SyncSettings = &SyncSettingsSpec{FailedKeys: &FailedKeysSpec{Retry: ptr(false)}}

	// Only the settings differ across the rows; the rest of the body has to
	// stay put, constants included — those travel as [] on every row.
	wantRequest := func(settings *retlClient.SyncSettings) *retlClient.UpdateRETLConnectionRequest {
		return &retlClient.UpdateRETLConnectionRequest{
			Enabled:      lo.ToPtr(true),
			Schedule:     retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(30)},
			Identifiers:  []retlClient.Mapping{{From: "id", To: "user_id"}},
			Mappings:     lo.ToPtr([]retlClient.Mapping{{From: "email", To: "traits.email"}}),
			Constants:    lo.ToPtr([]retlClient.Constant{}),
			SyncSettings: settings,
		}
	}

	tests := []struct {
		name   string
		config ConfigSpec
		stored ConfigSpec
		want   *retlClient.UpdateRETLConnectionRequest
	}{
		{
			name:   "settings the server already defaults are not restated",
			config: jsonMapperConfig(),
			stored: jsonMapperConfig(),
			want:   wantRequest(nil),
		},
		{
			name:   "nondefault settings travel as the whole normalized object",
			config: nondefault,
			stored: jsonMapperConfig(),
			want: wantRequest(&retlClient.SyncSettings{
				SyncLogsConfig:   &retlClient.SyncLogsConfig{Enabled: lo.ToPtr(true), LogRetentionInDays: lo.ToPtr(30), SnapshotsToRetain: lo.ToPtr(5)},
				FailedKeysConfig: &retlClient.FailedKeysConfig{EnableFailedKeysRetry: lo.ToPtr(false)},
			}),
		},
		{
			name:   "removing nondefault settings resets them to the server defaults",
			config: jsonMapperConfig(),
			stored: nondefault,
			want: wantRequest(&retlClient.SyncSettings{
				SyncLogsConfig:   &retlClient.SyncLogsConfig{Enabled: lo.ToPtr(true), LogRetentionInDays: lo.ToPtr(30), SnapshotsToRetain: lo.ToPtr(5)},
				FailedKeysConfig: &retlClient.FailedKeysConfig{EnableFailedKeysRetry: lo.ToPtr(true)},
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request, err := toUpdateRequest(graphData(t, tc.config), graphData(t, tc.stored))
			require.NoError(t, err)
			assert.Equal(t, tc.want, request)
		})
	}
}

func TestToUpdateRequestErrors(t *testing.T) {
	t.Parallel()

	data := graphData(t, jsonMapperConfig())

	_, err := toUpdateRequest(data, resources.ResourceData{})
	assert.ErrorContains(t, err, "reading stored connection config")

	delete(data, EnabledKey)
	_, err = toUpdateRequest(data, graphData(t, jsonMapperConfig()))
	assert.ErrorContains(t, err, `"enabled" is not a bool`)
}

func TestToOutput(t *testing.T) {
	t.Parallel()

	output := toResourceData(&retlClient.RETLConnection{
		ID:            "conn-1",
		SourceID:      "src-1",
		DestinationID: "dst-1",
		Enabled:       true,
		Object:        "Contact",
	})

	// Only the remote identifiers: config and enabled stay on the state input.
	assert.Equal(t, &resources.ResourceData{
		IDKey:            "conn-1",
		SourceIDKey:      "src-1",
		DestinationIDKey: "dst-1",
	}, output)
}

func TestConfigFromRemoteRejectsDestinationConfig(t *testing.T) {
	t.Parallel()

	// Anything that cannot be proven empty is refused, so a shape the spec has
	// no field for is surfaced instead of being read as an absent config.
	for _, raw := range []json.RawMessage{
		json.RawMessage(`{"audienceId": "aud-1"}`),
		json.RawMessage(`"opaque"`),
		json.RawMessage(`[1]`),
		json.RawMessage(`{"broken"`),
	} {
		_, err := configFromRemote(&retlClient.RETLConnection{ID: "conn-1", DestinationConfig: raw})
		assert.ErrorContainsf(t, err, `connection "conn-1": destination-specific configuration has no spec equivalent`, "%s must be refused", raw)
		assert.ErrorIsf(t, err, ErrUnrepresentableConfig, "%s must be skippable", raw)
	}

	// An absent one arrives as an empty, null or empty-object payload; none of
	// those carries anything to lose.
	for _, raw := range []json.RawMessage{nil, json.RawMessage("null"), json.RawMessage(" {} ")} {
		conn := representableConnection()
		conn.DestinationConfig = raw
		_, err := configFromRemote(conn)
		assert.NoErrorf(t, err, "%q must not read as destination config", raw)
	}
}

// representableConnection is the smallest response the spec contract accepts:
// a JSON mapper connection with one identifier.
func representableConnection() *retlClient.RETLConnection {
	return &retlClient.RETLConnection{
		ID:            "conn-1",
		SyncBehaviour: retlClient.SyncBehaviourUpsert,
		Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(30)},
		Identifiers:   []retlClient.Mapping{{From: "id", To: userIDTarget}},
		Mappings:      []retlClient.Mapping{{From: "email", To: "traits.email"}},
		SyncSettings:  mergedSyncSettings(nil),
	}
}

// A response the supported spec contract cannot express is surfaced for an
// explicit skip rather than exported as a spec that would fail validation or
// never stop diffing. Field-level rules stay with DEX-829.
func TestConfigFromRemoteRefusesUnrepresentableShapes(t *testing.T) {
	t.Parallel()

	noIdentifiers := representableConnection()
	noIdentifiers.Identifiers = nil

	objectWithConstants := representableConnection()
	objectWithConstants.Object = "Contact"
	objectWithConstants.Constants = []retlClient.Constant{{Key: "source", Value: "warehouse"}}

	objectWithEvent := representableConnection()
	objectWithEvent.Object = "Contact"
	objectWithEvent.Event = &retlClient.Event{Type: "track", Name: "signed_up"}

	tests := []struct {
		name    string
		conn    *retlClient.RETLConnection
		wantErr string
	}{
		{name: "no identifiers", conn: noIdentifiers, wantErr: "no identifiers"},
		{name: "object mapping carrying constants", conn: objectWithConstants, wantErr: "constants are not supported with object mapping"},
		{name: "object mapping carrying an event", conn: objectWithEvent, wantErr: "event is not supported with object mapping"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := configFromRemote(tc.conn)
			assert.ErrorContains(t, err, tc.wantErr)
			assert.ErrorIs(t, err, ErrUnrepresentableConfig)
		})
	}
}

func TestConfigFromRemoteOmitsAnEmptyObject(t *testing.T) {
	t.Parallel()

	config, err := configFromRemote(&retlClient.RETLConnection{
		ID:            "conn-1",
		SyncBehaviour: retlClient.SyncBehaviourUpsert,
		Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(30)},
		Identifiers:   []retlClient.Mapping{{From: "id", To: "user_id"}},
		Mappings:      []retlClient.Mapping{{From: "email", To: "traits.email"}},
		SyncSettings:  mergedSyncSettings(nil),
	})
	require.NoError(t, err)

	// A JSON mapper response carries object as an empty string; the spec has to
	// omit it entirely, or every apply would see drift against a nil pointer.
	assert.Equal(t, jsonMapperConfig(), config)
}

// The single invariant the whole conversion exists for: an accepted spec must
// compare equal to the API representation it produces on the next apply.
func TestRoundTrip(t *testing.T) {
	t.Parallel()

	withEvent := jsonMapperConfig()
	withEvent.Event = &EventSpec{Type: "track", NameColumn: "event_name"}

	withConstants := jsonMapperConfig()
	withConstants.Constants = []ConstantSpec{{Key: "source", Value: "warehouse"}}

	anonymous := jsonMapperConfig()
	anonymous.Identifiers = []MappingSpec{{From: "id", To: "user_id"}, {From: "device", To: "anonymous_id"}}

	cron := jsonMapperConfig()
	cron.Schedule = ScheduleSpec{Type: "cron", CronExpression: "0 * * * *"}
	cron.CursorColumn = "updated_at"

	defaultSettings := jsonMapperConfig()
	defaultSettings.SyncSettings = completeSettings(true, 30, 5, true)

	nondefaultSettings := jsonMapperConfig()
	nondefaultSettings.SyncSettings = &SyncSettingsSpec{SyncLogs: &SyncLogsSpec{Enabled: ptr(false), SnapshotsToRetain: ptr(0)}}

	objectWithMappings := objectMappingConfig()
	objectWithMappings.Mappings = []MappingSpec{{From: "email", To: "Email"}}

	emptyObjectMappings := objectMappingConfig()
	emptyObjectMappings.Mappings = []MappingSpec{}

	tests := []struct {
		name   string
		config ConfigSpec
	}{
		{name: "json mapping without an event", config: jsonMapperConfig()},
		{name: "json mapping with an event", config: withEvent},
		{name: "json mapping with constants", config: withConstants},
		{name: "json mapping with an anonymous id identifier", config: anonymous},
		{name: "a cron schedule and a cursor column", config: cron},
		{name: "sync settings the server would default anyway", config: defaultSettings},
		{name: "nondefault sync settings", config: nondefaultSettings},
		{name: "object mapping with mappings", config: objectWithMappings},
		{name: "object mapping with empty mappings", config: emptyObjectMappings},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request, err := toCreateRequest(graphData(t, tc.config))
			require.NoError(t, err)

			remote, err := configFromRemote(backendResponse(request))
			require.NoError(t, err)
			assert.Equal(t, normalizeConfig(tc.config), remote)
		})
	}
}

// The backend combines identifiers and mappings, so a user mapping aimed at a
// reserved target comes back somewhere else. These configs are unsupported —
// DEX-829 rejects them at validation — and the conversion must let the
// inequality show rather than move entries around to hide it.
func TestRoundTripSurfacesReservedMappingTargets(t *testing.T) {
	t.Parallel()

	t.Run("a json mapping aimed at anonymous_id reappears as an identifier", func(t *testing.T) {
		t.Parallel()

		config := jsonMapperConfig()
		config.Mappings = []MappingSpec{{From: "device", To: anonymousIDTarget}, {From: "email", To: "traits.email"}}

		request, err := toCreateRequest(graphData(t, config))
		require.NoError(t, err)

		// The request carries the user's entries untouched: the reclassification
		// below is the backend's doing, not ours.
		assert.Equal(t, []retlClient.Mapping{{From: "id", To: userIDTarget}}, request.Identifiers)
		assert.Equal(t, []retlClient.Mapping{{From: "device", To: anonymousIDTarget}, {From: "email", To: "traits.email"}}, request.Mappings)

		remote, err := configFromRemote(backendResponse(request))
		require.NoError(t, err)

		reclassified := jsonMapperConfig()
		reclassified.Identifiers = []MappingSpec{{From: "id", To: userIDTarget}, {From: "device", To: anonymousIDTarget}}
		assert.Equal(t, reclassified, remote)
		assert.NotEqual(t, normalizeConfig(config), remote)
	})

	t.Run("an object mapping aimed at the external id is dropped", func(t *testing.T) {
		t.Parallel()

		config := objectMappingConfig()
		config.Mappings = []MappingSpec{{From: "id", To: externalIDTarget}, {From: "email", To: "Email"}}

		request, err := toCreateRequest(graphData(t, config))
		require.NoError(t, err)

		// The entry aimed at the reserved target is still in the request; only
		// the backend consumes it as a synthetic identifier.
		assert.Equal(t, []retlClient.Mapping{{From: "id", To: userIDTarget}}, request.Identifiers)
		assert.Equal(t, []retlClient.Mapping{{From: "id", To: externalIDTarget}, {From: "email", To: "Email"}}, request.Mappings)

		remote, err := configFromRemote(backendResponse(request))
		require.NoError(t, err)

		dropped := objectMappingConfig()
		dropped.Mappings = []MappingSpec{{From: "email", To: "Email"}}
		assert.Equal(t, dropped, remote)
		assert.NotEqual(t, normalizeConfig(config), remote)
	})
}

// A change to an immutable field cannot ride on a PUT: the body has no field
// for it, so the server would apply nothing and the diff would return on every
// apply. Only an endpoint change is replaced; the rest are reported here, with
// the remedy the UI imposes too.
func TestToUpdateRequestRejectsImmutableChanges(t *testing.T) {
	t.Parallel()

	behaviour := jsonMapperConfig()
	behaviour.SyncBehaviour = "mirror"

	cursor := jsonMapperConfig()
	cursor.CursorColumn = "updated_at"

	object := jsonMapperConfig()
	object.Object = ptr("Contact")

	event := jsonMapperConfig()
	event.Event = &EventSpec{Type: "track", Name: "signed_up"}

	tests := []struct {
		name    string
		config  ConfigSpec
		wantErr string
	}{
		{name: "sync behaviour", config: behaviour, wantErr: "sync_behaviour is immutable"},
		{name: "cursor column", config: cursor, wantErr: "cursor_column is immutable"},
		{name: "object", config: object, wantErr: "object is immutable"},
		{name: "event", config: event, wantErr: "event is immutable"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := toUpdateRequest(graphData(t, tc.config), graphData(t, jsonMapperConfig()))
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// backendUpdate replays mergeUpdate: fields the PUT omits keep their stored
// value, present ones replace it. The merged result is stored in the same shape
// a create produces, so it reuses backendResponse for the identifier/mapping
// recombination. Mirrors connection-crud.service.ts mergeUpdate.
func backendUpdate(stored *retlClient.RETLConnection, request *retlClient.UpdateRETLConnectionRequest) *retlClient.RETLConnection {
	constants := stored.Constants
	if request.Constants != nil {
		constants = *request.Constants
	}
	mappings := stored.Mappings
	if request.Mappings != nil {
		mappings = *request.Mappings
	}
	settings := stored.SyncSettings
	if request.SyncSettings != nil {
		settings = request.SyncSettings
	}

	return backendResponse(&retlClient.CreateRETLConnectionRequest{
		SourceID:      stored.SourceID,
		DestinationID: stored.DestinationID,
		Enabled:       request.Enabled,
		Schedule:      request.Schedule,
		SyncBehaviour: lo.ToPtr(stored.SyncBehaviour),
		Identifiers:   request.Identifiers,
		Mappings:      mappings,
		Constants:     constants,
		Event:         stored.Event,
		CursorColumn:  stored.CursorColumn,
		Object:        stored.Object,
		SyncSettings:  settings,
	})
}

// The create round trip proves a new connection converges; this proves an
// edited one does. A PUT that reads back as anything but the desired config
// would re-diff on every apply.
func TestUpdateRoundTrip(t *testing.T) {
	t.Parallel()

	withConstants := jsonMapperConfig()
	withConstants.Constants = []ConstantSpec{{Key: "source", Value: "warehouse"}}

	otherConstants := jsonMapperConfig()
	otherConstants.Constants = []ConstantSpec{{Key: "source", Value: "lake"}}

	moreMappings := jsonMapperConfig()
	moreMappings.Mappings = []MappingSpec{{From: "email", To: "traits.email"}, {From: "name", To: "traits.name"}}

	nondefaultSettings := jsonMapperConfig()
	nondefaultSettings.SyncSettings = &SyncSettingsSpec{SyncLogs: &SyncLogsSpec{Enabled: ptr(false), SnapshotsToRetain: ptr(0)}}

	objectWithMappings := objectMappingConfig()
	objectWithMappings.Mappings = []MappingSpec{{From: "email", To: "Email"}}

	objectCleared := objectMappingConfig()
	objectCleared.Mappings = []MappingSpec{}

	tests := []struct {
		name         string
		stored, want ConfigSpec
	}{
		{name: "constants added", stored: jsonMapperConfig(), want: withConstants},
		{name: "constants changed", stored: withConstants, want: otherConstants},
		{name: "constants cleared", stored: withConstants, want: jsonMapperConfig()},
		{name: "mappings added", stored: jsonMapperConfig(), want: moreMappings},
		{name: "object mappings cleared", stored: objectWithMappings, want: objectCleared},
		{name: "sync settings made nondefault", stored: jsonMapperConfig(), want: nondefaultSettings},
		{name: "nondefault sync settings removed", stored: nondefaultSettings, want: jsonMapperConfig()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			created, err := toCreateRequest(graphData(t, tc.stored))
			require.NoError(t, err)
			stored := backendResponse(created)

			request, err := toUpdateRequest(graphData(t, tc.want), graphData(t, tc.stored))
			require.NoError(t, err)

			remote, err := configFromRemote(backendUpdate(stored, request))
			require.NoError(t, err)
			assert.Equal(t, normalizeConfig(tc.want), remote)
		})
	}
}
