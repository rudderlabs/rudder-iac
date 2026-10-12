package golang_test

import (
	"regexp"
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// payloadAndMethods matches the declarations of the payload types and of the
// event methods, without their bodies.
var payloadAndMethods = regexp.MustCompile(`(?m)^(type \w+(Properties|Traits)\d* .*?|func \(r \*RudderTyperAnalytics\) [A-Z].*?)( \{)?$`)

// One row per event type, section and schema shape: the payload type and
// method each generates, or the warning it is skipped with.
func TestGenerateEventTypes(t *testing.T) {
	rule := func(eventType plan.EventType, name string, section plan.IdentitySection) plan.EventRule {
		return plan.EventRule{Event: plan.Event{EventType: eventType, Name: name}, Section: section}
	}

	tests := []struct {
		name         string
		rules        []plan.EventRule
		want         []string
		wantWarnings []string
	}{
		{
			name:         "track on traits",
			rules:        []plan.EventRule{rule(plan.EventTypeTrack, "Some Event", plan.IdentitySectionTraits)},
			wantWarnings: []string{`invalid section "traits" for event type "track" and event name "Some Event"`},
		},
		{
			name:         "identify on properties",
			rules:        []plan.EventRule{rule(plan.EventTypeIdentify, "", plan.IdentitySectionProperties)},
			wantWarnings: []string{`invalid section "properties" for event type "identify"`},
		},
		{
			name:         "group on properties",
			rules:        []plan.EventRule{rule(plan.EventTypeGroup, "", plan.IdentitySectionProperties)},
			wantWarnings: []string{`invalid section "properties" for event type "group"`},
		},
		{
			name:         "page on traits",
			rules:        []plan.EventRule{rule(plan.EventTypePage, "", plan.IdentitySectionTraits)},
			wantWarnings: []string{`invalid section "traits" for event type "page"`},
		},
		{
			name:         "screen on context.traits",
			rules:        []plan.EventRule{rule(plan.EventTypeScreen, "", plan.IdentitySectionContextTraits)},
			wantWarnings: []string{`invalid section "context.traits" for event type "screen"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := captureWarnings(t)

			files, err := (&golang.Generator{}).Generate(&plan.TrackingPlan{Rules: tt.rules}, core.GenerateOptions{}, nil)
			require.NoError(t, err)
			require.Len(t, files, 1)

			var got []string
			for _, m := range payloadAndMethods.FindAllStringSubmatch(files[0].Content, -1) {
				got = append(got, m[1])
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantWarnings, *warnings)
		})
	}
}
