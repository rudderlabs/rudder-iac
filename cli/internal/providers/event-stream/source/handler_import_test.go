package source_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func TestFormatForExport_MissingTrackingPlan(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	logPath := filepath.Join(homeDir, ".rudder", "cli.log")
	before, err := os.Stat(logPath)
	require.NoError(t, err)

	remoteSource := &sourceClient.EventStreamSource{
		ID: "remote-source", Name: "Zoopla App Dev", Type: "javascript", Enabled: true,
		WorkspaceID: "workspace", TrackingPlan: &sourceClient.TrackingPlan{ID: "missing-tp"},
	}
	importable := resources.NewRemoteResources()
	importable.Set(source.ResourceType, map[string]*resources.RemoteResource{
		remoteSource.ID: {ID: remoteSource.ID, ExternalID: "zoopla-app-dev", Data: remoteSource},
	})
	refResolver := &resolver.ImportRefResolver{
		Importable: importable, Remote: resources.NewRemoteResources(), Graph: resources.NewGraph(),
	}

	var output bytes.Buffer
	ui.SetWriter(&output)
	defer ui.RestoreWriter()
	entities, _, err := source.NewHandler(source.NewMockSourceClient(), importDir).FormatForExport(importable, &mockNamer{}, refResolver)
	require.NoError(t, err)
	require.Len(t, entities, 1)
	spec, ok := entities[0].Content.(*specs.Spec)
	require.True(t, ok)
	assert.Equal(t, map[string]any{
		"id": "zoopla-app-dev", "name": "Zoopla App Dev", "type": "javascript", "enabled": true,
	}, spec.Spec)

	// The package logger writes to a shared append-only file; restrict the check to this call.
	var logBytes []byte
	require.Eventually(t, func() bool {
		logBytes, err = os.ReadFile(logPath)
		return err == nil && len(logBytes) > int(before.Size())
	}, time.Second, time.Millisecond*10)
	warning := string(logBytes[before.Size():])
	assert.Equal(t, 1, strings.Count(warning, "level=WARN"), "unexpected log: %s", warning)
	assert.Contains(t, warning, "Zoopla App Dev")
	assert.Contains(t, warning, "remote-source")
	assert.Contains(t, warning, "missing-tp")
	assert.Contains(t, warning, "governance was not imported")
	assert.Contains(t, warning, "remains unmanaged")
	assert.Equal(t, 1, strings.Count(output.String(), "Warning:"))
	assert.Contains(t, output.String(), "Zoopla App Dev")
	assert.Contains(t, output.String(), "remote-source")
	assert.Contains(t, output.String(), "missing-tp")
	assert.Contains(t, output.String(), "governance was not imported")
	assert.Contains(t, output.String(), "will not appear in later plans")
}

func TestFormatForExport_MergeMatchedMissingTrackingPlanWarns(t *testing.T) {
	remoteSource := &sourceClient.EventStreamSource{
		ID: "remote-source", Name: "Zoopla App Dev", WorkspaceID: "workspace",
		TrackingPlan: &sourceClient.TrackingPlan{ID: "missing-tp"},
	}
	importable := resources.NewRemoteResources()
	importable.Set(source.ResourceType, map[string]*resources.RemoteResource{
		remoteSource.ID: {
			ID: remoteSource.ID, ExternalID: "local-source", Data: remoteSource,
			MatchedWith: resources.NewResource("local-source", source.ResourceType, resources.ResourceData{}, nil),
		},
	})
	refResolver := &resolver.ImportRefResolver{
		Importable: importable, Remote: resources.NewRemoteResources(), Graph: resources.NewGraph(),
	}
	var output bytes.Buffer
	ui.SetWriter(&output)
	defer ui.RestoreWriter()

	entities, entries, err := source.NewHandler(nil, importDir).FormatForExport(importable, &mockNamer{}, refResolver)
	require.NoError(t, err)
	assert.Empty(t, entities)
	require.Len(t, entries, 1)
	assert.Equal(t, "event-stream-source:local-source", entries[0].URN)
	assert.Equal(t, 1, strings.Count(output.String(), "Warning:"))
	assert.Contains(t, output.String(), "Zoopla App Dev")
	assert.Contains(t, output.String(), "remote-source")
	assert.Contains(t, output.String(), "missing-tp")
	assert.Contains(t, output.String(), "governance was not imported")
}

func TestFormatForExport_OtherTrackingPlanResolutionError(t *testing.T) {
	importable := resources.NewRemoteResources()
	importable.Set(source.ResourceType, map[string]*resources.RemoteResource{
		"remote-source": {
			ID: "remote-source", ExternalID: "local-source",
			Data: &sourceClient.EventStreamSource{
				ID: "remote-source", Name: "Source", Type: "javascript", WorkspaceID: "workspace",
				TrackingPlan: &sourceClient.TrackingPlan{ID: "remote-tp"},
			},
		},
	})
	entities, _, err := source.NewHandler(nil, importDir).FormatForExport(importable, &mockNamer{}, &mockResolver{resolveFunc: func(string, string) (string, error) {
		return "", fmt.Errorf("resource not present in resources graph")
	}})
	require.ErrorContains(t, err, "resolving tracking plan reference: resource not present in resources graph")
	assert.Nil(t, entities)
}

