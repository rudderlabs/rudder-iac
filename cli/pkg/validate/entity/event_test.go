package entity_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventValidationRules_RequiredKeys(t *testing.T) {
	var rule entity.EventRequiredKeysRule

	dc := &localcatalog.DataCatalog{
		Events: map[localcatalog.EntityGroup][]*localcatalog.Event{
			"group_1": {
				{
					LocalID: "", // missing local_id
					Name:    "", // missing required key
					Type:    "track",
				}, {
					LocalID: "event_2",
					Name:    "event_2",  // display_name field not allowed
					Type:    "identify", // non-track event
				},
			},

			"group_2": {
				{
					LocalID: "", // missing local_id
					Name:    "",
					Type:    "invalid_event_type", // invalid type
				},
			},
		},
	}

	t.Run("track event", func(t *testing.T) {
		t.Parallel()

		errs := rule.Validate("#/events/group_1/event_1", dc.Events["group_1"][0], dc)
		require.Len(t, errs, 2)

		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/events/group_1/event_1",
			Err:        entity.ErrMissingRequiredKeysID,
			EntityType: entity.Event,
		})

		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/events/group_1/event_1",
			Err:        entity.ErrMissingRequiredKeysName,
			EntityType: entity.Event,
		})
	})

	t.Run("non-track event", func(t *testing.T) {
		t.Parallel()

		errs := rule.Validate("#/events/group_1/event_2", dc.Events["group_1"][1], dc)
		require.Len(t, errs, 1)

		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/events/group_1/event_2",
			Err:        entity.ErrNotAllowedKeyName,
			EntityType: entity.Event,
		})
	})

	t.Run("invalid event type", func(t *testing.T) {
		t.Parallel()

		errs := rule.Validate("#/events/group_2/event_3", dc.Events["group_2"][0], dc)
		require.Len(t, errs, 2)

		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/events/group_2/event_3",
			Err:        entity.ErrMissingRequiredKeysID,
			EntityType: entity.Event,
		})
		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/events/group_2/event_3",
			Err:        entity.ErrInvalidRequiredKeysEventType,
			EntityType: entity.Event,
		})
	})
}

func TestEventValidationRules_DuplicateKeys(t *testing.T) {
	var rule entity.EventDuplicateKeysRule

	dc := &localcatalog.DataCatalog{
		Events: map[localcatalog.EntityGroup][]*localcatalog.Event{
			"group_1": {
				{
					LocalID: "event_1", // duplicate id within group_1
					Name:    "event_1",
					Type:    "track",
				},
				{
					LocalID: "event_1",
					Name:    "event_2",
					Type:    "track",
				},
				{
					LocalID: "event_3", // duplicate id with group_2
					Name:    "event_3", // duplicate name with group_2
					Type:    "track",
				},
			},
			"group_2": {
				{
					LocalID: "event_3",
					Name:    "event_3",
					Type:    "track",
				},
			},
		},
	}

	t.Run("duplicate id and names", func(t *testing.T) {
		t.Parallel()

		errs := rule.Validate("#/events/group_1/event_1", dc.Events["group_1"][0], dc)
		require.Len(t, errs, 1)

		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/events/group_1/event_1",
			Err:        entity.ErrDuplicateByID,
			EntityType: entity.Event,
		})

		errs = rule.Validate("#/events/group_1/event_3", dc.Events["group_1"][2], dc)
		require.Len(t, errs, 2)

		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/events/group_1/event_3",
			Err:        entity.ErrDuplicateByID,
			EntityType: entity.Event,
		})

		assert.Contains(t, errs, entity.ValidationError{
			Reference:  "#/events/group_1/event_3",
			Err:        entity.ErrDuplicateByName,
			EntityType: entity.Event,
		})

	})
}
