package connection

import (
	"encoding/json"
	"fmt"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/samber/lo"
)

// endpointIDFromData reads one of the two endpoint ids out of a graph entry. The
// syncer dereferences the spec's PropertyRefs before the lifecycle runs, so
// anything but a nonempty string here means the reference never resolved and
// the request would be built against a dangling endpoint.
func endpointIDFromData(data resources.ResourceData, key string) (string, error) {
	id, ok := data[key].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("connection data: %q is not a resolved endpoint id (got %T)", key, data[key])
	}
	return id, nil
}

func enabledFromData(data resources.ResourceData) (bool, error) {
	value, ok := data[EnabledKey].(bool)
	if !ok {
		return false, fmt.Errorf("connection data: %q is not a bool (got %T)", EnabledKey, data[EnabledKey])
	}
	return value, nil
}

// toCreateRequest builds the flattened POST /v2/retl-connections body: the API
// takes every setting at the top level and derives the flow itself from the
// destination and the object, so there is no nested config field to fill.
// externalId is left out on purpose — the per-flow allow-list rejects it on
// create, and the handler claims the identity in a separate call afterwards.
func toCreateRequest(data resources.ResourceData) (*retlClient.CreateRETLConnectionRequest, error) {
	sourceID, err := endpointIDFromData(data, SourceKey)
	if err != nil {
		return nil, err
	}

	destinationID, err := endpointIDFromData(data, DestinationKey)
	if err != nil {
		return nil, err
	}

	enabled, err := enabledFromData(data)
	if err != nil {
		return nil, err
	}

	config, err := configFromMap(data[ConfigKey])
	if err != nil {
		return nil, err
	}

	request := &retlClient.CreateRETLConnectionRequest{
		SourceID:      sourceID,
		DestinationID: destinationID,
		Enabled:       &enabled,
		Schedule:      toAPISchedule(config.Schedule),
		SyncBehaviour: lo.ToPtr(retlClient.SyncBehaviour(config.SyncBehaviour)),
		Identifiers:   toAPIMappings(config.Identifiers),
		Mappings:      toAPIMappings(config.Mappings),
		Event:         toAPIEvent(config.Event),
		CursorColumn:  config.CursorColumn,
		Object:        lo.FromPtr(config.Object),
	}
	// Constants belong to the JSON mapper flow only; the per-flow allow-list
	// rejects them on an object mapping create just as it does on update.
	if config.Object == nil {
		request.Constants = toAPIConstants(config.Constants)
	}
	// The config map is canonical, so sync settings are present only when they
	// differ from what the server fills in by itself.
	if config.SyncSettings != nil {
		request.SyncSettings = toAPISyncSettings(config.SyncSettings)
	}
	return request, nil
}

// toUpdateRequest builds the PUT body from the desired graph entry and the
// stored state. Only the mutable fields exist on the request type, so the
// immutable ones — endpoints, syncBehaviour, object, event, cursorColumn —
// cannot leak into it; the API rejects a body carrying them even unchanged.
// Mappings and constants travel as pointers so that clearing them serializes
// as an empty array instead of vanishing through omitempty, which the server
// reads as "keep what is stored".
func toUpdateRequest(data, state resources.ResourceData) (*retlClient.UpdateRETLConnectionRequest, error) {
	enabled, err := enabledFromData(data)
	if err != nil {
		return nil, err
	}

	config, err := configFromMap(data[ConfigKey])
	if err != nil {
		return nil, err
	}

	stored, err := configFromMap(state[ConfigKey])
	if err != nil {
		return nil, fmt.Errorf("reading stored connection config: %w", err)
	}

	request := &retlClient.UpdateRETLConnectionRequest{
		Enabled:     &enabled,
		Schedule:    toAPISchedule(config.Schedule),
		Identifiers: toAPIMappings(config.Identifiers),
		Mappings:    lo.ToPtr(toAPIMappings(config.Mappings)),
	}
	// Constants are mutable on the JSON mapper flow only; an object mapping PUT
	// that carries them, an empty list included, is refused as an immutable
	// field. An object is what tells the two flows apart, here as in the API.
	if config.Object == nil {
		request.Constants = lo.ToPtr(toAPIConstants(config.Constants))
	}
	// Omitting syncSettings keeps whatever the server stored, so dropping
	// nondefault settings has to be spelled out as the defaults a fresh create
	// would have filled in.
	if config.SyncSettings != nil || stored.SyncSettings != nil {
		request.SyncSettings = toAPISyncSettings(config.SyncSettings)
	}
	return request, nil
}

// toResourceData is the lifecycle's state output: the remote identifiers only. The
// config and the enabled flag stay on the state input the syncer keeps.
func toResourceData(conn *retlClient.RETLConnection) *resources.ResourceData {
	return &resources.ResourceData{
		IDKey:            conn.ID,
		SourceIDKey:      conn.SourceID,
		DestinationIDKey: conn.DestinationID,
	}
}

