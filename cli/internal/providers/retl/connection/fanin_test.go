package connection

import (
	"context"
	"errors"
	"testing"

	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckFanIn(t *testing.T) {
	conn := func(id, src, dst string, opts ...resources.ResourceOpts) *resources.Resource {
		return resources.NewResource(id, ResourceType, resources.ResourceData{
			SourceKey:      &resources.PropertyRef{URN: "retl-source-sql-model:" + src, Property: "id"},
			DestinationKey: &resources.PropertyRef{URN: destination.DestinationResourceType + ":" + dst, Property: "id"},
		}, nil, opts...)
	}
	st := state.EmptyState()
	st.AddResource(&state.ResourceState{ID: "src", Type: "retl-source-sql-model", Output: map[string]any{"id": "remote-src"}})
	st.AddResource(&state.ResourceState{ID: "dst", Type: destination.DestinationResourceType, OutputRaw: &destination.DestinationState{ID: "remote-dst"}})
	st.AddResource(&state.ResourceState{ID: "other", Type: "retl-source-sql-model", Output: map[string]any{"id": "remote-other"}})
	st.AddResource(&state.ResourceState{ID: "third", Type: "retl-source-sql-model", Output: map[string]any{"id": "remote-third"}})
	st.AddResource(&state.ResourceState{ID: "managed", Type: ResourceType, Output: map[string]any{"id": "existing"}})
	row := retlClient.RETLConnection{ID: "existing", SourceID: "remote-src", DestinationID: "remote-dst"}
	tests := []struct {
		name string
		ops  []*planner.Operation
		rows []retlClient.RETLConnection
		want string
	}{
		{"unchanged import into inherited fan-in", []*planner.Operation{{Type: planner.Import, Resource: conn("imported", "src", "dst", resources.WithResourceImportMetadata("existing", "ws-1"))}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-other", DestinationID: "remote-dst"}}, ""},
		{"import endpoint replacement refuses fan-in", []*planner.Operation{{Type: planner.Import, Resource: conn("imported", "other", "dst", resources.WithResourceImportMetadata("existing", "ws-1"))}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-src", DestinationID: "remote-dst"}}, "already has rETL connection"},
		{"update source repoint refuses fan-in despite same connection ID", []*planner.Operation{{Type: planner.Update, Resource: conn("managed", "other", "dst")}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-src", DestinationID: "remote-dst"}}, "already has rETL connection"},
		{"new connection refuses unmanaged occupant", []*planner.Operation{{Type: planner.Create, Resource: conn("new", "other", "dst")}}, []retlClient.RETLConnection{row}, "already has rETL connection"},
		{"unchanged update does not reject inherited fan-in", []*planner.Operation{{Type: planner.Update, Resource: conn("managed", "src", "dst")}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-other", DestinationID: "remote-dst"}}, ""},
		{"same-source replacement collides with occupied pair", []*planner.Operation{{Type: planner.Update, Resource: conn("managed", "other", "dst")}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-other", DestinationID: "remote-dst"}}, "already has rETL connection"},
		{"full inherited group with unmodified peer and no changes", []*planner.Operation{{Type: planner.Update, Resource: conn("managed", "src", "dst")}, {Type: planner.Import, Resource: conn("imported", "other", "dst", resources.WithResourceImportMetadata("foreign", "ws-1"))}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-other", DestinationID: "remote-dst"}}, ""},
		{"unchanged peer checked against changed write", []*planner.Operation{{Type: planner.Update, Resource: conn("managed", "other", "dst")}, {Type: planner.Import, Resource: conn("imported", "src", "dst", resources.WithResourceImportMetadata("foreign", "ws-1"))}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-src", DestinationID: "remote-dst"}}, "already has rETL connection"},
		{"three-row fan-in refuses new source on repoint", []*planner.Operation{{Type: planner.Update, Resource: conn("managed", "third", "dst")}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-other", DestinationID: "remote-dst"}, {ID: "peer", SourceID: "remote-src", DestinationID: "remote-dst"}}, "already has rETL connection"},
		{"three-row fan-in refuses occupied pair despite same source", []*planner.Operation{{Type: planner.Update, Resource: conn("managed", "third", "dst")}}, []retlClient.RETLConnection{row, {ID: "foreign", SourceID: "remote-third", DestinationID: "remote-dst"}, {ID: "peer", SourceID: "remote-third", DestinationID: "remote-dst"}}, "already has rETL connection"},
		{"two planned writes to a new destination", []*planner.Operation{{Type: planner.Create, Resource: resources.NewResource("fresh", destination.DestinationResourceType, nil, nil)}, {Type: planner.Create, Resource: conn("one", "src", "fresh")}, {Type: planner.Create, Resource: conn("two", "other", "fresh")}}, nil, "multiple planned sources"},
		{"missing imported row fails closed", []*planner.Operation{{Type: planner.Import, Resource: conn("imported", "src", "dst", resources.WithResourceImportMetadata("absent", "ws-1"))}}, nil, "was not found"},
		{"new endpoint not in plan fails closed", []*planner.Operation{{Type: planner.Create, Resource: conn("new", "src", "missing")}}, nil, "no remote id or planned create"},
		{"new destination created in plan", []*planner.Operation{{Type: planner.Create, Resource: resources.NewResource("fresh", destination.DestinationResourceType, nil, nil)}, {Type: planner.Create, Resource: conn("new", "src", "fresh")}}, nil, ""},
		{"imported destination refuses a different source", []*planner.Operation{{Type: planner.Import, Resource: resources.NewResource("fresh", destination.DestinationResourceType, nil, nil, resources.WithResourceImportMetadata("remote-dst", "ws-1"))}, {Type: planner.Create, Resource: conn("new", "other", "fresh")}}, []retlClient.RETLConnection{row}, "already has rETL connection"},
		{"imported destination allows existing source", []*planner.Operation{{Type: planner.Import, Resource: resources.NewResource("fresh", destination.DestinationResourceType, nil, nil, resources.WithResourceImportMetadata("remote-dst", "ws-1"))}, {Type: planner.Create, Resource: conn("new", "src", "fresh")}}, []retlClient.RETLConnection{row}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockConnectionClient{
				Sources:      []retlClient.RETLSource{{ID: "remote-src"}, {ID: "remote-other"}, {ID: "remote-third"}},
				Destinations: []apiClient.Destination{{ID: "remote-dst"}},
				ListFunc: func(_ *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
					return &retlClient.RETLConnectionsPage{Data: tt.rows}, nil
				}}
			err := lifecycleHandler(t, mock).CheckFanIn(context.Background(), &planner.Plan{Operations: tt.ops}, st)
			if tt.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.want)
			}
			assert.Len(t, mock.ListCalls, 1)
			assert.Nil(t, mock.ListCalls[0].HasExternalID, "include unmanaged remote rows")
			assert.Empty(t, mock.DeleteCalls)
		})
	}
	t.Run("list failure prevents any mutation", func(t *testing.T) {
		mock := &MockConnectionClient{ListFunc: func(_ *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
			return nil, errors.New("network down")
		}}
		err := lifecycleHandler(t, mock).CheckFanIn(context.Background(), &planner.Plan{Operations: []*planner.Operation{{Type: planner.Create, Resource: conn("new", "src", "dst")}}}, st)
		require.ErrorContains(t, err, "network down")
		assert.Empty(t, mock.CreateCalls)
	})
	t.Run("endpoint catalog failure prevents any mutation", func(t *testing.T) {
		mock := &MockConnectionClient{EndpointsErr: errors.New("catalog unavailable")}
		err := lifecycleHandler(t, mock).CheckFanIn(context.Background(), &planner.Plan{Operations: []*planner.Operation{{Type: planner.Create, Resource: conn("new", "src", "dst")}}}, st)
		require.ErrorContains(t, err, "catalog unavailable")
		assert.Empty(t, mock.CreateCalls)
	})
	t.Run("missing remote endpoint fails closed", func(t *testing.T) {
		mock := &MockConnectionClient{Sources: []retlClient.RETLSource{{ID: "remote-src"}}}
		err := lifecycleHandler(t, mock).CheckFanIn(context.Background(), &planner.Plan{Operations: []*planner.Operation{{Type: planner.Create, Resource: conn("new", "src", "dst")}}}, st)
		require.ErrorContains(t, err, "was not found in this workspace")
		assert.Empty(t, mock.CreateCalls)
	})
}

func TestReplacementRefusesOccupiedPairBeforeDelete(t *testing.T) {
	mock := lifecycleClient()
	mock.ListFunc = func(_ *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
		return &retlClient.RETLConnectionsPage{Data: []retlClient.RETLConnection{{ID: "peer", SourceID: "src-1", DestinationID: "dst-1"}}}, nil
	}
	prior := stateData(t, jsonMapperConfig())
	prior[SourceIDKey] = "old-source"
	_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, graphData(t, jsonMapperConfig()), prior)
	require.ErrorContains(t, err, "refusing to delete")
	assert.Empty(t, mock.DeleteCalls)
	assert.Empty(t, mock.CreateCalls)
}

func TestReplacementRefusesFanInBeforeDelete(t *testing.T) {
	mock := lifecycleClient()
	mock.ListFunc = func(_ *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
		return &retlClient.RETLConnectionsPage{Data: []retlClient.RETLConnection{{ID: "other", SourceID: "different-source", DestinationID: "dst-1"}}}, nil
	}
	prior := stateData(t, jsonMapperConfig())
	prior[SourceIDKey] = "old-source"
	_, err := lifecycleHandler(t, mock).Update(context.Background(), localID, graphData(t, jsonMapperConfig()), prior)
	require.ErrorContains(t, err, "refusing to delete")
	assert.Empty(t, mock.DeleteCalls)
	assert.Empty(t, mock.CreateCalls)
}
