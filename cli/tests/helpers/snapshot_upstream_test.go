package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/stretchr/testify/require"
)

type MockUpstreamAdapter struct {
	event     *catalog.Event
	remoteIDs map[string]string
}

func (m *MockUpstreamAdapter) FetchResource(ctx context.Context, resourceType, resourceID string) (any, error) {
	switch resourceType {
	case state.EventResourceType:
		return m.event, nil
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func (m *MockUpstreamAdapter) RemoteIDs(ctx context.Context) (map[string]string, error) {
	return m.remoteIDs, nil
}

func TestUpstreamSnapshot(t *testing.T) {
	t.Parallel()

	adapter := &MockUpstreamAdapter{
		remoteIDs: map[string]string{
			"event:product_viewed_1": "ca94de47-123b-4dc2-9558-02c57bc289b7",
		},
		event: &catalog.Event{
			ID:          "ca94de47-123b-4dc2-9558-02c57bc289b7",
			Name:        "Product Viewed 1",
			Description: "This event is triggered every time a user views a product.",
			EventType:   "track",
			CategoryId:  strptr("abc"),
			WorkspaceId: "workspace_1",
			CreatedAt:   time.Date(2025, 6, 24, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2025, 6, 24, 0, 0, 0, 0, time.UTC),
		},
	}

	fileManager, err := NewSnapshotFileManager("testdata/snapshot/expected/upstream")
	require.NoError(t, err, "creating state file manager")

	ignoreFields := []string{
		"createdAt",
		"updatedAt",
		"id",
		"workspaceId",
		"categoryId",
	}

	upstreamTester := NewUpstreamSnapshotTester(
		adapter,
		fileManager,
		ignoreFields,
	)

	err = upstreamTester.SnapshotTest(context.Background())
	require.NoError(t, err, "upstream snapshot test")
}

func strptr(str string) *string {
	return &str
}
