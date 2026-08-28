package connection

import (
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// webhookTestConfig is a minimal destination config model for the test
// registry: one plain field usable as a connect-time required key.
type webhookTestConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
}

// modeAwareUseNativeSDK mirrors the real definitions' native-SDK block: one
// field per source type, so a type it does not name has nowhere to sit.
type modeAwareUseNativeSDK struct {
	Web     *bool `mapstructure:"web"`
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
}

// nativeOnlyUseNativeSDK names web alone.
type nativeOnlyUseNativeSDK struct {
	Web *bool `mapstructure:"web"`
}

// nativeOnlyTestConfig carries use_native_sdk and no connection_mode.
type nativeOnlyTestConfig struct {
	Endpoint     string                  `mapstructure:"endpoint"`
	UseNativeSDK *nativeOnlyUseNativeSDK `mapstructure:"use_native_sdk"`
}

// modeAwareTestConfig carries both settings blocks.
type modeAwareTestConfig struct {
	Endpoint       string                 `mapstructure:"endpoint"`
	ConnectionMode common.ConnectionMode  `mapstructure:"connection_mode"`
	UseNativeSDK   *modeAwareUseNativeSDK `mapstructure:"use_native_sdk"`
}

// newTestRegistry registers three definitions, each shaped for one facet of
// the settings check:
//
//   - "webhook" has neither settings block, so the check never applies. Its
//     connect-time required keys drive the V-C5 tests instead: web needs
//     webhook_url and use_native_sdk, android needs connection_mode, ios needs
//     nothing.
//   - "mode-aware" has both blocks, so either can satisfy the check. Its
//     use_native_sdk names web, android and ios but not cloud, and only ios
//     lists connection_mode as a required key.
//   - "native-only" has use_native_sdk alone, naming web but not android.
func newTestRegistry(t *testing.T) *definitions.Registry {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:      "webhook",
		Version:   1,
		NewConfig: func() any { return &webhookTestConfig{} },
		SourceTypes: []string{
			"web",
			"android",
			"ios",
		},
		ConnectionModes: map[string][]string{
			"web":     {"cloud"},
			"android": {"cloud"},
			"ios":     {"cloud"},
		},
		SupportedSourcesValidation: map[string][]string{
			"web":     {"webhook_url", "use_native_sdk"},
			"android": {"connection_mode"},
		},
	}))
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:      "mode-aware",
		Version:   1,
		NewConfig: func() any { return &modeAwareTestConfig{} },
		SourceTypes: []string{
			"web",
			"android",
			"ios",
			"cloud",
		},
		ConnectionModes: map[string][]string{
			"web":     {"cloud", "device"},
			"android": {"cloud"},
			"ios":     {"cloud"},
			"cloud":   {"cloud"},
		},
		SupportedSourcesValidation: map[string][]string{
			"ios": {"connection_mode"},
		},
	}))
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "native-only",
		Version:     1,
		NewConfig:   func() any { return &nativeOnlyTestConfig{} },
		SourceTypes: []string{"web", "android"},
		ConnectionModes: map[string][]string{
			"web":     {"device"},
			"android": {"cloud"},
		},
	}))
	return registry
}

func addSourceResource(graph *resources.Graph, id, sourceType string, enabled bool) {
	graph.AddResource(resources.NewResource(id, esSource.ResourceType, resources.ResourceData{
		esSource.NameKey:             id,
		esSource.EnabledKey:          enabled,
		esSource.SourceDefinitionKey: sourceType,
	}, nil))
}

func addDestinationResource(graph *resources.Graph, id, destType string, enabled bool, config map[string]any) {
	graph.AddResource(resources.NewResource(id, destination.DestinationResourceType, resources.ResourceData{}, nil,
		resources.WithRawData(&destination.DestinationResource{
			ID:                id,
			Type:              destType,
			DefinitionVersion: 1,
			Enabled:           enabled,
			Config:            config,
		}),
	))
}

