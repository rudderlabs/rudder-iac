package utils

import (
	"os/exec"
	"testing"
)

// SetCmdEnv sets the RUDDERSTACK_ACCESS_TOKEN env for the command
func SetCmdEnv(cmd *exec.Cmd, token string) {
	cmd.Env = append(cmd.Env, "RUDDERSTACK_ACCESS_TOKEN="+token)
}

func RunValidate(t *testing.T, dir, token string) {
	cmd := exec.Command("rudder-cli", "tp", "validate", "-l", dir)
	cmd.Dir = ".."
	SetCmdEnv(cmd, token)
	t.Logf("Running validate command: %v", cmd.Args)
	out, err := cmd.CombinedOutput()
	t.Logf("Output:\n%s", out)
	if err != nil {
		t.Fatalf("validate failed: %v\nOutput: %s", err, out)
	}
}

func RunApplyDryRun(t *testing.T, dir, token string) {
	cmd := exec.Command("rudder-cli", "tp", "apply", "-l", dir, "--dry-run")
	cmd.Dir = ".."
	SetCmdEnv(cmd, token)
	t.Logf("Running: %v", cmd.Args)
	out, err := cmd.CombinedOutput()
	t.Logf("Output:\n%s", out)
	if err != nil {
		t.Fatalf("apply --dry-run failed: %v\nOutput: %s", err, out)
	}
}

func RunApply(t *testing.T, dir, token string) {
	cmd := exec.Command("rudder-cli", "tp", "apply", "-l", dir, "--confirm=false")
	cmd.Dir = ".."
	SetCmdEnv(cmd, token)
	t.Logf("Running: %v", cmd.Args)
	out, err := cmd.CombinedOutput()
	t.Logf("Output:\n%s", out)
	if err != nil {
		t.Fatalf("apply failed: %v\nOutput: %s", err, out)
	}
}