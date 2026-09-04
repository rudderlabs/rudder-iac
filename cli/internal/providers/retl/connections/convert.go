package connections

import (
	"encoding/json"
	"fmt"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// refValue reads an endpoint id from resource data. By the time the lifecycle
// runs, the syncer has dereferenced the PropertyRefs the graph was built from,
// so the value is a plain string.
func refValue(data resources.ResourceData, key string) (string, error) {
	switch v := data[key].(type) {
	case string:
		if v == "" {
			return "", fmt.Errorf("%s resolved to an empty id", key)
		}
		return v, nil
	case *resources.PropertyRef:
		return "", fmt.Errorf("%s was not dereferenced before the lifecycle call", key)
	default:
		return "", fmt.Errorf("%s is missing or of unexpected type %T", key, data[key])
	}
}

func toSchedule(data resources.ResourceData) retlClient.Schedule {
	spec, ok := data[ScheduleKey].(ScheduleSpec)
	if !ok {
		return retlClient.Schedule{}
	}
	s := retlClient.Schedule{Type: retlClient.ScheduleType(spec.Type)}
	if spec.EveryMinutes != nil {
		s.EveryMinutes = spec.EveryMinutes
	}
	if spec.CronExpression != "" {
		cron := spec.CronExpression
		s.CronExpression = &cron
	}
	return s
}

func toEvent(data resources.ResourceData) *retlClient.Event {
	spec, ok := data[EventKey].(*EventSpec)
	if !ok || spec == nil {
		return nil
	}
	return &retlClient.Event{
		Type:       retlClient.EventType(spec.Type),
		Name:       spec.Name,
		NameColumn: spec.NameColumn,
	}
}

func toMappings(data resources.ResourceData, key string) []retlClient.Mapping {
	specs, ok := data[key].([]MappingSpec)
	if !ok || len(specs) == 0 {
		return nil
	}
	out := make([]retlClient.Mapping, 0, len(specs))
	for _, m := range specs {
		out = append(out, retlClient.Mapping{From: m.From, To: m.To})
	}
	return out
}

func toConstants(data resources.ResourceData) []retlClient.Constant {
	specs, ok := data[ConstantsKey].([]ConstantSpec)
	if !ok || len(specs) == 0 {
		return nil
	}
	out := make([]retlClient.Constant, 0, len(specs))
	for _, c := range specs {
		out = append(out, retlClient.Constant{Key: c.Key, Value: c.Value})
	}
	return out
}

// toSyncSettings sends only what the user declared. Every level is a pointer:
// an omitted section stays nil and serialises away, so the server merges it
// against the stored value instead of resetting it to a default. Populating a
// full object here would reproduce the partial-update clobber that config
// backend #6598 fixes for Terraform.
func toSyncSettings(data resources.ResourceData) *retlClient.SyncSettings {
	spec, ok := data[SyncSettingsKey].(*SyncSettingsSpec)
	if !ok || spec == nil {
		return nil
	}
	out := &retlClient.SyncSettings{}
	if spec.SyncLogs != nil {
		out.SyncLogsConfig = &retlClient.SyncLogsConfig{
			Enabled:            spec.SyncLogs.Enabled,
			LogRetentionInDays: spec.SyncLogs.RetentionDays,
			SnapshotsToRetain:  spec.SyncLogs.SnapshotsToRetain,
		}
	}
	if spec.FailedKeys != nil {
		out.FailedKeysConfig = &retlClient.FailedKeysConfig{
			EnableFailedKeysRetry: spec.FailedKeys.Retry,
		}
	}
	if out.SyncLogsConfig == nil && out.FailedKeysConfig == nil {
		return nil
	}
	return out
}

// toCreateRequest deliberately omits ExternalID — see Handler.Create.
func toCreateRequest(externalID string, data resources.ResourceData) (*retlClient.CreateRETLConnectionRequest, error) {
	sourceID, err := refValue(data, SourceKey)
	if err != nil {
		return nil, err
	}
	destinationID, err := refValue(data, DestinationKey)
	if err != nil {
		return nil, err
	}

	enabled, _ := data[EnabledKey].(bool)
	req := &retlClient.CreateRETLConnectionRequest{
		SourceID:      sourceID,
		DestinationID: destinationID,
		Enabled:       &enabled,
		Schedule:      toSchedule(data),
		SyncSettings:  toSyncSettings(data),
		Identifiers:   toMappings(data, IdentifiersKey),
		Mappings:      toMappings(data, MappingsKey),
		Event:         toEvent(data),
		Constants:     toConstants(data),
	}
	if sb, ok := data[SyncBehaviourKey].(string); ok && sb != "" {
		behaviour := retlClient.SyncBehaviour(sb)
		req.SyncBehaviour = &behaviour
	}
	if cc, ok := data[CursorColumnKey].(string); ok {
		req.CursorColumn = cc
	}
	if dc, ok := data[DestinationConfigKey].(map[string]any); ok && len(dc) > 0 {
		raw, err := json.Marshal(dc)
		if err != nil {
			return nil, fmt.Errorf("encoding destination_config: %w", err)
		}
		req.DestinationConfig = raw
	}
	return req, nil
}

// toUpdateRequest carries only the mutable fields. sync_behaviour,
// cursor_column, object, event, source and destination are absent from
// UpdateRETLConnectionRequest by design — the API treats them as immutable, so
// a change to any of them needs a replace rather than an update.
//
// Mappings and Constants are pointer-to-slice in the request so a nil means
// "not provided" and an empty slice means "clear these". Passing nil when the
// user declared none preserves whatever the server holds.
func toUpdateRequest(data resources.ResourceData) (*retlClient.UpdateRETLConnectionRequest, error) {
	enabled, _ := data[EnabledKey].(bool)
	req := &retlClient.UpdateRETLConnectionRequest{
		Enabled:      &enabled,
		Schedule:     toSchedule(data),
		SyncSettings: toSyncSettings(data),
		Identifiers:  toMappings(data, IdentifiersKey),
	}
	if m := toMappings(data, MappingsKey); m != nil {
		req.Mappings = &m
	}
	if c := toConstants(data); c != nil {
		req.Constants = &c
	}
	return req, nil
}

func toResourceData(c *retlClient.RETLConnection) *resources.ResourceData {
	return &resources.ResourceData{
		IDKey:            c.ID,
		SourceIDKey:      c.SourceID,
		DestinationIDKey: c.DestinationID,
		ExternalIDKey:    c.ExternalID,
		EnabledKey:       c.Enabled,
		SyncBehaviourKey: string(c.SyncBehaviour),
		CursorColumnKey:  c.CursorColumn,
		ObjectKey:        c.Object,
	}
}

// toSpecShapedInput rebuilds the spec-side view of a remote connection. It must
// mirror exactly what GetResources emits, or every plan reports a spurious
// update: the differ compares the union of keys, so a state map using different
// key names (sourceId vs source) or omitting fields produces a permanent diff.
func toSpecShapedInput(c *retlClient.RETLConnection, sourceURN, destinationURN string) resources.ResourceData {
	data := resources.ResourceData{
		SourceKey:        &resources.PropertyRef{URN: sourceURN, Property: "id"},
		DestinationKey:   &resources.PropertyRef{URN: destinationURN, Property: "id"},
		EnabledKey:       c.Enabled,
		SyncBehaviourKey: string(c.SyncBehaviour),
		CursorColumnKey:  c.CursorColumn,
		ObjectKey:        c.Object,
		ScheduleKey:      scheduleSpecFrom(c.Schedule),
		EventKey:         eventSpecFrom(c.Event),
		IdentifiersKey:   mappingSpecsFrom(c.Identifiers),
		MappingsKey:      mappingSpecsFrom(c.Mappings),
		ConstantsKey:     constantSpecsFrom(c.Constants),
		SyncSettingsKey:  syncSettingsSpecFrom(c.SyncSettings),
	}
	if len(c.DestinationConfig) > 0 {
		var dc map[string]any
		if err := json.Unmarshal(c.DestinationConfig, &dc); err == nil {
			data[DestinationConfigKey] = dc
		}
	}
	return data
}

func scheduleSpecFrom(s retlClient.Schedule) ScheduleSpec {
	out := ScheduleSpec{Type: string(s.Type), EveryMinutes: s.EveryMinutes}
	if s.CronExpression != nil {
		out.CronExpression = *s.CronExpression
	}
	return out
}

func eventSpecFrom(e *retlClient.Event) *EventSpec {
	if e == nil {
		return nil
	}
	return &EventSpec{Type: string(e.Type), Name: e.Name, NameColumn: e.NameColumn}
}

func mappingSpecsFrom(ms []retlClient.Mapping) []MappingSpec {
	if len(ms) == 0 {
		return nil
	}
	out := make([]MappingSpec, 0, len(ms))
	for _, m := range ms {
		out = append(out, MappingSpec{From: m.From, To: m.To})
	}
	return out
}

func constantSpecsFrom(cs []retlClient.Constant) []ConstantSpec {
	if len(cs) == 0 {
		return nil
	}
	out := make([]ConstantSpec, 0, len(cs))
	for _, c := range cs {
		out = append(out, ConstantSpec{Key: c.Key, Value: c.Value})
	}
	return out
}

// syncSettingsSpecFrom reconstructs the remote sync settings.
//
// ponytail: this is where declarative diffing and merge semantics collide. The
// server fills defaults on create, so remote always carries a FULL object while
// a spec may declare a subset — the differ then reports a change forever. Until
// the differ can compare declared fields only, a spec that declares
// sync_settings must declare every field it cares about. Tracked as a spike
// finding, not solved here.
func syncSettingsSpecFrom(s *retlClient.SyncSettings) *SyncSettingsSpec {
	if s == nil {
		return nil
	}
	out := &SyncSettingsSpec{}
	if s.SyncLogsConfig != nil {
		out.SyncLogs = &SyncLogsSpec{
			Enabled:           s.SyncLogsConfig.Enabled,
			RetentionDays:     s.SyncLogsConfig.LogRetentionInDays,
			SnapshotsToRetain: s.SyncLogsConfig.SnapshotsToRetain,
		}
	}
	if s.FailedKeysConfig != nil {
		out.FailedKeys = &FailedKeysSpec{Retry: s.FailedKeysConfig.EnableFailedKeysRetry}
	}
	return out
}
