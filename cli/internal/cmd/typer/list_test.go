package typer

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
)

func TestPrintTrackingPlansTable(t *testing.T) {
	tests := []struct {
		name          string
		trackingPlans []*catalog.TrackingPlanWithIdentifiers
		expectedTexts []string
		shouldError   bool
	}{
		{
			name:          "empty tracking plans",
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{},
			expectedTexts: []string{"NAME", "ID", "VERSION"},
			shouldError:   false,
		},
		{
			name: "single tracking plan",
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{
					ID:      "test-id-123",
					Name:    "Test Plan",
					Version: 1,
					Events:  nil,
				},
			},
			expectedTexts: []string{"Test Plan", "test-id-123", "1"},
			shouldError:   false,
		},
		{
			name: "multiple tracking plans",
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{
					ID:      "id-1",
					Name:    "Plan 1",
					Version: 1,
					Events:  nil,
				},
				{
					ID:      "id-2",
					Name:    "Plan 2",
					Version: 2,
					Events:  nil,
				},
			},
			expectedTexts: []string{"Plan 1", "Plan 2"},
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := printTrackingPlansTable(&buf, tt.trackingPlans)

			if (err != nil) != tt.shouldError {
				t.Errorf("unexpected error: %v", err)
			}

			output := buf.String()
			if len(output) == 0 {
				t.Error("expected output, got empty string")
			}

			for _, expectedText := range tt.expectedTexts {
				if !bytes.Contains([]byte(output), []byte(expectedText)) {
					t.Errorf("expected '%s' in output", expectedText)
				}
			}
		})
	}
}
