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
			// Bing Ads accepts mirror alone, so on object mapping the endpoints
			// share nothing a spec can express. Recreating cannot help here, so
			// the message says the row is left alone instead of naming a remedy
			// that would cost the user a working connection for nothing.
			name: "mirror on an object mapping destination is skipped",
			conn: syncBehaviourRow("conn-mirror-object", "dst-object", retlClient.SyncBehaviourMirror, "Contact"),
			message: `connection "conn-mirror-object": source definition "snowflake" and this destination share no sync behaviour ` +
				"a spec can express on the object_mapping flow; the connection stays in the workspace unchanged: connection config cannot be represented as a spec",
		},
		{
			// A webhook accepts all three, so recreating is a real remedy here
			// and the message names the behaviours that would work.
			name: "mirror on a json mapper destination is skipped",
			conn: syncBehaviourRow("conn-mirror-json", "dst-1", retlClient.SyncBehaviourMirror, ""),
			message: `connection "conn-mirror-json": sync behaviour "mirror" cannot be expressed as a spec on the json_mapper flow; ` +
				"recreate the connection with one of [upsert full] to bring it under the CLI: connection config cannot be represented as a spec",
		},
		{
			// The same pair as the first case with a behaviour a spec can
			// carry. It reads identically, and should: what cannot be managed
			// is the pair, not the behaviour the row happens to hold.
			name: "a behaviour the endpoints do not share is skipped",
			conn: syncBehaviourRow("conn-upsert-object", "dst-object", retlClient.SyncBehaviourUpsert, "Contact"),
			message: `connection "conn-upsert-object": source definition "snowflake" and this destination share no sync behaviour ` +
				"a spec can express on the object_mapping flow; the connection stays in the workspace unchanged: connection config cannot be represented as a spec",
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

// The gate must never touch a managed row. LoadResourcesFromRemote builds the
// state graph, and a row dropped from it is a row the planner cannot see: the
// spec beside it reads as a new resource, so apply plans a Create against a
// source-destination pair that already holds a connection, and removing the
// spec plans no Delete at all. The spec still cannot express mirror and V-R2
// still refuses it at validate, which is where the user can read it.
func TestManagedMirrorConnectionStaysInState(t *testing.T) {
	t.Parallel()

	managed := syncBehaviourRow("conn-mirror-managed", "dst-object", retlClient.SyncBehaviourMirror, "Contact")
	managed.ExternalID = "users-to-bing-ads"

	mock := &MockConnectionClient{
		Sources:      syncBehaviourSources(),
		Destinations: remoteDestinations(),
		ListFunc:     pagedList([]retlClient.RETLConnection{managed}),
	}

	collection, err := remoteHandler(mock, t).LoadResourcesFromRemote(t.Context())
	require.NoError(t, err)

	rows := collection.GetAll(ResourceType)
	require.Len(t, rows, 1, "a managed row must stay in state whatever its sync behaviour")
	require.Contains(t, rows, managed.ID)
	assert.Equal(t, "users-to-bing-ads", rows[managed.ID].ExternalID)

	// The same row, unmanaged, is still refused: this is the import candidate.
	unmanaged := managed
	unmanaged.ExternalID = ""
	require.Error(t, syncBehaviourRepresentable(unmanaged, "snowflake", []string{"mirror"}, FlowObjectMapping))
	require.NoError(t, syncBehaviourRepresentable(managed, "snowflake", []string{"mirror"}, FlowObjectMapping))
}

// A source definition the CLI has no metadata for leaves the endpoint
// intersection inert — guessing a set there would skip rows that are fine — but
// not the spec check: a behaviour no spec can carry is a dead end on any
// source, and importing it would write a spec that fails validate on an
// immutable field.
func TestUnknownSourceDefinitionGatesOnTheSpecContractOnly(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		behaviour retlClient.SyncBehaviour
		kept      bool
	}{
		{name: "an expressible behaviour is kept", behaviour: retlClient.SyncBehaviourUpsert, kept: true},
		{name: "mirror is still refused", behaviour: retlClient.SyncBehaviourMirror},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conn := syncBehaviourRow("conn-unknown-def", "dst-1", tc.behaviour, "")
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

			if !tc.kept {
				assert.Empty(t, remotes, "a behaviour no spec can carry is a dead end whatever the source definition")
				return
			}
			require.Len(t, remotes, 1)
			assert.Equal(t, string(tc.behaviour), remotes[0].Config.SyncBehaviour)
		})
	}
}

// An empty source definition name takes the unknown path too, so the same
// split applies: the endpoint intersection is skipped, the spec check is not.
func TestEmptySourceDefinitionGatesOnTheSpecContractOnly(t *testing.T) {
	t.Parallel()

	row := func(behaviour retlClient.SyncBehaviour) retlClient.RETLConnection {
		return syncBehaviourRow("conn-empty-def", "dst-1", behaviour, "")
	}
	require.NoError(t, syncBehaviourRepresentable(row(retlClient.SyncBehaviourUpsert), "", []string{"upsert", "full"}, FlowJSONMapper))
	require.Error(t, syncBehaviourRepresentable(row(retlClient.SyncBehaviourMirror), "", []string{"mirror"}, FlowJSONMapper))
}

// The endpoints can also share some expressible behaviour without sharing the
// one this row carries, which reads differently from sharing none: the message
// names what is left. No fixture destination produces the pair, so drive the
// gate directly.
func TestSyncBehaviourNotSharedNamesWhatIsLeft(t *testing.T) {
	t.Parallel()

	conn := syncBehaviourRow("conn-full", "dst-1", retlClient.SyncBehaviourFull, "")
	err := syncBehaviourRepresentable(conn, "snowflake", []string{"upsert"}, FlowJSONMapper)
	require.Error(t, err)
	assert.EqualError(t, err, `connection "conn-full": sync behaviour "full" is not accepted by source definition "snowflake" `+
		"and this destination together, which share only [upsert] on the json_mapper flow; "+
		"the connection stays in the workspace unchanged: connection config cannot be represented as a spec")
	assert.ErrorIs(t, err, ErrUnrepresentableConfig)
}
