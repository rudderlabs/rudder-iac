package typer

import (
	"context"
	"testing"
)

func TestNewOrchestrator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "constructor with nil deps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orchestrator := NewOrchestrator(nil)
			if orchestrator == nil {
				t.Fatal("NewOrchestrator() returned nil")
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
				TrackingPlanID: "tp_123",
				Platform:       "kotlin",
				OutputPath:     "./output",
			},
		},
		{
			name: "different values",
			options: GenerationOptions{
				TrackingPlanID: "another_tp",
				Platform:       "kotlin",
				OutputPath:     "/tmp/out",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.options.TrackingPlanID == "" {
				t.Error("TrackingPlanID should not be empty")
			}
			if tt.options.Platform != "kotlin" {
				t.Errorf("expected Platform 'kotlin', got %s", tt.options.Platform)
			}
			if tt.options.OutputPath == "" {
				t.Error("OutputPath should not be empty")
			}
		})
	}
}

func TestOrchestrator_Generate(t *testing.T) {
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
					TrackingPlanID: "tp_123",
					Platform:       "kotlin",
					OutputPath:     "./output",
				},
				expectErr: false,
			},
			{
				name: "unsupported platform",
				options: GenerationOptions{
					TrackingPlanID: "tp_123",
					Platform:       "unsupported",
					OutputPath:     "./output",
				},
				expectErr: true,
				wantErr:   "generating code: unsupported platform: unsupported (supported platforms: kotlin)",
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				orchestrator := NewOrchestrator(nil)

				err := orchestrator.Generate(context.Background(), tt.options)

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