// TestFormatForExport_TrackingPlanReferences tests the bug fix for RUD-2580
// where importing event-stream sources fails if the source references a tracking plan
// that is already managed by the CLI (not in the importable collection).
func TestFormatForExport_TrackingPlanReferences(t *testing.T) {
	tests := []struct {
		name                string
		setupImportable     func() *resources.RemoteResources
		setupRemote         func() *resources.RemoteResources
		setupGraph          func() *resources.Graph
		expectedSpec        *specs.Spec
		expectedErrContains string
	}{
		{
			name: "source with tracking plan already managed by CLI",
			setupImportable: func() *resources.RemoteResources {
				importable := resources.NewRemoteResources()
				importableSourceMap := map[string]*resources.RemoteResource{
					"remote123": {
						ID:         "remote123",
						ExternalID: "test-source-1",
						Reference:  "#event-stream-source:test-source-1",
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
			setupRemote: func() *resources.RemoteResources {
				remote := resources.NewRemoteResources()
				remoteTpMap := map[string]*resources.RemoteResource{
					"remote-tp-456": {
						ID:         "remote-tp-456",
						ExternalID: "existing-tp-456",
						Reference:  "#tracking-plan:existing-tp-456",
					},
				}
				remote.Set(types.TrackingPlanResourceType, remoteTpMap)
				return remote
			},
			setupGraph: func() *resources.Graph {
				graph := resources.NewGraph()
				tpResource := resources.NewResource(
					"existing-tp-456",
					types.TrackingPlanResourceType,
					nil,
					nil,
					resources.WithResourceFileMetadata("#tracking-plan:existing-tp-456"),
				)
				graph.AddResource(tpResource)
				return graph
			},
			expectedSpec: &specs.Spec{
				Kind:    "event-stream-source",
				Version: specs.SpecVersionV1,
				Spec: map[string]any{
					"id":      "test-source-1",
					"name":    "Test Source 1",
					"type":    "javascript",
					"enabled": true,
					"governance": map[string]any{
						"validations": map[string]any{
							"tracking_plan": "#tracking-plan:existing-tp-456",
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
						"workspaces": []any{
							map[string]any{
								"workspace_id": "workspace-123",
								"resources": []any{
									map[string]any{
										"urn":       "event-stream-source:test-source-1",
										"remote_id": "remote123",
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
			setupImportable: func() *resources.RemoteResources {
				importable := resources.NewRemoteResources()
				importableSourceMap := map[string]*resources.RemoteResource{
					"remote123": {
						ID:         "remote123",
						ExternalID: "test-source-1",
						Reference:  "#event-stream-source:test-source-1",
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

				importableTpMap := map[string]*resources.RemoteResource{
					"remote-tp-456": {
						ID:         "remote-tp-456",
						ExternalID: "new-tp-456",
						Reference:  "#tracking-plan:new-tp-456",
					},
				}
				importable.Set(types.TrackingPlanResourceType, importableTpMap)
				return importable
			},
			setupRemote: func() *resources.RemoteResources {
				return resources.NewRemoteResources()
			},
			setupGraph: func() *resources.Graph {
				return resources.NewGraph()
			},
			expectedSpec: &specs.Spec{
				Kind:    "event-stream-source",
				Version: specs.SpecVersionV1,
				Spec: map[string]any{
					"id":      "test-source-1",
					"name":    "Test Source 1",
					"type":    "javascript",
					"enabled": true,
					"governance": map[string]any{
						"validations": map[string]any{
							"tracking_plan": "#tracking-plan:new-tp-456",
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
						"workspaces": []any{
							map[string]any{
								"workspace_id": "workspace-123",
								"resources": []any{
									map[string]any{
										"urn":       "event-stream-source:test-source-1",
										"remote_id": "remote123",
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
			setupImportable: func() *resources.RemoteResources {
				importable := resources.NewRemoteResources()
				importableSourceMap := map[string]*resources.RemoteResource{
					"remote123": {
						ID:         "remote123",
						ExternalID: "test-source-1",
						Reference:  "#event-stream-source:test-source-1",
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
			setupRemote: func() *resources.RemoteResources {
				return resources.NewRemoteResources()
			},
			setupGraph: func() *resources.Graph {
				return resources.NewGraph()
			},
			expectedSpec: &specs.Spec{
				Kind:    "event-stream-source",
				Version: specs.SpecVersionV1,
				Spec: map[string]any{
					"id":      "test-source-1",
					"name":    "Test Source 1",
					"type":    "javascript",
					"enabled": true,
				},
				Metadata: map[string]any{
					"name": "event-stream-source",
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "workspace-123",
								"resources": []any{
									map[string]any{
										"urn":       "event-stream-source:test-source-1",
										"remote_id": "remote123",
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
			importable := tt.setupImportable()
			remote := tt.setupRemote()
			graph := tt.setupGraph()

			resolverImpl := &resolver.ImportRefResolver{
				Importable: importable,
				Remote:     remote,
				Graph:      graph,
			}

			entities, _, err := handler.FormatForExport(importable, &mockNamer{}, resolverImpl)

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
