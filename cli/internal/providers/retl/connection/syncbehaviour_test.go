package connection

import (
	"reflect"
	"strings"
	"testing"

	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptedSyncBehaviours(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name                string
		sourceDefinition    string
		destinationAccepted []string
		flow                Flow
		accepted            []string
		known               bool
	}{
		{
			name:                "intersects both endpoints and drops mirror",
			sourceDefinition:    "snowflake",
			destinationAccepted: []string{"upsert", "mirror", "full"},
			flow:                FlowJSONMapper,
			accepted:            []string{"upsert", "full"},
			known:               true,
		},
		{
			name:                "a mirror-only destination accepts nothing a spec can express",
			sourceDefinition:    "snowflake",
			destinationAccepted: []string{"mirror"},
			flow:                FlowObjectMapping,
			accepted:            []string{},
			known:               true,
		},
		{
			name:                "the source list narrows the result too",
			sourceDefinition:    "s3",
			destinationAccepted: []string{"upsert", "mirror", "full"},
			flow:                FlowJSONMapper,
			accepted:            []string{"upsert"},
			known:               true,
		},
		{
			name:                "an unknown source definition has nothing to intersect",
			sourceDefinition:    "duckdb",
			destinationAccepted: []string{"upsert"},
			flow:                FlowJSONMapper,
			accepted:            nil,
			known:               false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			accepted, known := AcceptedSyncBehaviours(tc.sourceDefinition, tc.destinationAccepted, tc.flow)

			assert.Equal(t, tc.known, known)
			assert.Equal(t, tc.accepted, accepted)
		})
	}
}

// The ConfigSpec.SyncBehaviour `oneof` tag is the spec's own allow-list, and
// AcceptedSyncBehaviours is what the rule and the import path derive their
// answer from. A behaviour the helper hands out that the tag refuses is exactly
// the dead end this gate exists to close — an imported spec that fails validate
// on an immutable field — so pin the two together: whatever the helper can
// return, for any flow, the tag has to accept.
func TestAcceptedSyncBehavioursStayWithinTheSpecTag(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeFor[ConfigSpec]().FieldByName("SyncBehaviour")
	require.True(t, ok)

	var tagged []string
	for _, rule := range strings.Split(field.Tag.Get("validate"), ",") {
		if values, found := strings.CutPrefix(rule, "oneof="); found {
			tagged = strings.Fields(values)
		}
	}
	require.NotEmpty(t, tagged, "ConfigSpec.SyncBehaviour carries no oneof tag to pin against")

	// Every warehouse the CLI knows, offered everything any destination could
	// accept, on every flow: the widest set the helper can ever hand out.
	for _, sourceDefinition := range []string{"bigquery", "clickhouse", "databricks", "mysql", "postgres", "redshift", "s3", "sftp", "snowflake", "trino"} {
		for _, flow := range []Flow{FlowJSONMapper, FlowObjectMapping} {
			accepted, known := AcceptedSyncBehaviours(sourceDefinition, []string{"upsert", "mirror", "full"}, flow)
			require.True(t, known, "source definition %q is in the metadata table", sourceDefinition)
			for _, behaviour := range accepted {
				assert.Contains(t, tagged, behaviour,
					"%s on the %s flow accepts %q, which the spec tag refuses", sourceDefinition, flow, behaviour)
			}
		}
	}
}

// syncBehaviourSources are the endpoint fixtures of this file: the shared ones
// declare no source definition, which leaves the gate inert by design.
func syncBehaviourSources() []retlClient.RETLSource {
	return []retlClient.RETLSource{
		{ID: "src-sf", Name: "Users", SourceType: retlClient.ModelSourceType, SourceDefinitionName: "snowflake", WorkspaceID: "ws-1"},
	}
}

func syncBehaviourRow(id, destinationID string, behaviour retlClient.SyncBehaviour, object string) retlClient.RETLConnection {
	conn := remoteRow(id, "", "src-sf", destinationID)
	conn.SyncBehaviour = behaviour
	conn.Object = object
	return conn
}

// Import eligibility has to refuse exactly what the spec rules refuse. A remote
// mirror connection used to pass eligibility and land in a generated spec that
// then failed validate on an immutable field, leaving the user no edit out.
func TestEligibleSkipsUnrepresentableSyncBehaviour(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		conn    retlClient.RETLConnection
		kept    bool
		message string
	}{
		{
			name: "mirror on an object mapping destination is skipped",
			conn: syncBehaviourRow("conn-mirror-object", "dst-object", retlClient.SyncBehaviourMirror, "Contact"),
			message: `connection "conn-mirror-object": sync behaviour "mirror" cannot be expressed as a spec, ` +
				`and source definition "snowflake" and this destination share no behaviour that can on the object_mapping flow; ` +
				"the connection stays in the workspace unchanged: connection config cannot be represented as a spec",
		},
		{
			name: "mirror on a destination with expressible behaviours names them",
			conn: syncBehaviourRow("conn-mirror-json", "dst-1", retlClient.SyncBehaviourMirror, ""),
			message: `connection "conn-mirror-json": sync behaviour "mirror" cannot be expressed as a spec on the json_mapper flow, ` +
				"where only [upsert full] can; the connection stays in the workspace unchanged: connection config cannot be represented as a spec",
		},
		{
			name: "an expressible behaviour is kept",
			conn: syncBehaviourRow("conn-upsert", "dst-1", retlClient.SyncBehaviourUpsert, ""),
			kept: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockConnectionClient{
				Sources:      syncBehaviourSources(),
				Destinations: remoteDestinations(),
				ListFunc:     pagedList([]retlClient.RETLConnection{tc.conn}),
			}

			remotes, err := remoteHandler(mock, t).eligible(t.Context(), lo.ToPtr(false))
			require.NoError(t, err)

			if tc.kept {
				require.Len(t, remotes, 1)
				assert.Equal(t, tc.conn.ID, remotes[0].ID)
				return
			}
			assert.Empty(t, remotes, "the row has to be skipped, not written into a spec that fails validate")

			// The skip message is what the user reads, so assert it whole.
			_, skipErr := remoteHandler(mock, t).remoteConnection(
				tc.conn,
				lo.SliceToMap(syncBehaviourSources(), func(s retlClient.RETLSource) (string, retlClient.RETLSource) { return s.ID, s }),
				lo.SliceToMap(remoteDestinations(), func(d apiClient.Destination) (string, apiClient.Destination) { return d.ID, d }),
			)
			require.Error(t, skipErr)
			assert.EqualError(t, skipErr, tc.message)
			assert.ErrorIs(t, skipErr, ErrUnrepresentableConfig)
		})
	}
}

// A source definition the CLI has no metadata for leaves the gate inert: the
// source's own rules report the definition, and guessing a set here would skip
// rows that are fine.
func TestEligibleKeepsRowsWithUnknownSourceDefinition(t *testing.T) {
	t.Parallel()

	conn := syncBehaviourRow("conn-unknown-def", "dst-1", retlClient.SyncBehaviourMirror, "")
	sources := []retlClient.RETLSource{
		{ID: "src-sf", Name: "Users", SourceType: retlClient.ModelSourceType, SourceDefinitionName: "duckdb", WorkspaceID: "ws-1"},
	}
	mock := &MockConnectionClient{
		Sources:      sources,
		Destinations: remoteDestinations(),
		ListFunc:     pagedList([]retlClient.RETLConnection{conn}),
	}

	remotes, err := remoteHandler(mock, t).eligible(t.Context(), lo.ToPtr(false))
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	assert.Equal(t, "mirror", remotes[0].Config.SyncBehaviour)
}
