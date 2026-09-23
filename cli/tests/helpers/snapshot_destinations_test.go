package helpers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDestinationLister implements DestinationLister, returning a preset list and
// capturing the options the tester asked for.
type mockDestinationLister struct {
	destinations []client.Destination
	opts         client.ListDestinationsOptions
}

func (m *mockDestinationLister) GetAll(_ context.Context, opts ...client.ListDestinationsOption) ([]client.Destination, error) {
	for _, opt := range opts {
		opt(&m.opts)
	}
	return m.destinations, nil
}

// s3Destination mirrors the destination_s3 snapshot fixture, including the
// volatile fields the live API always returns (id/workspaceId/version/timestamps)
// that the ignore list must drop by value. Setting them here means a regression
// where one goes missing (e.g. omitempty drops version:0) is caught without a
// live stack.
func s3Destination() client.Destination {
	var (
		created = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		updated = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	)

	return client.Destination{
		ID:          "srv-generated-id",
		WorkspaceID: "ws-123",
		Version:     3,
		CreatedAt:   &created,
		UpdatedAt:   &updated,
		ExternalID:  "s3",
		Name:        "My S3",
		Type:        "S3",
		IsEnabled:   true,
		Config: json.RawMessage(`{
			"bucketName": "my-bucket",
			"roleBasedAuth": true,
			"iamRoleARN": "arn:aws:iam::123456789012:role/S3Access"
		}`),
	}
}

var destinationTestIgnore = []string{"id", "workspaceId", "version", "createdAt", "updatedAt"}

func newDestinationTester(t *testing.T, dests []client.Destination) (*DestinationSnapshotTester, *mockDestinationLister) {
	t.Helper()
	fileManager, err := NewSnapshotFileManager("testdata/snapshot/destinations")
	require.NoError(t, err)

	lister := &mockDestinationLister{destinations: dests}
	return NewDestinationSnapshotTester(lister, fileManager, destinationTestIgnore), lister
}

func TestDestinationSnapshotTester(t *testing.T) {
	t.Parallel()

	t.Run("managed destination matches snapshot", func(t *testing.T) {
		t.Parallel()
		tester, _ := newDestinationTester(t, []client.Destination{s3Destination()})
		assert.NoError(t, tester.SnapshotTest(context.Background()))
	})

	t.Run("unmanaged destinations are left to the API's hasExternalId filter", func(t *testing.T) {
		t.Parallel()
		tester, lister := newDestinationTester(t, []client.Destination{s3Destination()})
		require.NoError(t, tester.SnapshotTest(context.Background()))
		assert.Equal(t, client.ListDestinationsOptions{HasExternalID: lo.ToPtr(true)}, lister.opts)
	})

	t.Run("count mismatch fails", func(t *testing.T) {
		t.Parallel()
		// Two managed destinations but only one expected snapshot file.
		extra := s3Destination()
		extra.ExternalID = "s3-extra"
		tester, _ := newDestinationTester(t, []client.Destination{s3Destination(), extra})
		err := tester.SnapshotTest(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource count mismatch")
	})

	t.Run("comparison mismatch is reported by URN", func(t *testing.T) {
		t.Parallel()
		diverged := s3Destination()
		diverged.Name = "Renamed S3"
		tester, _ := newDestinationTester(t, []client.Destination{diverged})
		err := tester.SnapshotTest(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "destination:s3")
	})
}
