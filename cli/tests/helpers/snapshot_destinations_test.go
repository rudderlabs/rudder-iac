package helpers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDestinationLister implements DestinationLister, returning a preset list.
type mockDestinationLister struct {
	destinations []client.Destination
}

func (m *mockDestinationLister) GetAll(context.Context) ([]client.Destination, error) {
	return m.destinations, nil
}

// s3Destination mirrors the destination_s3 snapshot fixture, plus overridable
// volatile fields (id/workspaceId) that the ignore list must drop.
func s3Destination() client.Destination {
	return client.Destination{
		ID:         "srv-generated-id",
		ExternalID: "s3",
		Name:       "My S3",
		Type:       "S3",
		IsEnabled:  true,
		Config: json.RawMessage(`{
			"bucketName": "my-bucket",
			"roleBasedAuth": true,
			"iamRoleARN": "arn:aws:iam::123456789012:role/S3Access"
		}`),
	}
}

var destinationTestIgnore = []string{"id", "workspaceId", "version", "createdAt", "updatedAt"}

func newDestinationTester(t *testing.T, dests []client.Destination) *DestinationSnapshotTester {
	t.Helper()
	fileManager, err := NewSnapshotFileManager("testdata/snapshot/destinations")
	require.NoError(t, err)
	return NewDestinationSnapshotTester(&mockDestinationLister{destinations: dests}, fileManager, destinationTestIgnore)
}

func TestDestinationSnapshotTester(t *testing.T) {
	t.Parallel()

	t.Run("managed destination matches snapshot", func(t *testing.T) {
		t.Parallel()
		tester := newDestinationTester(t, []client.Destination{s3Destination()})
		assert.NoError(t, tester.SnapshotTest(context.Background()))
	})

	t.Run("unmanaged destinations are filtered out by external ID", func(t *testing.T) {
		t.Parallel()
		// An extra destination without an ExternalID (e.g. UI-created) must not
		// count toward the managed set, so the count still matches the one fixture.
		unmanaged := client.Destination{Name: "UI Destination", Type: "S3"}
		tester := newDestinationTester(t, []client.Destination{s3Destination(), unmanaged})
		assert.NoError(t, tester.SnapshotTest(context.Background()))
	})

	t.Run("count mismatch fails", func(t *testing.T) {
		t.Parallel()
		// Two managed destinations but only one expected snapshot file.
		extra := s3Destination()
		extra.ExternalID = "s3-extra"
		tester := newDestinationTester(t, []client.Destination{s3Destination(), extra})
		err := tester.SnapshotTest(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource count mismatch")
	})

	t.Run("comparison mismatch is reported by URN", func(t *testing.T) {
		t.Parallel()
		diverged := s3Destination()
		diverged.Name = "Renamed S3"
		tester := newDestinationTester(t, []client.Destination{diverged})
		err := tester.SnapshotTest(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "destination:s3")
	})
}
