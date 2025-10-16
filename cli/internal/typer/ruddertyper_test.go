package typer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{
					Properties:           make(map[string]plan.PropertySchema),
					AdditionalProperties: false,
				},
			},
		},
	}, nil
}

func TestRudderTyper_Generate(t *testing.T) {
	t.Run("Generate (table)", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			options   core.GenerationOptions
			expectErr bool
			wantErr   string
		}{
			{
				name: "valid kotlin generation",
				options: core.GenerationOptions{
					Platform:   "kotlin",
					OutputPath: "./output",
				},
				expectErr: false,
			},
			{
				name: "unsupported platform",
				options: core.GenerationOptions{
					Platform:   "unsupported",
					OutputPath: "./output",
				},
				expectErr: true,
				wantErr:   "generating code: unsupported platform: unsupported (supported platforms: kotlin)",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				mockProvider := &mockPlanProvider{}
				rudderTyper := NewRudderTyper(mockProvider)

				// Create a temporary directory for output
				tempDir := t.TempDir()
				tt.options.OutputPath = tempDir

				err := rudderTyper.Generate(context.Background(), tt.options)

				if tt.expectErr {
					require.Error(t, err)
					if tt.wantErr != "" {
						assert.Contains(t, err.Error(), tt.wantErr)
					}
				} else {
					require.NoError(t, err)

					// Verify that expected files were created
					if tt.options.Platform == "kotlin" {
						expectedFile := filepath.Join(tempDir, "Main.kt")
						assert.FileExists(t, expectedFile, "Expected Main.kt file to be created")

						// Verify the file is not empty
						fileInfo, err := os.Stat(expectedFile)
						require.NoError(t, err)
						assert.Greater(t, fileInfo.Size(), int64(0), "Generated file should not be empty")
					}
				}
			})
		}
	})
}
