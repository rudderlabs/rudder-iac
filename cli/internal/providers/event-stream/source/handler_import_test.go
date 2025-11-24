package source_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	dcstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// TestFormatForExport_TrackingPlanReferences tests the bug fix for RUD-2580
// where importing event-stream sources fails if the source references a tracking plan
// that is already managed by the CLI (not in the importable collection).
func TestFormatForExport_TrackingPlanReferences(t *testing.T) {
	tests := []struct {
		name                string
		setupImportable     func() *resources.ResourceCollection
		setupRemote         func() *resources.ResourceCollection
		setupGraph          func() *resources.Graph
		expectedSpec        *specs.Spec
		expectedErrContains string
	}{
		{
			name: "source with tracking plan already managed by CLI",
			setupImportable: func() *resources.ResourceCollection {
				importable := resources.NewResourceCollection()
				importableSourceMap := map[string]*resources.RemoteResource{
					"remote123": {
						ID:         "remote123",
						ExternalID: "test-source-1",
						Reference:  "#/event-stream-source/event-stream-source/test-source-1",
						Data: &sourceClient.EventStreamSource{
							ID:          "remote123",
							ExternalID:  "test-source-1",
							Name:        "Test Source 1",
							Type:        "javascript",
							Enabled:     true,
							WorkspaceID: "workspace-123",
							TrackingPlan: &sourceClient.TrackingPlan{
								ID: "remote-tp-456",
								Config: &sourceClient.TrackingPlanConfig{
									Track: &sourceClient.TrackConfig{
										DropUnplannedEvents: boolPtr(true),
										EventTypeConfig: &sourceClient.EventTypeConfig{
											PropagateViolations: boolPtr(false),
											DropOtherViolations: boolPtr(true),
										},
									},
								},
							},
						},
					},
				}
				importable.Set(source.ResourceType, importableSourceMap)
				return importable
			},
			setupRemote: func() *resources.ResourceCollection {
				remote := resources.NewResourceCollection()
				remoteTpMap := map[string]*resources.RemoteResource{
					"remote-tp-456": {
						ID:         "remote-tp-456",
						ExternalID: "existing-tp-456",
						Reference:  "#/tracking-plans/tracking-plan/existing-tp-456",
					},
				}
				remote.Set(dcstate.TrackingPlanResourceType, remoteTpMap)
				return remote
			},
			setupGraph: func() *resources.Graph {
				graph := resources.NewGraph()
				tpResource := resources.NewResource(
					"existing-tp-456",
					dcstate.TrackingPlanResourceType,
					nil,
					nil,
					resources.WithResourceFileMetadata("#/tracking-plans/tracking-plan/existing-tp-456"),
				)
				graph.AddResource(tpResource)
				return graph
			},
			expectedSpec: &specs.Spec{
				Kind:    "event-stream-source",
				Version: "rudder/v0.1",
				Spec: map[string]any{
					"id":      "test-source-1",
					"name":    "Test Source 1",
					"type":    "javascript",
					"enabled": true,
					"governance": map[string]any{
						"validations": map[string]any{
							"tracking_plan": "#/tracking-plans/tracking-plan/existing-tp-456",
							"config": map[string]any{
								"track": map[string]any{
									"drop_unplanned_events": true,
									"propagate_violations":  false,
									"drop_other_violations": true,
								},
							},
						},
					},
				},
				Metadata: map[string]any{
					"name": "event-stream-source",
					"import": map[string]any{
						"workspaces": []importremote.WorkspaceImportMetadata{
							{
								WorkspaceID: "workspace-123",
								Resources: []importremote.ImportIds{
									{
										LocalID:  "test-source-1",
										RemoteID: "remote123",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "source with tracking plan being imported together",
			setupImportable: func() *resources.ResourceCollection {
				importable := resources.NewResourceCollection()
				importableSourceMap := map[string]*resources.RemoteResource{
					"remote123": {
						ID:         "remote123",
						ExternalID: "test-source-1",
						Reference:  "#/event-stream-source/event-stream-source/test-source-1",
						Data: &sourceClient.EventStreamSource{
							ID:          "remote123",
							ExternalID:  "test-source-1",
							Name:        "Test Source 1",
							Type:        "javascript",
							Enabled:     true,
							WorkspaceID: "workspace-123",
							TrackingPlan: &sourceClient.TrackingPlan{
								ID: "remote-tp-456",
								Config: &sourceClient.TrackingPlanConfig{
									Track: &sourceClient.TrackConfig{
										DropUnplannedEvents: boolPtr(false),
										EventTypeConfig: &sourceClient.EventTypeConfig{
											PropagateViolations: boolPtr(true),
										},
									},
								},
							},
						},
					},
				}
				importable.Set(source.ResourceType, importableSourceMap)

				// Tracking plan is also in importable (being imported together)
				importableTpMap := map[string]*resources.RemoteResource{
					"remote-tp-456": {
						ID:         "remote-tp-456",
						ExternalID: "new-tp-456",
						Reference:  "#/tracking-plans/tracking-plan/new-tp-456",
					},
				}
				importable.Set(dcstate.TrackingPlanResourceType, importableTpMap)
				return importable
			},
			setupRemote: func() *resources.ResourceCollection {
				return resources.NewResourceCollection()
			},
			setupGraph: func() *resources.Graph {
				return resources.NewGraph()
			},
			expectedSpec: &specs.Spec{
				Kind:    "event-stream-source",
				Version: "rudder/v0.1",
				Spec: map[string]any{
					"id":      "test-source-1",
					"name":    "Test Source 1",
					"type":    "javascript",
					"enabled": true,
					"governance": map[string]any{
						"validations": map[string]any{
							"tracking_plan": "#/tracking-plans/tracking-plan/new-tp-456",
							"config": map[string]any{
								"track": map[string]any{
									"drop_unplanned_events": false,
									"propagate_violations":  true,
								},
							},
						},
					},
				},
				Metadata: map[string]any{
					"name": "event-stream-source",
					"import": map[string]any{
						"workspaces": []importremote.WorkspaceImportMetadata{
							{
								WorkspaceID: "workspace-123",
								Resources: []importremote.ImportIds{
									{
										LocalID:  "test-source-1",
										RemoteID: "remote123",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "source without tracking plan",
			setupImportable: func() *resources.ResourceCollection {
				importable := resources.NewResourceCollection()
				importableSourceMap := map[string]*resources.RemoteResource{
					"remote123": {
						ID:         "remote123",
						ExternalID: "test-source-1",
						Reference:  "#/event-stream-source/event-stream-source/test-source-1",
						Data: &sourceClient.EventStreamSource{
							ID:          "remote123",
							ExternalID:  "test-source-1",
							Name:        "Test Source 1",
							Type:        "javascript",
							Enabled:     true,
							WorkspaceID: "workspace-123",
						},
					},
				}
				importable.Set(source.ResourceType, importableSourceMap)
				return importable
			},
			setupRemote: func() *resources.ResourceCollection {
				return resources.NewResourceCollection()
			},
			setupGraph: func() *resources.Graph {
				return resources.NewGraph()
			},
			expectedSpec: &specs.Spec{
				Kind:    "event-stream-source",
				Version: "rudder/v0.1",
				Spec: map[string]any{
					"id":      "test-source-1",
					"name":    "Test Source 1",
					"type":    "javascript",
					"enabled": true,
				},
				Metadata: map[string]any{
					"name": "event-stream-source",
					"import": map[string]any{
						"workspaces": []importremote.WorkspaceImportMetadata{
							{
								WorkspaceID: "workspace-123",
								Resources: []importremote.ImportIds{
									{
										LocalID:  "test-source-1",
										RemoteID: "remote123",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := source.NewMockSourceClient()
			handler := source.NewHandler(mockClient, importDir)
			ctx := context.Background()

			importable := tt.setupImportable()
			remote := tt.setupRemote()
			graph := tt.setupGraph()

			resolverImpl := &resolver.ImportRefResolver{
				Importable: importable,
				Remote:     remote,
				Graph:      graph,
			}

			entities, err := handler.FormatForExport(ctx, importable, &mockNamer{}, resolverImpl)

			if tt.expectedErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrContains)
				return
			}

			require.NoError(t, err)
			require.Len(t, entities, 1)

			spec, ok := entities[0].Content.(*specs.Spec)
			require.True(t, ok)

			assert.Equal(t, tt.expectedSpec, spec)
		})
	}
}
