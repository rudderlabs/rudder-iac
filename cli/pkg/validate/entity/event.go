package entity

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/samber/lo"
)

var _ CatalogEntityValidator = &EventValidator{}

var (
	TrackEventTypes    = []string{"track"}
	NonTrackEventTypes = []string{"identify", "group", "page", "screen"}
)

type EventValidationRule interface {
	Validate(ref string, ev *localcatalog.Event, dc *localcatalog.DataCatalog) []ValidationError
}

type EventValidator struct {
	rules []EventValidationRule
}

func NewEventValidator(rules []EventValidationRule) CatalogEntityValidator {
	return &EventValidator{
		rules,
	}
}

func (ev *EventValidator) Validate(dc *localcatalog.DataCatalog) (errs []ValidationError) {
	log.Info("validating event entities in the catalog")

	for group, events := range dc.Events {
		for _, event := range events {

			reference := fmt.Sprintf(
				"#/events/%s/%s",
				group,
				event.LocalID,
			)

			for _, rule := range ev.rules {
				errs = append(errs, rule.Validate(reference, event, dc)...)
			}
		}
	}

	return errs
}

type EventRequiredKeysRule struct {
}

func (rule *EventRequiredKeysRule) Validate(ref string, ev *localcatalog.Event, dc *localcatalog.DataCatalog) (errs []ValidationError) {

	if ev.LocalID == "" {
		errs = append(errs, ValidationError{
			Reference:  ref,
			Err:        ErrMissingRequiredKeysID,
			EntityType: Event,
		})
	}

	switch ev.Type {

	case "track":
		errs = append(errs, rule.handleTrack(ref, ev)...)

	case "page", "screen", "identify", "group":
		errs = append(errs, rule.handleNonTrack(ref, ev)...)

	default:
		errs = append(errs, ValidationError{
			Reference:  ref,
			Err:        ErrInvalidRequiredKeysEventType,
			EntityType: Event,
		})
	}

	return errs
}

func (rule *EventRequiredKeysRule) handleTrack(ref string, ev *localcatalog.Event) (errs []ValidationError) {
	if ev.Name == "" {
		errs = append(errs, ValidationError{
			Reference:  ref,
			Err:        ErrMissingRequiredKeysName,
			EntityType: Event,
		})
	}
	return
}

func (rule *EventRequiredKeysRule) handleNonTrack(ref string, ev *localcatalog.Event) (err []ValidationError) {
	if ev.Name != "" {
		err = append(err, ValidationError{
			Reference:  ref,
			Err:        ErrNotAllowedKeyName,
			EntityType: Event,
		})
	}
	return
}

type EventDuplicateKeysRule struct {
}

// This rule checks for duplicate events in the catalog
// based on localID for all and only on display_name for track events
func (rule *EventDuplicateKeysRule) Validate(ref string, ev *localcatalog.Event, dc *localcatalog.DataCatalog) (errs []ValidationError) {
	byID := rule.eventsById(ev.LocalID, dc)

	if len(byID) > 1 {
		errs = append(errs, ValidationError{
			Reference:  ref,
			Err:        ErrDuplicateByID,
			EntityType: Event,
		})
	}

	// Only check valid track events with other track events
	// in terms of the display_name check
	byName := rule.eventsByName(ev.Name, dc, withTrackEvents)
	if lo.Contains(TrackEventTypes, ev.Type) && len(byName) > 1 {
		errs = append(errs, ValidationError{
			Reference:  ref,
			Err:        ErrDuplicateByName,
			EntityType: Event,
		})
	}

	return
}

func (rule *EventDuplicateKeysRule) eventsById(id string, dc *localcatalog.DataCatalog, filters ...filterEvent) []*localcatalog.Event {
	var events []*localcatalog.Event
	for _, e := range dc.Events {
		for _, event := range e {
			if event.LocalID == id {
				events = append(events, event)
			}
		}
	}

	return applyFilter(events, filters...)
}

func (rule *EventDuplicateKeysRule) eventsByName(name string, dc *localcatalog.DataCatalog, filters ...filterEvent) []*localcatalog.Event {
	var events []*localcatalog.Event
	for _, e := range dc.Events {
		for _, event := range e {
			if event.Name == name {
				events = append(events, event)
			}
		}
	}

	return applyFilter(events, filters...)
}

type filterEvent func(ev *localcatalog.Event) bool

var withTrackEvents = func(ev *localcatalog.Event) bool {
	return ev.Type == "track"
}

func applyFilter(events []*localcatalog.Event, filters ...filterEvent) []*localcatalog.Event {
	return lo.Filter(events, func(ev *localcatalog.Event, i int) bool {
		var filtered = true

		for _, filter := range filters {
			filtered = filtered && filter(ev)
		}

		return filtered
	})
}
