package connections

import (
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
		ExternalID:    externalID,
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
