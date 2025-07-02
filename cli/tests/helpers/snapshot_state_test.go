package helpers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockUpstreamStateReader struct {
	state map[string]any
}

func (m *MockUpstreamStateReader) RawState(ctx context.Context) (map[string]any, error) {
	return m.state, nil
}

func TestSnapshotState(t *testing.T) {
	t.Parallel()

	mockState := map[string]any{
		"version": "1.0.0",
		"resources": map[string]any{
			"event_product_viewed_1": map[string]any{
				"id":   "product_viewed_1",
				"type": "event",
				"input": map[string]any{
					"description": "This event is triggered every time a user views a product.",
					"eventType":   "track",
					"name":        "Product Viewed 1",
					"categoryId":  "dynamic-category-id", // This should be ignored
				},
				"output": map[string]any{
					"description": "This event is triggered every time a user views a product.",
					"eventArgs": map[string]any{
						"description": "This event is triggered every time a user views a product.",
						"eventType":   "track",
						"name":        "Product Viewed 1",
						"categoryId":  "dynamic-category-id", // This should be ignored
					},
					"eventType":   "track",
					"name":        "Product Viewed 1",
					"createdAt":   "2024-01-01T00:00:00Z", // This should be ignored
					"updatedAt":   "2024-01-01T00:00:00Z", // This should be ignored
					"id":          "dynamic-id",           // This should be ignored
					"workspaceId": "dynamic-workspace-id", // This should be ignored
					"categoryId":  "dynamic-category-id",  // This should be ignored
				},
				"dependencies": []any{},
			},
		},
	}

	mockReader := &MockUpstreamStateReader{state: mockState}

	fileManager, err := NewSnapshotFileManager("testdata/snapshot/expected/state")
	require.NoError(t, err, "creating state file manager")

	ignoreFields := []string{
		"output.createdAt",
		"output.updatedAt",
		"output.id",
		"output.workspaceId",
		"input.categoryId",
		"output.categoryId",
		"output.eventArgs.categoryId",
	}

	tester := NewStateSnapshotTester(
		mockReader,
		fileManager,
		ignoreFields,
	)
	err = tester.SnapshotTest(context.Background())
	require.NoError(t, err, "state snapshot test")
}
