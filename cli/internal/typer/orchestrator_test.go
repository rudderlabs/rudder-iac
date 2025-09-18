package typer

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

// mockPlanProvider is a test implementation of PlanProvider
type mockPlanProvider struct{}

func (m *mockPlanProvider) GetTrackingPlan(ctx context.Context) (*plan.TrackingPlan, error) {
	return &plan.TrackingPlan{
		Name: "Test Tracking Plan",
		Rules: []plan.EventRule{
			{
				Event: plan.Event{
					EventType:   plan.EventTypeTrack,
					Name:        "TestEvent",
					Description: "Test event",
				},
				Section: plan.EventRuleSectionProperties,
				Schema: plan.ObjectSchema{
					Properties:           make(map[string]plan.PropertySchema),
					AdditionalProperties: false,
				},
			},
		},
	}, nil
}

func TestNewRudderTyper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "constructor with mock plan provider"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockProvider := &mockPlanProvider{}
			rudderTyper := NewRudderTyper(mockProvider)
			if rudderTyper == nil {
				t.Fatal("NewRudderTyper() returned nil")
			}
		})
	}
}

func TestGenerationOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options GenerationOptions
	}{
		{
			name: "kotlin output path default",
			options: GenerationOptions{
				Platform:   "kotlin",
				OutputPath: "./output",
			},
		},
		{
			name: "different values",
			options: GenerationOptions{
				Platform:   "kotlin",
				OutputPath: "/tmp/out",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// TrackingPlanID is now handled by the PlanProvider, not in options
			if tt.options.Platform != "kotlin" {
				t.Errorf("expected Platform 'kotlin', got %s", tt.options.Platform)
			}
			if tt.options.OutputPath == "" {
				t.Error("OutputPath should not be empty")
			}
		})
	}
}

func TestRudderTyper_Generate(t *testing.T) {
	t.Run("Generate (table)", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			options   GenerationOptions
			expectErr bool
			wantErr   string
		}{
			{
				name: "valid kotlin generation",
				options: GenerationOptions{
					Platform:   "kotlin",
					OutputPath: "./output",
				},
				expectErr: false,
			},
			{
				name: "unsupported platform",
				options: GenerationOptions{
					Platform:   "unsupported",
					OutputPath: "./output",
				},
				expectErr: true,
				wantErr:   "generating code: unsupported platform: unsupported (supported platforms: kotlin)",
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				mockProvider := &mockPlanProvider{}
				rudderTyper := NewRudderTyper(mockProvider)

				err := rudderTyper.Generate(context.Background(), tt.options)

				if tt.expectErr {
					if err == nil {
						t.Error("Expected error but got none")
						return
					}
					if tt.wantErr != "" && err.Error() != tt.wantErr {
						t.Errorf("Expected error '%s', got '%s'", tt.wantErr, err.Error())
					}
				} else {
					if err != nil {
						t.Errorf("Expected no error but got: %v", err)
					}
				}
			})
		}
	})
}
