package tests

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestK8sVerbs_GetApplyRoundTrip_ScopedNoDelete proves the new kubectl-style verbs
// end-to-end on the event-stream-source type:
//
//  1. Creates two sources (A and B) via apply.
//  2. get event-stream-source <A-id> -o yaml → round-trip: apply --dry-run reports no changes.
//  3. Modify A's display name → apply -f reports an update; B is untouched.
//  4. Cleanup via delete.
func TestK8sVerbs_GetApplyRoundTrip_ScopedNoDelete(t *testing.T) {
	// The resource verbs (get/describe/delete/set-external-id, apply -f) are
	// gated behind the experimental `resourceCommands` flag; enable it for the
	// child rudder-cli processes this test spawns.
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_RESOURCE_COMMANDS", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var (
		aID   = fmt.Sprintf("k8se2e-a-%s", suffix)
		bID   = fmt.Sprintf("k8se2e-b-%s", suffix)
		aName = fmt.Sprintf("K8s E2E Source A %s", suffix)
		bName = fmt.Sprintf("K8s E2E Source B %s", suffix)
	)

	// ─── Step 1: Create both sources ─────────────────────────────────────────

	setupDir := t.TempDir()

	aSpec := buildSourceSpec(aID, aName, "javascript")
	bSpec := buildSourceSpec(bID, bName, "node")

	require.NoError(t, writeFile(filepath.Join(setupDir, "a.yaml"), aSpec))
	require.NoError(t, writeFile(filepath.Join(setupDir, "b.yaml"), bSpec))

	out, err := executor.Execute(cliBinPath, "apply", "-l", setupDir, "--confirm=false")
	require.NoError(t, err, "initial apply failed: %s", string(out))

	// Register cleanup BEFORE any step that could fail so we always attempt deletion.
	t.Cleanup(func() {
		delA, errA := executor.Execute(cliBinPath, "delete", "event-stream-source", aID, "--confirm")
		if errA != nil {
			t.Logf("cleanup: delete source A (%s) failed (may already be gone): %v\n%s", aID, errA, string(delA))
		}
		delB, errB := executor.Execute(cliBinPath, "delete", "event-stream-source", bID, "--confirm")
		if errB != nil {
			t.Logf("cleanup: delete source B (%s) failed (may already be gone): %v\n%s", bID, errB, string(delB))
		}
	})

	// Verify the apply created the sources by listing them.
	listOut, err := executor.Execute(cliBinPath, "get", "event-stream-source")
	require.NoError(t, err, "get list failed: %s", string(listOut))
	assert.Contains(t, string(listOut), aID, "source A should appear in list after create")
	assert.Contains(t, string(listOut), bID, "source B should appear in list after create")

	// ─── Step 2: Capture A's YAML spec (stdout-only) ─────────────────────────

	// We MUST capture stdout only — combined output would mix log noise into YAML.
	yamlBytes, err := captureStdout(cliBinPath, "get", "event-stream-source", aID, "-o", "yaml")
	require.NoError(t, err, "get -o yaml failed for source A (%s)", aID)
	require.NotEmpty(t, yamlBytes, "get -o yaml must produce non-empty output")

	// Sanity-check: must parse as a valid spec.
	capturedSpec, err := specs.New(yamlBytes)
	require.NoError(t, err, "captured YAML is not a valid spec:\n%s", string(yamlBytes))
	assert.Equal(t, "event-stream-source", capturedSpec.Kind, "kind must be event-stream-source")
	assert.Equal(t, "rudder/v1", capturedSpec.Version, "version must be rudder/v1")
	assert.Equal(t, aID, capturedSpec.Spec["id"], "spec.id must match the source's external ID")

	tmpDir := t.TempDir()
	aYAMLPath := filepath.Join(tmpDir, "a.yaml")
	require.NoError(t, writeFile(aYAMLPath, string(yamlBytes)))

	// ─── Step 3: Round-trip dry-run — must report no changes ─────────────────

	roundTripOut, err := executor.Execute(
		cliBinPath, "apply", "-f", aYAMLPath, "--dry-run", "--confirm=false",
	)
	require.NoError(t, err, "round-trip dry-run failed: %s", string(roundTripOut))
	require.Contains(t, string(roundTripOut), "No changes to apply",
		"round-trip dry-run must report no changes; got:\n%s", string(roundTripOut))

	// ─── Step 4: Scoped update of A, assert B is untouched ───────────────────

	// Modify A's display name in the captured YAML.
	modifiedName := fmt.Sprintf("K8s E2E Source A Modified %s", suffix)
	modifiedYAML, err := modifySpecName(yamlBytes, modifiedName)
	require.NoError(t, err, "failed to modify spec name")

	aModPath := filepath.Join(tmpDir, "a-mod.yaml")
	require.NoError(t, writeFile(aModPath, string(modifiedYAML)))

	updateOut, err := executor.Execute(cliBinPath, "apply", "-f", aModPath, "--confirm=false")
	require.NoError(t, err, "scoped update apply failed: %s", string(updateOut))

	// The plan reporter prints "Updated resources:" when there are updates.
	assert.Contains(t, string(updateOut), "Updated resources",
		"update apply must report updated resources; got:\n%s", string(updateOut))

	// Verify A was actually updated by fetching it (stdout-only).
	aAfterBytes, err := captureStdout(cliBinPath, "get", "event-stream-source", aID, "-o", "yaml")
	require.NoError(t, err, "get -o yaml for A after update failed")

	aAfterSpec, err := specs.New(aAfterBytes)
	require.NoError(t, err, "A's post-update YAML is not a valid spec")
	assert.Equal(t, modifiedName, aAfterSpec.Spec["name"],
		"A's name must reflect the update; got spec:\n%s", string(aAfterBytes))

	// Assert B is untouched: fetch B and verify it still has its original name.
	bAfterBytes, err := captureStdout(cliBinPath, "get", "event-stream-source", bID, "-o", "yaml")
	require.NoError(t, err, "get -o yaml for B after A-only update failed")
	require.NotEmpty(t, bAfterBytes, "source B must still exist after scoped apply of A")

	bAfterSpec, err := specs.New(bAfterBytes)
	require.NoError(t, err, "B's YAML after A-only apply is not a valid spec")
	assert.Equal(t, bID, bAfterSpec.Spec["id"],
		"B's external ID must be unchanged; got:\n%s", string(bAfterBytes))
	assert.Equal(t, bName, bAfterSpec.Spec["name"],
		"B's display name must be unchanged after scoped apply of A only; got:\n%s", string(bAfterBytes))

	// Cleanup is handled by t.Cleanup registered above.
}