func addConnectionResource(graph *resources.Graph, id, sourceURN, destinationURN string) {
	graph.AddResource(resources.NewResource(id, esConnection.EventStreamConnectionResourceType, resources.ResourceData{
		esConnection.SourceKey:      &resources.PropertyRef{URN: sourceURN, Property: "id"},
		esConnection.DestinationKey: &resources.PropertyRef{URN: destinationURN, Property: "id"},
		esConnection.EnabledKey:     true,
	}, nil))
}

func connectionEntry(id, sourceID, destinationID string) esConnection.ConnectionSpec {
	return esConnection.ConnectionSpec{
		LocalID:     id,
		Source:      "#event-stream-source:" + sourceID,
		Destination: "#destination:" + destinationID,
	}
}

// compatibleGraph builds a graph holding a javascript source src-1, a webhook
// destination dest-1 satisfying the web source type's required config, and
// the connection conn-1 between them.
func compatibleGraph() *resources.Graph {
	graph := resources.NewGraph()
	addSourceResource(graph, "src-1", "javascript", true)
	addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{
		"webhook_url":    "https://example.com/hook",
		"use_native_sdk": map[string]any{"web": true},
	})
	addConnectionResource(graph, "conn-1",
		resources.URN("src-1", esSource.ResourceType),
		resources.URN("dest-1", destination.DestinationResourceType),
	)
	return graph
}

func TestConnectionSemanticValidRule_Metadata(t *testing.T) {
	rule := NewConnectionSemanticValidRule(definitions.NewRegistry())

	assert.Equal(t, "event-stream/connection/semantic-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "event stream connection endpoints must exist in the project and form a valid, compatible topology", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(esConnection.EventStreamConnectionResourceKind), rule.AppliesTo())
}

func TestConnectionEnabledEndpointsRule_Metadata(t *testing.T) {
	rule := NewConnectionEnabledEndpointsRule()

	assert.Equal(t, "event-stream/connection/enabled-endpoints-valid", rule.ID())
	assert.Equal(t, rules.Warning, rule.Severity())
	assert.Equal(t, "an enabled event stream connection needs both its endpoints enabled to deliver events", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(esConnection.EventStreamConnectionResourceKind), rule.AppliesTo())
}

func TestConnectionSemanticValid_EndpointExistence(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	t.Run("both endpoints exist and are compatible", func(t *testing.T) {
		t.Parallel()

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, compatibleGraph())
		assert.Empty(t, results)
	})

	t.Run("source not found in the project", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{})

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-missing", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/source", results[0].Reference)
		assert.Contains(t, results[0].Message, "event stream source 'src-missing' not found in the project")
	})

	t.Run("destination not found in the project", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-missing")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/destination", results[0].Reference)
		assert.Contains(t, results[0].Message, "destination 'dest-missing' not found in the project")
	})

	t.Run("malformed refs are the syntactic rule's concern", func(t *testing.T) {
		t.Parallel()

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{
				{
					LocalID:     "conn-1",
					Source:      "not-a-ref",
					Destination: "#retl-source-sql-model:wrong-kind",
				},
			},
		}

		results := validateConnectionsSemantic(registry, spec, resources.NewGraph())
		assert.Empty(t, results)
	})
}

func TestConnectionSemanticValid_PairUniqueness(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	t.Run("same pair connected twice in the spec", func(t *testing.T) {
		t.Parallel()

		graph := compatibleGraph()
		addConnectionResource(graph, "conn-2",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{
				connectionEntry("conn-1", "src-1", "dest-1"),
				connectionEntry("conn-2", "src-1", "dest-1"),
			},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 2)
		assert.Equal(t, "/connections/0", results[0].Reference)
		assert.Equal(t, "/connections/1", results[1].Reference)
		for _, r := range results {
			assert.Contains(t, r.Message, "source 'src-1' and destination 'dest-1' are connected more than once in the project")
		}
	})

	t.Run("same pair connected again by another spec", func(t *testing.T) {
		t.Parallel()

		// The duplicate lives only in the graph (another spec file); this
		// spec still reports its own entry.
		graph := compatibleGraph()
		addConnectionResource(graph, "conn-other",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "connected more than once")
	})

	t.Run("different destinations are not duplicates", func(t *testing.T) {
		t.Parallel()

		graph := compatibleGraph()
		addDestinationResource(graph, "dest-2", "webhook", true, map[string]any{
			"webhook_url":    "https://example.com/hook2",
			"use_native_sdk": map[string]any{"web": true},
		})
		addConnectionResource(graph, "conn-2",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-2", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{
				connectionEntry("conn-1", "src-1", "dest-1"),
				connectionEntry("conn-2", "src-1", "dest-2"),
			},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Empty(t, results)
	})
}

