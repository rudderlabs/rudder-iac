package connection

import (
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestConnectionSpecSyntaxValidRule_Metadata(t *testing.T) {
	rule := NewConnectionSpecSyntaxValidRule()

	assert.Equal(t, "event-stream/connection/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "event stream connection spec syntax must be valid", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(esConnection.EventStreamConnectionResourceKind), rule.AppliesTo())
}

func TestConnectionSpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec esConnection.ConnectionsSpec
	}{
		{
			name: "minimal connection",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#event-stream-source:src-1",
						Destination: "#destination:dest-1",
					},
				},
			},
		},
		{
			name: "connection with enabled set",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#event-stream-source:src-1",
						Destination: "#destination:dest-1",
						Enabled:     boolPtr(false),
					},
				},
			},
		},
		{
			name: "multiple connections",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#event-stream-source:src-1",
						Destination: "#destination:dest-1",
					},
					{
						LocalID:     "conn-2",
						Source:      "#event-stream-source:src-2",
						Destination: "#destination:dest-1",
					},
				},
			},
		},
		{
			// Refs with surrounding whitespace apply cleanly (the handler
			// trims before parsing), so validation accepts them too.
			name: "refs with surrounding whitespace",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      " #event-stream-source:src-1 ",
						Destination: " #destination:dest-1 ",
					},
				},
			},
		},
		{
			// Endpoint local ids carry no charset restriction and the handler
			// accepts any non-empty id — even one spanning a newline — so the
			// ref check must be equally lenient.
			name: "ref id with unusual characters",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#event-stream-source:src.1",
						Destination: "#destination:dest\n1",
					},
				},
			},
		},
		{
			// An empty (but present) connections list is a valid no-op spec.
			name: "empty connections list",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := validateConnectionsSpec("", "", nil, tt.spec)
			assert.Empty(t, results, "expected no validation errors")
		})
	}
}

func TestConnectionSpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		spec        esConnection.ConnectionsSpec
		wantResults []rules.ValidationResult
	}{
		{
			name: "missing connections",
			spec: esConnection.ConnectionsSpec{},
			wantResults: []rules.ValidationResult{
				{Reference: "/connections", Message: "'connections' is required"},
			},
		},
		{
			name: "missing id",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						Source:      "#event-stream-source:src-1",
						Destination: "#destination:dest-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{Reference: "/connections/0/id", Message: "'id' is required"},
			},
		},
		{
			name: "missing source",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Destination: "#destination:dest-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{Reference: "/connections/0/source", Message: "'source' is required"},
			},
		},
		{
			name: "missing destination",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID: "conn-1",
						Source:  "#event-stream-source:src-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{Reference: "/connections/0/destination", Message: "'destination' is required"},
			},
		},
		{
			name: "all required fields missing",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{{}},
			},
			wantResults: []rules.ValidationResult{
				{Reference: "/connections/0/id", Message: "'id' is required"},
				{Reference: "/connections/0/source", Message: "'source' is required"},
				{Reference: "/connections/0/destination", Message: "'destination' is required"},
			},
		},
		{
			name: "source ref not a reference",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "src-1",
						Destination: "#destination:dest-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{
					Reference: "/connections/0/source",
					Message:   "'source' is invalid: must be of pattern #event-stream-source:<id>",
				},
			},
		},
		{
			name: "source ref with empty id",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#event-stream-source:",
						Destination: "#destination:dest-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{
					Reference: "/connections/0/source",
					Message:   "'source' is invalid: must be of pattern #event-stream-source:<id>",
				},
			},
		},
		{
			name: "source ref of wrong family",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#retl-source-sql-model:my-model",
						Destination: "#destination:dest-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{
					Reference: "/connections/0/source",
					Message:   "'source' must reference an event stream source (#event-stream-source:<id>), got a 'retl-source-sql-model' reference",
				},
			},
		},
		{
			name: "source ref pointing at a destination",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#destination:dest-1",
						Destination: "#destination:dest-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{
					Reference: "/connections/0/source",
					Message:   "'source' must reference an event stream source (#event-stream-source:<id>), got a 'destination' reference",
				},
			},
		},
		{
			name: "destination ref not a reference",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#event-stream-source:src-1",
						Destination: "dest-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{
					Reference: "/connections/0/destination",
					Message:   "'destination' is invalid: must be of pattern #destination:<id>",
				},
			},
		},
		{
			name: "destination ref pointing at a source",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#event-stream-source:src-1",
						Destination: "#event-stream-source:src-2",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{
					Reference: "/connections/0/destination",
					Message:   "'destination' must reference a destination (#destination:<id>), got a 'event-stream-source' reference",
				},
			},
		},
		{
			name: "errors reported per entry",
			spec: esConnection.ConnectionsSpec{
				Connections: []esConnection.ConnectionSpec{
					{
						LocalID:     "conn-1",
						Source:      "#event-stream-source:src-1",
						Destination: "#destination:dest-1",
					},
					{
						LocalID:     "conn-2",
						Source:      "bad-ref",
						Destination: "#event-stream-source:src-1",
					},
				},
			},
			wantResults: []rules.ValidationResult{
				{
					Reference: "/connections/1/source",
					Message:   "'source' is invalid: must be of pattern #event-stream-source:<id>",
				},
				{
					Reference: "/connections/1/destination",
					Message:   "'destination' must reference a destination (#destination:<id>), got a 'event-stream-source' reference",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := validateConnectionsSpec("", "", nil, tt.spec)
			require.Len(t, results, len(tt.wantResults))
			assert.ElementsMatch(t, tt.wantResults, results)
		})
	}
}
