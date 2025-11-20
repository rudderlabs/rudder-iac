package helpers

import (
	"context"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/stretchr/testify/require"
)

type MockDataCatalogClient struct {
	datacatalog.EmptyCatalog
	event *catalog.Event
}

func (m *MockDataCatalogClient) SetEvent(event *catalog.Event) {
	m.event = event
}

func (m *MockDataCatalogClient) GetEvent(ctx context.Context, id string) (*catalog.Event, error) {
	return m.event, nil
}

type MockUpstreamStateReader struct {
	remoteIDs map[string]string
}

func (m *MockUpstreamStateReader) RemoteIDs(ctx context.Context) (map[string]string, error) {
	return m.remoteIDs, nil
}

func TestUpstreamSnapshot(t *testing.T) {
	t.Parallel()

	mockState := map[string]string{
		"event:product_viewed_1": "ca94de47-123b-4dc2-9558-02c57bc289b7",
	}
	reader := &MockUpstreamStateReader{remoteIDs: mockState}

	fileManager, err := NewSnapshotFileManager("testdata/snapshot/expected/upstream")
	require.NoError(t, err, "creating state file manager")

	ignoreFields := []string{
		"createdAt",
		"updatedAt",
		"id",
		"workspaceId",
		"categoryId",
	}

	catalogClient := &MockDataCatalogClient{}
	catalogClient.SetEvent(&catalog.Event{
		ID:          "ca94de47-123b-4dc2-9558-02c57bc289b7",
		Name:        "Product Viewed 1",
		Description: "This event is triggered every time a user views a product.",
		EventType:   "track",
		CategoryId:  strptr("abc"),
		WorkspaceId: "workspace_1",
		CreatedAt:   time.Date(2025, 6, 24, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 6, 24, 0, 0, 0, 0, time.UTC),
	})
	upstreamTester := NewUpstreamSnapshotTester(
		catalogClient,
		reader,
		fileManager,
		ignoreFields,
	)

	err = upstreamTester.SnapshotTest(context.Background())
	require.NoError(t, err, "upstream snapshot test")
}

func strptr(str string) *string {
	return &str
}