func TestConnectionSemanticValid_SourceTypeCompatibility(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	t.Run("unsupported source type", func(t *testing.T) {
		t.Parallel()

		// python connects as a cloud source, which the webhook test
		// definition does not support.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-py", "python", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-py", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-py", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/destination", results[0].Reference)
		assert.Contains(t, results[0].Message, "destination 'dest-1' (type 'webhook') does not support source 'src-py'")
		assert.Contains(t, results[0].Message, "source type 'cloud' is not among supported source types: web, android")
	})

	t.Run("missing required config fields for the source type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/destination", results[0].Reference)
		assert.Contains(t, results[0].Message, "destination 'dest-1' config is missing fields required to connect a 'web' source")
		assert.Contains(t, results[0].Message, "webhook_url")
		assert.Contains(t, results[0].Message, "use_native_sdk.web")
	})

	t.Run("source-type-scoped key present for another type only", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{
			"webhook_url":    "https://example.com/hook",
			"use_native_sdk": map[string]any{"android": true},
		})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "use_native_sdk.web")
		assert.NotContains(t, results[0].Message, "webhook_url")
	})

	t.Run("missing connection_mode entry for the source type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-droid", "android", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{
			"connection_mode": map[string]any{"web": "cloud"},
		})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-droid", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-droid", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/destination", results[0].Reference)
		assert.Contains(t, results[0].Message, "destination 'dest-1' config is missing fields required to connect a 'android' source")
		assert.Contains(t, results[0].Message, "connection_mode.android")
	})

	t.Run("connection_mode names the source type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-droid", "android", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{
			"connection_mode": map[string]any{"android": "cloud"},
		})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-droid", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-droid", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("source type without required config keys", func(t *testing.T) {
		t.Parallel()

		// ios has no supported-sources-validation entry, so an empty
		// destination config is fine.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-ios", "ios", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-ios", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-ios", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("unregistered destination type is the destination rule's concern", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "unregistered-type", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Empty(t, results)
	})
}

func TestConnectionSemanticValid_SourceTypeSettings(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	// modeAwareGraph wires a source of the given type to a "mode-aware"
	// destination with the given config.
	modeAwareGraph := func(sourceType string, config map[string]any) *resources.Graph {
		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", sourceType, true)
		addDestinationResource(graph, "dest-1", "mode-aware", true, config)
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)
		return graph
	}

	spec := esConnection.ConnectionsSpec{
		Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
	}

	t.Run("no settings block is set", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{"endpoint": "https://example.com"})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'connection_mode' or 'use_native_sdk' entry for source type 'web'",
		}}, results)
	})

	t.Run("both blocks set, neither names the source type", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{
			"connection_mode": map[string]any{"android": "cloud"},
			"use_native_sdk":  map[string]any{"android": true},
		})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'connection_mode' or 'use_native_sdk' entry for source type 'web'",
		}}, results)
	})

	t.Run("connection_mode entry present for the source type", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{
			"connection_mode": map[string]any{"web": "device"},
		})

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("use_native_sdk alone names the source type", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{
			"use_native_sdk": map[string]any{"web": true},
		})

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("a wrong-shaped block drops out, the other is still checked", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{"connection_mode": "device"})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'use_native_sdk' entry for source type 'web'",
		}}, results)
	})

	t.Run("a wrong-shaped block on its own is the destination rule's concern", func(t *testing.T) {
		t.Parallel()

		// With use_native_sdk written but not an object, no candidate is left
		// to report against; its shape is the destination rule's to flag.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "native-only", true, map[string]any{
			"use_native_sdk": "yes",
		})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("a destination with no settings block is exempt", func(t *testing.T) {
		t.Parallel()

		// bqstream, confluent_cloud and googlesheets advertise connection modes
		// as metadata but declare neither settings block, so nothing an author
		// could write would clear the error this exemption suppresses.
		// "webhook" stands in for that shape.
		//
		// Delete this subtest along with the exemption once those three
		// declare connection_mode.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "ios", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("a block that cannot name the source type is not demanded", func(t *testing.T) {
		t.Parallel()

		// "native-only" names web alone, so an android source has nowhere to
		// put an entry: the config model rejects use_native_sdk.android as an
		// unknown field.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "android", true)
		addDestinationResource(graph, "dest-1", "native-only", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("a block that can name the source type is demanded", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "native-only", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'use_native_sdk' entry for source type 'web'",
		}}, results)
	})

	t.Run("the error names only the blocks that could hold the entry", func(t *testing.T) {
		t.Parallel()

		// A cloud source fits connection_mode's open map but has no field in
		// use_native_sdk, so only connection_mode is named.
		graph := modeAwareGraph("python", map[string]any{})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'connection_mode' entry for source type 'cloud'",
		}}, results)
	})

	t.Run("a key V-C5 already requires is reported once", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("ios", map[string]any{})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config is missing fields required to connect a 'ios' source: connection_mode.ios",
		}}, results)
	})
}

