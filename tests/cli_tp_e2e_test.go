package tests

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/tests/utils"
)

var testDataRoot = flag.String("testdata.root", "tests/tp", "Root directory for test data relative to project root")

// getToken loads the RUDDERSTACK_ACCESS_TOKEN token from the environment
func getToken(t *testing.T) string {
	token := os.Getenv("RUDDERSTACK_ACCESS_TOKEN")
	if token == "" {
		t.Skip("RUDDERSTACK_ACCESS_TOKEN not set in environment; skipping test")
	}
	return token
}

func TestTrackingPlan_CreateFlow(t *testing.T) {
	dir := filepath.Join(*testDataRoot, "create")
	token := getToken(t)

	// Fetch state before any CLI commands
	stateBefore, err := utils.FetchResourceState(token)
	if err != nil {
		t.Fatalf("failed to fetch resource state: %v", err)
	}

	t.Logf("Starting TestTrackingPlan_CreateFlow for directory: %s", dir)

	t.Run("validate", func(t *testing.T) {
		utils.RunValidate(t, dir, token)
	})
	t.Run("apply_dry_run", func(t *testing.T) {
		utils.RunApplyDryRun(t, dir, token)
	})
	t.Run("apply", func(t *testing.T) {
		utils.RunApply(t, dir, token)
	})

	// Fetch state after apply
	stateAfter, err := utils.FetchResourceState(token)
	if err != nil {
		t.Fatalf("failed to fetch resource state: %v", err)
	}

	t.Run("validate_state", func(t *testing.T) {
		utils.CompareStates(t, stateBefore, stateAfter)
	})
}

func TestTrackingPlan_UpdateFlow(t *testing.T) {
	dir := filepath.Join(*testDataRoot, "update")
	token := getToken(t)

	// Fetch state before any CLI commands
	stateBefore, err := utils.FetchResourceState(token)
	if err != nil {
		t.Fatalf("failed to fetch resource state: %v", err)
	}

	t.Run("validate", func(t *testing.T) {
		utils.RunValidate(t, dir, token)
	})
	t.Run("apply_dry_run", func(t *testing.T) {
		utils.RunApplyDryRun(t, dir, token)
	})
	
	t.Run("apply", func(t *testing.T) {
		utils.RunApply(t, dir, token)
	})

	// Fetch state after apply
	stateAfter, err := utils.FetchResourceState(token)
	if err != nil {
		t.Fatalf("failed to fetch resource state: %v", err)
	}

	t.Run("validate_state", func(t *testing.T) {
		utils.CompareStates(t, stateBefore, stateAfter)
	})
}

func TestTrackingPlan_DeleteFlow(t *testing.T) {
	dir := filepath.Join(*testDataRoot, "delete")
	token := getToken(t)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("%s does not exist; skipping delete flow test", dir)
	}

	// Fetch state before any CLI commands
	stateBefore, err := utils.FetchResourceState(token)
	if err != nil {
		t.Fatalf("failed to fetch resource state: %v", err)
	}

	t.Run("validate", func(t *testing.T) {
		utils.RunValidate(t, dir, token)
	})
	t.Run("apply_dry_run", func(t *testing.T) {
		utils.RunApplyDryRun(t, dir, token)
	})
	t.Run("apply", func(t *testing.T) {
		utils.RunApply(t, dir, token)
	})

	// Fetch state after apply
	stateAfter, err := utils.FetchResourceState(token)
	if err != nil {
		t.Fatalf("failed to fetch resource state: %v", err)
	}

	t.Run("validate_state", func(t *testing.T) {
		utils.CompareStates(t, stateBefore, stateAfter)
	})
}
