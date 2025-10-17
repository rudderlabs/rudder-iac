package typer

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
)

func TestPrintTrackingPlansTable(t *testing.T) {
	t.Run("empty tracking plans", func(t *testing.T) {
		trackingPlans := []*catalog.TrackingPlanWithIdentifiers{}
		var buf bytes.Buffer
		err := printTrackingPlansTable(&buf, trackingPlans)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Should still have header and separator
		output := buf.String()
		if len(output) == 0 {
			t.Error("expected header output, got empty string")
		}
	})

	t.Run("single tracking plan", func(t *testing.T) {
		desc := "Test description"
		trackingPlans := []*catalog.TrackingPlanWithIdentifiers{
			{
				ID:          "test-id-123",
				Name:        "Test Plan",
				Version:     1,
				Description: &desc,
				WorkspaceID: "workspace-123",
				Events:      []catalog.TrackingPlanEventPropertyIdentifiers{},
			},
		}
		var buf bytes.Buffer
		err := printTrackingPlansTable(&buf, trackingPlans)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := buf.String()
		if !bytes.Contains([]byte(output), []byte("Test Plan")) {
			t.Errorf("expected name 'Test Plan' in output")
		}
		if !bytes.Contains([]byte(output), []byte("test-id-123")) {
			t.Errorf("expected id 'test-id-123' in output")
		}
		if !bytes.Contains([]byte(output), []byte("1")) {
			t.Errorf("expected version '1' in output")
		}
	})

	t.Run("multiple tracking plans", func(t *testing.T) {
		trackingPlans := []*catalog.TrackingPlanWithIdentifiers{
			{
				ID:      "id-1",
				Name:    "Plan 1",
				Version: 1,
				Events:  []catalog.TrackingPlanEventPropertyIdentifiers{},
			},
			{
				ID:      "id-2",
				Name:    "Plan 2",
				Version: 2,
				Events:  []catalog.TrackingPlanEventPropertyIdentifiers{},
			},
		}
		var buf bytes.Buffer
		err := printTrackingPlansTable(&buf, trackingPlans)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := buf.String()
		if !bytes.Contains([]byte(output), []byte("Plan 1")) {
			t.Errorf("expected 'Plan 1' in output")
		}
		if !bytes.Contains([]byte(output), []byte("Plan 2")) {
			t.Errorf("expected 'Plan 2' in output")
		}
	})
}
