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
// registry: two plain fields usable as connect-time required keys, plus the
// connection_mode block that makes them reachable — keys are looked up by the
// mode a spec declares, which this model has to be able to carry.
type webhookTestConfig struct {
	WebhookURL     string                `mapstructure:"webhook_url"`
	AuthToken      string                `mapstructure:"auth_token"`
	ConnectionMode common.ConnectionMode `mapstructure:"connection_mode"`
}

// barebonesTestConfig models neither settings block.
type barebonesTestConfig struct {
	Endpoint string `mapstructure:"endpoint"`
}

// modeAwareTestConfig carries the source-type settings block.
type modeAwareTestConfig struct {
	Endpoint       string                `mapstructure:"endpoint"`
	ConnectionMode common.ConnectionMode `mapstructure:"connection_mode"`
}

// multiModeTestConfig backs a definition whose web source type supports more
// than one connection mode, so required keys differ by mode.
type multiModeTestConfig struct {
	AppID          string                `mapstructure:"app_id"`
	APIKey         string                `mapstructure:"api_key"`
	ConnectionMode common.ConnectionMode `mapstructure:"connection_mode"`
}

// newTestRegistry registers four definitions, each shaped for one facet of the
// checks:
//
//   - "webhook" drives the V-C5 tests: web needs webhook_url and auth_token,
//     android needs connection_mode, ios needs nothing. It models
//     connection_mode so those keys are reachable at all.
//   - "barebones" has neither settings block, so the settings check never
//     applies.
//   - "mode-aware" has both blocks, so either can satisfy the settings check.
//     ios lists connection_mode as a required key.
//   - "multimode" mirrors intercom: a web source needs api_key in cloud mode
//     and app_id in device mode, so required keys cannot be resolved from the
//     source type alone.
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
			"android_kotlin",
		},
		ConnectionModes: map[string][]string{
			"web":            {"cloud"},
			"android":        {"cloud"},
			"ios":            {"cloud"},
			"android_kotlin": {"cloud"},
		},
		ConnectionRequiredKeys: map[string]map[string][]string{
			"web":     {"cloud": {"webhook_url", "auth_token"}},
			"android": {"cloud": {"connection_mode"}},
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
		ConnectionRequiredKeys: map[string]map[string][]string{
			"ios": {"cloud": {"connection_mode"}},
		},
	}))
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "barebones",
		Version:     1,
		NewConfig:   func() any { return &barebonesTestConfig{} },
		SourceTypes: []string{"web", "ios"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
			"ios": {"cloud"},
		},
	}))

	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:            "multimode",
		Version:         1,
		NewConfig:       func() any { return &multiModeTestConfig{} },
		SourceTypes:     []string{"web", "android"},
		ConnectionModes: map[string][]string{"web": {"cloud", "device"}, "android": {"cloud"}},
		ConnectionRequiredKeys: map[string]map[string][]string{
			"web":     {"cloud": {"api_key"}, "device": {"app_id"}},
			"android": {"cloud": {"api_key"}},
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
		"webhook_url":     "https://example.com/hook",
		"auth_token":      "token-1",
		"connection_mode": map[string]any{"web": "cloud"},
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
			"webhook_url":     "https://example.com/hook2",
			"auth_token":      "token-2",
			"connection_mode": map[string]any{"web": "cloud"},
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

	t.Run("android_kotlin source resolves to its own type", func(t *testing.T) {
		t.Parallel()

		// The event stream sources API reports the Android Kotlin SDK under the
		// same token destinations declare, so the source reaches the membership
		// check untranslated.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-ak", "android_kotlin", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{
			"connection_mode": map[string]any{"android_kotlin": "cloud"},
		})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-ak", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-ak", "dest-1")},
		}

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("missing required config fields for the source type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{
			"connection_mode": map[string]any{"web": "cloud"},
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
		assert.Equal(t, "/connections/0/destination", results[0].Reference)
		assert.Contains(t, results[0].Message, "destination 'dest-1' config is missing fields required to connect a 'web' source")
		assert.Contains(t, results[0].Message, "webhook_url")
		assert.Contains(t, results[0].Message, "auth_token")
	})

	t.Run("declared connection_mode selects that mode's required keys", func(t *testing.T) {
		t.Parallel()

		// app_id satisfies device mode; the cloud-mode key api_key is absent
		// and must not be demanded.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "multimode", true, map[string]any{
			"app_id":          "intercom-app",
			"connection_mode": map[string]any{"web": "device"},
		})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("the other mode's key does not satisfy the declared mode", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "multimode", true, map[string]any{
			"app_id":          "intercom-app",
			"connection_mode": map[string]any{"web": "cloud"},
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
		assert.Contains(t, results[0].Message, "destination 'dest-1' config is missing fields required to connect a 'web' source")
		assert.Contains(t, results[0].Message, "api_key")
	})

	// Without a declared mode there are no required keys to look up, so V-C5
	// stays silent; the settings check reports the absent declaration.
	t.Run("undeclared connection_mode on a multi-mode source type is not checked", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "javascript", true)
		addDestinationResource(graph, "dest-1", "multimode", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "has no 'connection_mode' entry for source type 'web'")
	})

	// A source type the definition offers exactly one mode for is no exception:
	// ConnectionModes says what the destination supports, not what this
	// connection does, so the mode is still undeclared.
	t.Run("undeclared connection_mode on a single-mode source type is not checked", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "android", true)
		addDestinationResource(graph, "dest-1", "multimode", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		spec := esConnection.ConnectionsSpec{
			Connections: []esConnection.ConnectionSpec{connectionEntry("conn-1", "src-1", "dest-1")},
		}

		results := validateConnectionsSemantic(registry, spec, graph)
		require.Len(t, results, 1)
		assert.NotContains(t, results[0].Message, "api_key")
		assert.Contains(t, results[0].Message, "has no 'connection_mode' entry for source type 'android'")
	})

	// V-C5 can never report connection_mode missing: the absence that would
	// make it missing is the same one that leaves the mode undeclared. The
	// settings check reports it instead.
	t.Run("connection_mode as a required key is reported by the settings check", func(t *testing.T) {
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
		assert.NotContains(t, results[0].Message, "missing fields required")
		assert.Contains(t, results[0].Message, "has no 'connection_mode' entry for source type 'android'")
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

		// ios has no required keys, so declaring its mode is all the config needs.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-ios", "ios", true)
		addDestinationResource(graph, "dest-1", "webhook", true, map[string]any{
			"connection_mode": map[string]any{"ios": "cloud"},
		})
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
			Message:   "destination 'dest-1' config has no 'connection_mode' entry for source type 'web'",
		}}, results)
	})

	t.Run("connection_mode set for another source type", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{
			"connection_mode": map[string]any{"android": "cloud"},
		})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'connection_mode' entry for source type 'web'",
		}}, results)
	})

	t.Run("connection_mode entry present for the source type", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{
			"connection_mode": map[string]any{"web": "device"},
		})

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("a wrong-shaped connection_mode block is the destination rule's concern", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{"connection_mode": "device"})

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("connection_mode written with no value names nothing", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("javascript", map[string]any{
			"connection_mode": nil,
		})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'connection_mode' entry for source type 'web'",
		}}, results)
	})

	t.Run("a destination declaring neither block asks for nothing", func(t *testing.T) {
		t.Parallel()

		// "barebones" declares neither settings block, so there is nowhere to
		// write the entry an error would ask for.
		graph := resources.NewGraph()
		addSourceResource(graph, "src-1", "ios", true)
		addDestinationResource(graph, "dest-1", "barebones", true, map[string]any{})
		addConnectionResource(graph, "conn-1",
			resources.URN("src-1", esSource.ResourceType),
			resources.URN("dest-1", destination.DestinationResourceType),
		)

		assert.Empty(t, validateConnectionsSemantic(registry, spec, graph))
	})

	t.Run("the error names only the blocks that could hold the entry", func(t *testing.T) {
		t.Parallel()

		// A cloud source fits connection_mode's open map, so connection_mode is named.
		graph := modeAwareGraph("python", map[string]any{})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'connection_mode' entry for source type 'cloud'",
		}}, results)
	})

	// Deferring to V-C5 cannot help when the required key is connection_mode:
	// V-C5 resolves the mode from the very entry whose absence is at issue, so
	// there is nothing to defer to.
	t.Run("a connection_mode required key does not silence this check", func(t *testing.T) {
		t.Parallel()

		graph := modeAwareGraph("ios", map[string]any{})

		results := validateConnectionsSemantic(registry, spec, graph)
		assert.Equal(t, []rules.ValidationResult{{
			Reference: "/connections/0/destination",
			Message:   "destination 'dest-1' config has no 'connection_mode' entry for source type 'ios'",
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
