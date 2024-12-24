package state

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
)

func TestLocalManager_SaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "localmanager_test_*")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(tmpDir)

	// Cleanup after test
	t.Cleanup(func() {
		// os.RemoveAll(tmpDir)
	})

	manager := &LocalManager{
		BaseDir:   tmpDir,
		StateFile: "test_state.json",
	}

	inputState := &State{
		Version: 1,
		Resources: map[string]*StateResource{
			"test:resource1": {
				ID:   "resource1",
				Type: "test",
				Input: map[string]interface{}{
					"key1": "value1",
					"key2": resources.PropertyRef{URN: "test:resource2", Property: "id"},
				},
				Output: map[string]interface{}{
					"id": "output-resource1",
				},
				Dependencies: []string{"test:resource2"},
			},
			"test:resource2": {
				ID:   "resource2",
				Type: "test",
				Input: map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
				Output: map[string]interface{}{
					"id": "output-resource2",
				},
				Dependencies: []string{},
			},
		},
	}

	// Test Save
	err = manager.Save(context.Background(), inputState)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(path.Join(tmpDir, "test_state.json"))
	assert.NoError(t, err)

	// Test Load
	loadedState, err := manager.Load(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, inputState, loadedState)
}
