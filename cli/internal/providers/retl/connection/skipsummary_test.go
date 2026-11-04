package connection

import (
	"testing"

	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every reason a bulk read drops a row for has to carry a label free of
// row-specific detail: an id or a type leaking into one would put the closing
// summary back to one line per row. A reason that loses its label still counts,
// but under a vaguer line, so pin the labels to the refusals that produce them.
func TestSkipReasonLabelsEveryRefusal(t *testing.T) {
	t.Parallel()

	sources := append(syncBehaviourSources(), remoteSources()...)
	noIdentifiers := syncBehaviourRow("conn-no-identifiers", "dst-1", retlClient.SyncBehaviourUpsert, "")
	noIdentifiers.Identifiers = nil

	for _, tc := range []struct {
		name   string
		conn   retlClient.RETLConnection
		reason string
	}{
		{
			name:   "source missing from the workspace",
			conn:   remoteRow("conn-1", "", "src-gone", "dst-1"),
			reason: "its source is not a rETL source in this workspace",
		},
		{
			name:   "source reporting no workspace",
			conn:   remoteRow("conn-2", "", "src-no-workspace", "dst-1"),
			reason: "its source reports no workspace",
		},
		{
			name:   "destination missing from the workspace",
			conn:   remoteRow("conn-3", "", "src-sf", "dst-gone"),
			reason: "its destination was not found in this workspace",
		},
		{
			name:   "destination version not registered",
			conn:   remoteRow("conn-4", "", "src-sf", "dst-old"),
			reason: "its destination type is not registered in this CLI",
		},
		{
			name:   "destination taking no rETL source",
			conn:   remoteRow("conn-5", "", "src-sf", "dst-eventstream"),
			reason: "its destination does not accept rETL sources",
		},
		{
			name:   "destination on its own flow",
			conn:   remoteRow("conn-6", "", "src-sf", "dst-specific"),
			reason: "its destination runs a flow this CLI does not support",
		},
		{
			name:   "sync behaviour with no spec equivalent",
			conn:   syncBehaviourRow("conn-7", "dst-object", retlClient.SyncBehaviourMirror, "Contact"),
			reason: unrepresentableSyncBehaviour,
		},
		{
			name:   "config with no spec equivalent",
			conn:   noIdentifiers,
			reason: "its configuration has no spec equivalent",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockConnectionClient{Sources: sources, Destinations: remoteDestinations()}
			_, err := remoteHandler(mock, t).remoteConnection(
				tc.conn,
				lo.SliceToMap(sources, func(s retlClient.RETLSource) (string, retlClient.RETLSource) { return s.ID, s }),
				lo.SliceToMap(remoteDestinations(), func(d apiClient.Destination) (string, apiClient.Destination) { return d.ID, d }),
			)
			require.Error(t, err)
			assert.Equal(t, tc.reason, skipReason(err))
		})
	}
}