// configFromRemote turns an API connection back into the config that produces
// it.
//
// It assumes the response satisfies the supported flow contracts, and that is
// what makes the conversion lossless. The backend combines identifiers with
// mappings when it stores a connection and reconstructs them on read: a JSON
// mapper mapping aimed at user_id or anonymous_id reappears as an identifier,
// and an object mapping one aimed at user_id or context.externalId[0].id is
// consumed as a synthetic identifier and vanishes from the user mappings. Those
// targets are therefore forbidden in user mappings — DEX-829 validates it — and
// nothing here moves or drops entries to paper over one. Destination and source
// eligibility is checked before this runs.
func configFromRemote(conn *retlClient.RETLConnection) (ConfigSpec, error) {
	if hasDestinationConfig(conn.DestinationConfig) {
		return ConfigSpec{}, fmt.Errorf("connection %q: destination-specific configuration has no spec equivalent", conn.ID)
	}

	config := ConfigSpec{
		SyncBehaviour: string(conn.SyncBehaviour),
		Schedule:      fromAPISchedule(conn.Schedule),
		Identifiers:   fromAPIMappings(conn.Identifiers),
		Mappings:      fromAPIMappings(conn.Mappings),
		Constants:     fromAPIConstants(conn.Constants),
		Event:         fromAPIEvent(conn.Event),
		CursorColumn:  conn.CursorColumn,
		SyncSettings:  fromAPISyncSettings(conn.SyncSettings),
	}
	// A JSON mapper connection has no object at all, and the API sends the
	// field empty rather than absent; the spec has to omit it entirely for the
	// two to compare equal.
	if conn.Object != "" {
		config.Object = lo.ToPtr(conn.Object)
	}
	return canonicalConfig(config), nil
}

// hasDestinationConfig reports whether a response carries integration-owned
// destination settings. They belong to the destination-specific flow, which
// the spec has no field for, so a connection carrying them cannot round-trip
// and must be refused rather than silently stripped. It reports true for
// anything it cannot prove empty, malformed payloads included.
func hasDestinationConfig(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var fields map[string]json.RawMessage
	// Only an empty object or null proves the field carries nothing. A scalar,
	// an array or malformed bytes cannot, so they are refused rather than
	// dropped on the assumption that they are empty.
	if err := json.Unmarshal(raw, &fields); err != nil {
		return true
	}
	return len(fields) > 0
}

func toAPISchedule(schedule ScheduleSpec) retlClient.Schedule {
	converted := retlClient.Schedule{
		Type:         retlClient.ScheduleType(schedule.Type),
		EveryMinutes: schedule.EveryMinutes,
	}
	if schedule.CronExpression != "" {
		converted.CronExpression = lo.ToPtr(schedule.CronExpression)
	}
	return converted
}

func fromAPISchedule(schedule retlClient.Schedule) ScheduleSpec {
	return ScheduleSpec{
		Type:           string(schedule.Type),
		EveryMinutes:   schedule.EveryMinutes,
		CronExpression: lo.FromPtr(schedule.CronExpression),
	}
}

// toAPIMappings always returns a non-nil slice: an update that clears mappings
// has to serialize as [], and create's omitempty drops the empty slice anyway.
func toAPIMappings(mappings []MappingSpec) []retlClient.Mapping {
	converted := make([]retlClient.Mapping, len(mappings))
	for i, mapping := range mappings {
		converted[i] = retlClient.Mapping{From: mapping.From, To: mapping.To}
	}
	return converted
}

func fromAPIMappings(mappings []retlClient.Mapping) []MappingSpec {
	if len(mappings) == 0 {
		return nil
	}
	converted := make([]MappingSpec, len(mappings))
	for i, mapping := range mappings {
		converted[i] = MappingSpec{From: mapping.From, To: mapping.To}
	}
	return converted
}

func toAPIConstants(constants []ConstantSpec) []retlClient.Constant {
	converted := make([]retlClient.Constant, len(constants))
	for i, constant := range constants {
		converted[i] = retlClient.Constant{Key: constant.Key, Value: constant.Value}
	}
	return converted
}

func fromAPIConstants(constants []retlClient.Constant) []ConstantSpec {
	if len(constants) == 0 {
		return nil
	}
	converted := make([]ConstantSpec, len(constants))
	for i, constant := range constants {
		converted[i] = ConstantSpec{Key: constant.Key, Value: constant.Value}
	}
	return converted
}

func toAPIEvent(event *EventSpec) *retlClient.Event {
	if event == nil {
		return nil
	}
	return &retlClient.Event{
		Type:       retlClient.EventType(event.Type),
		Name:       event.Name,
		NameColumn: event.NameColumn,
	}
}

func fromAPIEvent(event *retlClient.Event) *EventSpec {
	if event == nil {
		return nil
	}
	return &EventSpec{
		Type:       string(event.Type),
		Name:       event.Name,
		NameColumn: event.NameColumn,
	}
}

// toAPISyncSettings renders the complete settings object the API stores. A nil
// spec is the fully defaulted object — canonicalConfig collapses exactly that
// to nil — so an update that has to state the settings explicitly can pass nil
// to reset them.
func toAPISyncSettings(settings *SyncSettingsSpec) *retlClient.SyncSettings {
	filled := syncSettingsWithDefaults(settings)
	return &retlClient.SyncSettings{
		SyncLogsConfig: &retlClient.SyncLogsConfig{
			Enabled:            filled.SyncLogs.Enabled,
			LogRetentionInDays: filled.SyncLogs.RetentionDays,
			SnapshotsToRetain:  filled.SyncLogs.SnapshotsToRetain,
		},
		FailedKeysConfig: &retlClient.FailedKeysConfig{
			EnableFailedKeysRetry: filled.FailedKeys.Retry,
		},
	}
}

func fromAPISyncSettings(settings *retlClient.SyncSettings) *SyncSettingsSpec {
	if settings == nil {
		return nil
	}

	converted := &SyncSettingsSpec{}
	if logs := settings.SyncLogsConfig; logs != nil {
		converted.SyncLogs = &SyncLogsSpec{
			Enabled:           logs.Enabled,
			RetentionDays:     logs.LogRetentionInDays,
			SnapshotsToRetain: logs.SnapshotsToRetain,
		}
	}
	if failed := settings.FailedKeysConfig; failed != nil {
		converted.FailedKeys = &FailedKeysSpec{Retry: failed.EnableFailedKeysRetry}
	}
	return converted
}
