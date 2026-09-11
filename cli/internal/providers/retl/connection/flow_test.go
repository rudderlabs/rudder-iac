package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsesDestinationSpecificFlow(t *testing.T) {
	t.Parallel()

	assert.True(t, UsesDestinationSpecificFlow("CUSTOMERIO"))
	assert.True(t, UsesDestinationSpecificFlow("CUSTOMERIO_AUDIENCE"))
	assert.False(t, UsesDestinationSpecificFlow("WEBHOOK"))
	// API types are upper case everywhere in the registry, so the match is
	// exact rather than case-insensitive.
	assert.False(t, UsesDestinationSpecificFlow("customerio"))
}

func TestClassifyFlow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		apiType              string
		supportsVisualMapper bool
		object               *string
		want                 Flow
		wantErr              string
	}{
		{
			name:    "no object is a json mapper connection",
			apiType: "WEBHOOK",
			want:    FlowJSONMapper,
		},
		{
			name:                 "a mapper destination without an object stays a json mapper connection",
			apiType:              "AM",
			supportsVisualMapper: true,
			want:                 FlowJSONMapper,
		},
		{
			name:                 "a mapper destination with an object maps objects",
			apiType:              "AM",
			supportsVisualMapper: true,
			object:               ptr("users"),
			want:                 FlowObjectMapping,
		},
		{
			name:    "an object on a non-mapper destination",
			apiType: "WEBHOOK",
			object:  ptr("users"),
			wantErr: `'object' is not allowed: destination api type "WEBHOOK" does not support object mapping`,
		},
		{
			name:                 "an explicitly empty object",
			apiType:              "AM",
			supportsVisualMapper: true,
			object:               ptr(""),
			wantErr:              "'object' must not be empty",
		},
		{
			name:    "a destination-specific flow",
			apiType: "CUSTOMERIO",
			wantErr: `destination api type "CUSTOMERIO" uses a destination-specific rETL flow, which is not supported`,
		},
		{
			// Every other input here says object mapping, so this pins that the
			// destination-specific check wins over the rest of the table.
			name:                 "a destination-specific flow that otherwise maps objects",
			apiType:              "CUSTOMERIO_AUDIENCE",
			supportsVisualMapper: true,
			object:               ptr("users"),
			wantErr:              `destination api type "CUSTOMERIO_AUDIENCE" uses a destination-specific rETL flow, which is not supported`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			flow, err := ClassifyFlow(test.apiType, test.supportsVisualMapper, test.object)
			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
				assert.Empty(t, flow)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, flow)
		})
	}
}