func TestConnectionSemanticValid_DestinationFamily(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	t.Run("destination shared with a rETL source", func(t *testing.T) {
		t.Parallel()

		// A project connection whose source URN belongs to another family
		// (as the retl-connections kind will produce once it lands).
		graph := compatibleGraph()
		addConnectionResource(graph, "conn-retl",
			resources.URN("my-model", "retl-source-sql-model"),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/destination", results[0].Reference)
		assert.Contains(t, results[0].Message, "destination 'dest-1' is also connected to rETL source 'my-model'")
		assert.Contains(t, results[0].Message, "cannot receive from both event stream and rETL sources")
	})

	t.Run("rETL connection to a different destination is fine", func(t *testing.T) {
		t.Parallel()

		graph := compatibleGraph()
		addConnectionResource(graph, "conn-retl",
			resources.URN("my-model", "retl-source-sql-model"),
			resources.URN("dest-other", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("missing destination reports only its absence", func(t *testing.T) {
		t.Parallel()

		// The rETL edge points at the same destination URN, but with no
		// destination resource in the project the family check would only
		// cascade on top of the V-C1 error.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addConnectionResource(graph, "conn-retl",
			resources.URN("my-model", "retl-source-sql-model"),
			resources.URN("dest-missing", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-missing")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/destination", results[0].Reference)
		assert.Contains(t, results[0].Message, "destination 'dest-missing' not found in the project")
	})
}

func TestConnectionEnabledEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("all enabled", func(t *testing.T) {
		t.Parallel()

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateEnabledEndpoints("", "", nil, spec, compatibleGraph())
		assert.Empty(t, results)
	})

	t.Run("enabled connection with disabled source", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", false)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{})

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateEnabledEndpoints("", "", nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/source", results[0].Reference)
		assert.Contains(t, results[0].Message, "connection 'conn-1' is enabled but its source 'src-1' is disabled")
	})

	t.Run("enabled connection with disabled destination", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "webhook", false, map[string]any{})

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateEnabledEndpoints("", "", nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/connections/0/destination", results[0].Reference)
		assert.Contains(t, results[0].Message, "connection 'conn-1' is enabled but its destination 'dest-1' is disabled")
	})

	t.Run("disabled connection never warns", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", false)
		addDestinationResource(graph, "dest-1", "webhook", false, map[string]any{})

		entry := connectionEntry("conn-1", "src-1", "dest-1")
		entry.Enabled = boolPtr(false)
		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{entry},
		}

		results := validateEnabledEndpoints("", "", nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("missing endpoints are the semantic-valid rule's concern", func(t *testing.T) {
		t.Parallel()

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-missing", "dest-missing")},
		}

		results := validateEnabledEndpoints("", "", nil, spec, resources.NewGraph())
		assert.Empty(t, results)
	})
}