// captureStdout runs cliBinPath with args and returns STDOUT only.
// This is critical for YAML capture — CombinedOutput would mix log noise into
// the YAML document and break spec parsing.
func captureStdout(binPath string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command %q failed: %w\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// buildSourceSpec returns a rudder/v1 event-stream-source YAML spec string.
func buildSourceSpec(id, name, sourceType string) string {
	return fmt.Sprintf(`version: "rudder/v1"
kind: "event-stream-source"
metadata:
  name: "event-stream-source"
spec:
  id: "%s"
  name: "%s"
  type: "%s"
`, id, name, sourceType)
}

// writeFile writes content to path, creating any necessary parent directories.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating parent dirs: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// modifySpecName parses the YAML bytes, changes spec.name to newName, and
// re-serialises to YAML. This preserves the full metadata (workspace import
// info, remote ID) so the apply knows it is an update rather than a create.
func modifySpecName(yamlBytes []byte, newName string) ([]byte, error) {
	// Use a generic map to preserve the full structure without a typed decode.
	var doc map[string]any
	if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
		return nil, fmt.Errorf("unmarshaling captured YAML: %w", err)
	}

	specSection, ok := doc["spec"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("spec section missing or wrong type in:\n%s", string(yamlBytes))
	}
	specSection["name"] = newName
	doc["spec"] = specSection

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling modified spec: %w", err)
	}
	return out, nil
}
