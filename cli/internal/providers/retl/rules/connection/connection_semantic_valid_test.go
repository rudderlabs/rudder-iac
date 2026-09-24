package connection

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	activecampaign "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/active_campaign"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/am"
	attentivetag "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/attentive_tag"
	bingads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bingads_offline_conversions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bqstream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/braze"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio"
	customerioaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio_audience"
	facebookconversions "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_conversions"
	facebookpixel "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_pixel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/ga4"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/gcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/hs"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/iterable"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/mp"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/posthog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3"
	tiktokads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/tiktok_ads"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/webhook"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// endpointOnlyTestConfig models neither settings block, so the settings check
// has nowhere to ask for an entry and stays out of these cases' results.
type endpointOnlyTestConfig struct {
	Endpoint string `mapstructure:"endpoint"`
}

// newTestRegistry holds the real definitions the fixtures name — http reaches
// warehouse sources through the JSON mapper, bingads also supports the visual
// mapper, customerio_audience drives its own destination-specific flow, braze
// demands rest_api_key before a warehouse source connects (V-C5) — plus three
// minimal fakes for the cases no shipped definition can produce:
//
//   - "eventstreamonly" declares no warehouse source type at all (V-C4).
//   - "mirroronly" accepts mirror alone without the visual mapper, so the JSON
//     mapper flow leaves the two endpoints with no behaviour in common (V-R2).
//   - "hyphen-mapper" is an object-mapping destination whose names carry a
//     hyphen (V-R10 backend parity).
func newTestRegistry(t *testing.T) *definitions.Registry {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))
	require.NoError(t, registry.Register(bingads.NewDefinition()))
	require.NoError(t, registry.Register(customerioaudience.NewDefinition()))
	require.NoError(t, registry.Register(braze.NewDefinition()))

	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:            "eventstreamonly",
		Version:         1,
		NewConfig:       func() any { return &endpointOnlyTestConfig{} },
		SourceTypes:     []string{common.SourceTypeWeb},
		ConnectionModes: map[string][]string{common.SourceTypeWeb: {"cloud"}},
	}))
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:            "mirroronly",
		Version:         1,
		NewConfig:       func() any { return &endpointOnlyTestConfig{} },
		SourceTypes:     []string{common.SourceTypeWarehouse},
		ConnectionModes: map[string][]string{common.SourceTypeWarehouse: {"cloud"}},
		SyncBehaviours:  []string{"mirror"},
	}))
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:                 "hyphen-mapper",
		APIType:              "HYPHEN-MAPPER",
		Version:              1,
		NewConfig:            func() any { return &endpointOnlyTestConfig{} },
		SourceTypes:          []string{common.SourceTypeWarehouse},
		ConnectionModes:      map[string][]string{common.SourceTypeWarehouse: {"cloud"}},
		SyncBehaviours:       []string{"mirror"},
		SupportsVisualMapper: true,
	}))
	return registry
}

// modelFixture and destinationFixture are the two graph endpoints a case wires
// together, written out so each case names only what it changes.
type modelFixture struct {
	id               string
	sourceDefinition string
	primaryKey       string
	enabled          bool
}

type destinationFixture struct {
	id      string
	typ     string
	enabled bool
	config  map[string]any
}

func postgresModel() modelFixture {
	return modelFixture{id: "users-model", sourceDefinition: "postgres", primaryKey: "user_id", enabled: true}
}

func httpDestination() destinationFixture {
	return destinationFixture{
		id:      "my-http-destination",
		typ:     "http",
		enabled: true,
		config: map[string]any{
			"api_url":         "https://example.com/events",
			"auth":            "noAuth",
			"connection_mode": map[string]any{"warehouse": "cloud"},
		},
	}
}

func bingDestination() destinationFixture {
	return destinationFixture{
		id:      "my-bing-destination",
		typ:     "bingads_offline_conversions",
		enabled: true,
		config:  map[string]any{"connection_mode": map[string]any{"warehouse": "cloud"}},
	}
}

func sqlModelURN(id string) string {
	return resources.URN(id, sqlmodel.ResourceType)
}

func destinationResourceURN(id string) string {
	return resources.URN(id, destination.DestinationResourceType)
}

func addSQLModelResource(graph *resources.Graph, model modelFixture) {
	graph.AddResource(resources.NewResource(model.id, sqlmodel.ResourceType, resources.ResourceData{
		retlConnection.SourceDefinitionKey: model.sourceDefinition,
		retlConnection.SourcePrimaryKeyKey: model.primaryKey,
		retlConnection.SourceEnabledKey:    model.enabled,
	}, nil))
}

func addDestinationResource(graph *resources.Graph, dest destinationFixture) {
	graph.AddResource(resources.NewResource(dest.id, destination.DestinationResourceType, resources.ResourceData{}, nil,
		resources.WithRawData(&destination.DestinationResource{
			ID:                dest.id,
			Type:              dest.typ,
			DefinitionVersion: 1,
			Enabled:           dest.enabled,
			Config:            dest.config,
		}),
	))
}

func addRETLConnectionResource(graph *resources.Graph, id, sourceURN, destinationURN string) {
	graph.AddResource(resources.NewResource(id, retlConnection.ResourceType, resources.ResourceData{
		retlConnection.SourceKey:      &resources.PropertyRef{URN: sourceURN, Property: "id"},
		retlConnection.DestinationKey: &resources.PropertyRef{URN: destinationURN, Property: "id"},
		retlConnection.EnabledKey:     true,
	}, nil))
}

func addESConnectionResource(graph *resources.Graph, id, sourceURN, destinationURN string) {
	graph.AddResource(resources.NewResource(id, esConnection.EventStreamConnectionResourceType, resources.ResourceData{
		esConnection.SourceKey:      &resources.PropertyRef{URN: sourceURN, Property: "id"},
		esConnection.DestinationKey: &resources.PropertyRef{URN: destinationURN, Property: "id"},
		esConnection.EnabledKey:     true,
	}, nil))
}

// connectedGraph is the standard project shape: one SQL model, one destination
// and the rETL connection between them.
func connectedGraph(model modelFixture, dest destinationFixture) *resources.Graph {
	graph := resources.NewGraph()
	addSQLModelResource(graph, model)
	addDestinationResource(graph, dest)
	addRETLConnectionResource(graph, "users-to-webhook", sqlModelURN(model.id), destinationResourceURN(dest.id))
	return graph
}

// connectionTo is the base connection entry aimed at a given destination.
func connectionTo(destinationID string) retlConnection.ConnectionSpec {
	c := validConnection()
	c.Destination = "#destination:" + destinationID
	return c
}

func specOf(connections ...retlConnection.ConnectionSpec) retlConnection.ConnectionsSpec {
	return retlConnection.ConnectionsSpec{Connections: connections}
}

func TestConnectionSemanticValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewConnectionSemanticValidRule(definitions.NewRegistry())

	assert.Equal(t, "retl/connection/semantic-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "retl connection endpoints must exist in the project and form a valid, compatible topology", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(retlConnection.ResourceKind), rule.AppliesTo())
}

func TestConnectionEnabledEndpointsRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewConnectionEnabledEndpointsRule()

	assert.Equal(t, "retl/connection/enabled-endpoints-valid", rule.ID())
	assert.Equal(t, rules.Warning, rule.Severity())
	assert.Equal(t, "an enabled retl connection needs both its endpoints enabled to sync", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(retlConnection.ResourceKind), rule.AppliesTo())
}

// semanticContext is the context the engine builds for a semantic rule: the
// raw spec map plus the graph the project's resources were built into.
func semanticContext(raw map[string]any, graph *resources.Graph) *rules.ValidationContext {
	return &rules.ValidationContext{
		Kind:    retlConnection.ResourceKind,
		Version: specs.SpecVersionV1,
		Spec:    raw,
		Graph:   graph,
	}
}

// TestConnectionSemanticValidRule_Validate and its enabled-endpoints sibling
// drive the rules through their public entry point, so the constructors' wiring
// and the engine's "/spec" prefixing are covered rather than only the
// validators behind them.
func TestConnectionSemanticValidRule_Validate(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	addDestinationResource(graph, httpDestination())
	addRETLConnectionResource(graph, "users-to-webhook",
		sqlModelURN("users-model"), destinationResourceURN("my-http-destination"))

	assert.Equal(t, []rules.ValidationResult{{
		Reference: "/spec/connections/0/source",
		Message:   "rETL source 'users-model' not found in the project",
	}}, NewConnectionSemanticValidRule(newTestRegistry(t)).Validate(semanticContext(validRawSpec(), graph)))
}

func TestConnectionEnabledEndpointsRule_Validate(t *testing.T) {
	t.Parallel()

	disabledModel := postgresModel()
	disabledModel.enabled = false

	assert.Equal(t, []rules.ValidationResult{{
		Reference: "/spec/connections/0/source",
		Message:   "connection 'users-to-webhook' is enabled but its source 'users-model' is disabled; the connection will not sync",
	}}, NewConnectionEnabledEndpointsRule().Validate(
		semanticContext(validRawSpec(), connectedGraph(disabledModel, httpDestination())),
	))
}

func TestConnectionSemanticValid_Endpoints(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	// A project holding the destination and the connection edge but no source
	// resource, and the mirror image of it.
	danglingSource := func() *resources.Graph {
		graph := resources.NewGraph()
		addDestinationResource(graph, httpDestination())
		addRETLConnectionResource(graph, "users-to-webhook", sqlModelURN("users-model"), destinationResourceURN("my-http-destination"))
		return graph
	}
	danglingDestination := func() *resources.Graph {
		graph := resources.NewGraph()
		addSQLModelResource(graph, postgresModel())
		addRETLConnectionResource(graph, "users-to-webhook", sqlModelURN("users-model"), destinationResourceURN("my-http-destination"))
		return graph
	}

	// The same dangling endpoints, in a project whose other edges would make
	// every topology check fire: the pair is duplicated, a second rETL source
	// feeds the destination and so does an event stream source.
	danglingSourceInCrowdedProject := func() *resources.Graph {
		graph := danglingSource()
		orders := postgresModel()
		orders.id = "orders-model"
		addSQLModelResource(graph, orders)
		addRETLConnectionResource(graph, "users-to-webhook-again",
			sqlModelURN("users-model"), destinationResourceURN("my-http-destination"))
		addRETLConnectionResource(graph, "orders-to-webhook",
			sqlModelURN("orders-model"), destinationResourceURN("my-http-destination"))
		addESConnectionResource(graph, "js-to-webhook",
			resources.URN("my-js-source", esSource.ResourceType), destinationResourceURN("my-http-destination"))
		return graph
	}
	danglingDestinationInCrowdedProject := func() *resources.Graph {
		graph := danglingDestination()
		addRETLConnectionResource(graph, "users-to-webhook-again",
			sqlModelURN("users-model"), destinationResourceURN("my-http-destination"))
		addESConnectionResource(graph, "js-to-webhook",
			resources.URN("my-js-source", esSource.ResourceType), destinationResourceURN("my-http-destination"))
		return graph
	}

	tests := []struct {
		name     string
		graph    func() *resources.Graph
		mutate   func(c *retlConnection.ConnectionSpec)
		expected []rules.ValidationResult
	}{
		{
			name: "both endpoints exist and are compatible",
		},
		{
			name:  "source not found in the project",
			graph: danglingSource,
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/source", Message: "rETL source 'users-model' not found in the project"},
			},
		},
		{
			name:  "destination not found in the project",
			graph: danglingDestination,
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/destination", Message: "destination 'my-http-destination' not found in the project"},
			},
		},
		{
			name:  "a missing source leaves only the error the author can act on",
			graph: danglingSourceInCrowdedProject,
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/source", Message: "rETL source 'users-model' not found in the project"},
			},
		},
		{
			name:  "a missing destination leaves only the error the author can act on",
			graph: danglingDestinationInCrowdedProject,
			expected: []rules.ValidationResult{
				{Reference: "/connections/0/destination", Message: "destination 'my-http-destination' not found in the project"},
			},
		},
		{
			name:   "a malformed source ref is the syntactic rule's concern",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Source = "users-model" },
		},
		{
			name:   "a source ref of the event stream family is the syntactic rule's concern",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Source = "#event-stream-source:my-js-source" },
		},
		{
			name:   "a destination ref of the wrong kind is the syntactic rule's concern",
			mutate: func(c *retlConnection.ConnectionSpec) { c.Destination = "#retl-source-sql-model:users-model" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			graph := connectedGraph(postgresModel(), httpDestination())
			if tt.graph != nil {
				graph = tt.graph()
			}
			c := connectionTo("my-http-destination")
			if tt.mutate != nil {
				tt.mutate(&c)
			}

			assert.Equal(t, tt.expected, validateConnectionsSemantic(registry, specOf(c), graph))
		})
	}
}

func TestConnectionSemanticValid_Topology(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	tests := []struct {
		name     string
		graph    func() *resources.Graph
		spec     retlConnection.ConnectionsSpec
		expected []rules.ValidationResult
	}{
		{
			name: "the same pair connected twice in one spec",
			graph: func() *resources.Graph {
				graph := connectedGraph(postgresModel(), httpDestination())
				addRETLConnectionResource(graph, "users-to-webhook-again",
					sqlModelURN("users-model"), destinationResourceURN("my-http-destination"))
				return graph
			},
			spec: specOf(connectionTo("my-http-destination"), connectionTo("my-http-destination")),
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0",
					Message:   "source 'users-model' and destination 'my-http-destination' are connected more than once in the project; a source-destination pair can only be connected once",
				},
				{
					Reference: "/connections/1",
					Message:   "source 'users-model' and destination 'my-http-destination' are connected more than once in the project; a source-destination pair can only be connected once",
				},
			},
		},
		{
			name: "the same pair connected again from another spec file",
			graph: func() *resources.Graph {
				graph := connectedGraph(postgresModel(), httpDestination())
				addRETLConnectionResource(graph, "users-to-webhook-elsewhere",
					sqlModelURN("users-model"), destinationResourceURN("my-http-destination"))
				return graph
			},
			spec: specOf(connectionTo("my-http-destination")),
			expected: []rules.ValidationResult{{
				Reference: "/connections/0",
				Message:   "source 'users-model' and destination 'my-http-destination' are connected more than once in the project; a source-destination pair can only be connected once",
			}},
		},
		{
			name: "two rETL sources feeding one destination",
			graph: func() *resources.Graph {
				graph := connectedGraph(postgresModel(), httpDestination())
				orders := postgresModel()
				orders.id = "orders-model"
				addSQLModelResource(graph, orders)
				addRETLConnectionResource(graph, "orders-to-webhook",
					sqlModelURN("orders-model"), destinationResourceURN("my-http-destination"))
				return graph
			},
			spec: specOf(connectionTo("my-http-destination")),
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/destination",
				Message:   "destination 'my-http-destination' is also connected to rETL source 'orders-model' in this project; a destination can only receive from one rETL source",
			}},
		},
		{
			name: "an event stream source sharing the destination",
			graph: func() *resources.Graph {
				graph := connectedGraph(postgresModel(), httpDestination())
				addESConnectionResource(graph, "js-to-webhook",
					resources.URN("my-js-source", esSource.ResourceType), destinationResourceURN("my-http-destination"))
				return graph
			},
			spec: specOf(connectionTo("my-http-destination")),
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/destination",
				Message:   "destination 'my-http-destination' is also connected to event stream source 'my-js-source' in this project; a destination cannot receive from both event stream and rETL sources",
			}},
		},
		{
			name: "unrelated destinations are not shared",
			graph: func() *resources.Graph {
				graph := connectedGraph(postgresModel(), httpDestination())
				orders := postgresModel()
				orders.id = "orders-model"
				other := httpDestination()
				other.id = "my-other-destination"
				addSQLModelResource(graph, orders)
				addDestinationResource(graph, other)
				addRETLConnectionResource(graph, "orders-elsewhere",
					sqlModelURN("orders-model"), destinationResourceURN("my-other-destination"))
				addESConnectionResource(graph, "js-elsewhere",
					resources.URN("my-js-source", esSource.ResourceType), destinationResourceURN("my-other-destination"))
				return graph
			},
			spec: specOf(connectionTo("my-http-destination")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, validateConnectionsSemantic(registry, tt.spec, tt.graph()))
		})
	}
}

func TestConnectionSemanticValid_DestinationCompatibility(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	tests := []struct {
		name        string
		destination destinationFixture
		mutate      func(c *retlConnection.ConnectionSpec)
		expected    []rules.ValidationResult
	}{
		{
			name:        "a destination declaring no warehouse source type",
			destination: destinationFixture{id: "my-es-destination", typ: "eventstreamonly", enabled: true, config: map[string]any{}},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/destination",
				Message:   "destination 'my-es-destination' (type 'eventstreamonly') does not accept rETL sources: source type 'warehouse' is not among supported source types: web",
			}},
		},
		{
			// V-C4 and V-R5 are independent verdicts about the same destination,
			// so the flow classification must not answer for both.
			name:        "a destination that neither takes rETL sources nor maps objects",
			destination: destinationFixture{id: "my-es-destination", typ: "eventstreamonly", enabled: true, config: map[string]any{}},
			mutate:      objectMappingEntry,
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/destination",
					Message:   "destination 'my-es-destination' (type 'eventstreamonly') does not accept rETL sources: source type 'warehouse' is not among supported source types: web",
				},
				{
					Reference: "/connections/0/config/object",
					Message:   `'object' is not allowed: destination api type "eventstreamonly" does not support object mapping`,
				},
			},
		},
		{
			name:        "a destination driving its own rETL flow",
			destination: destinationFixture{id: "my-cio-destination", typ: "customerio_audience", enabled: true, config: map[string]any{}},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/destination",
				Message:   `destination api type "CUSTOMERIO_AUDIENCE" uses a destination-specific rETL flow, which is not supported`,
			}},
		},
		{
			name:        "an object aimed at a destination without the visual mapper",
			destination: httpDestination(),
			mutate:      objectMappingEntry,
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/object",
				Message:   `'object' is not allowed: destination api type "HTTP" does not support object mapping`,
			}},
		},
		{
			name: "a destination config missing what a warehouse source needs to connect",
			destination: destinationFixture{
				id: "my-braze-destination", typ: "braze", enabled: true,
				config: map[string]any{"connection_mode": map[string]any{"warehouse": "cloud"}},
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/destination",
				Message:   "destination 'my-braze-destination' config is missing fields required to connect a 'warehouse' source: rest_api_key",
			}},
		},
		{
			name: "a destination config naming no settings for the warehouse source type",
			destination: destinationFixture{
				id: "my-http-destination", typ: "http", enabled: true,
				config: map[string]any{"api_url": "https://example.com/events", "auth": "noAuth"},
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/destination",
				Message:   "destination 'my-http-destination' config has no 'connection_mode' entry for source type 'warehouse'",
			}},
		},
		{
			name:        "an object mapping destination whose name carries a hyphen",
			destination: destinationFixture{id: "my-hyphen-destination", typ: "hyphen-mapper", enabled: true, config: map[string]any{}},
			mutate:      objectMappingEntry,
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/destination",
				Message:   "destination 'my-hyphen-destination' (type 'hyphen-mapper', api type 'HYPHEN-MAPPER') cannot be used with object mapping: its name contains a hyphen, which the backend cannot reconstruct",
			}},
		},
		{
			name:        "an unregistered destination type is the destination rule's concern",
			destination: destinationFixture{id: "my-unknown-destination", typ: "not-registered", enabled: true, config: map[string]any{}},
		},
		{
			name:        "a visual mapper destination on the object mapping flow",
			destination: bingDestination(),
			mutate:      objectMappingEntry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := connectionTo(tt.destination.id)
			if tt.mutate != nil {
				tt.mutate(&c)
			}

			graph := connectedGraph(postgresModel(), tt.destination)
			assert.Equal(t, tt.expected, validateConnectionsSemantic(registry, specOf(c), graph))
		})
	}
}

// TestConnectionSemanticValid_WarehouseDestinationFlows drives verified
// destinations that accept warehouse sources through their real definitions:
// every one runs a JSON mapper connection on upsert, and an object picks object
// mapping, on mirror, only where upstream declares the visual mapper.
// Customer.io accepts warehouse sources yet keeps its destination-specific
// flow refused.
func TestConnectionSemanticValid_WarehouseDestinationFlows(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	for _, def := range []*definitions.DestinationDefinition{
		activecampaign.NewDefinition(), am.NewDefinition(), attentivetag.NewDefinition(),
		bqstream.NewDefinition(), braze.NewDefinition(), customerio.NewDefinition(),
		facebookconversions.NewDefinition(), facebookpixel.NewDefinition(), ga4.NewDefinition(),
		gcs.NewDefinition(), hs.NewDefinition(), iterable.NewDefinition(), mp.NewDefinition(),
		posthog.NewDefinition(), s3.NewDefinition(), tiktokads.NewDefinition(), webhook.NewDefinition(),
	} {
		require.NoError(t, registry.Register(def))
	}

	objectRefused := func(apiType string) []rules.ValidationResult {
		return []rules.ValidationResult{{
			Reference: "/connections/0/config/object",
			Message:   fmt.Sprintf("'object' is not allowed: destination api type %q does not support object mapping", apiType),
		}}
	}
	specificFlow := []rules.ValidationResult{{
		Reference: "/connections/0/destination",
		Message:   `destination api type "CUSTOMERIO" uses a destination-specific rETL flow, which is not supported`,
	}}

	tests := []struct {
		typ string
		// requiredKey is what the destination needs before a warehouse source
		// connects (V-C5), set so the flow checks are all that is left.
		requiredKey   string
		jsonMapper    []rules.ValidationResult
		objectMapping []rules.ValidationResult
	}{
		{typ: "active_campaign", objectMapping: objectRefused("ACTIVE_CAMPAIGN")},
		{typ: "am"},
		{typ: "attentive_tag", objectMapping: objectRefused("ATTENTIVE_TAG")},
		{typ: "bqstream", objectMapping: objectRefused("BQSTREAM")},
		{typ: "braze", requiredKey: "rest_api_key"},
		{typ: "customerio", jsonMapper: specificFlow, objectMapping: specificFlow},
		{typ: "facebook_conversions", objectMapping: objectRefused("FACEBOOK_CONVERSIONS")},
		{typ: "facebook_pixel", requiredKey: "access_token", objectMapping: objectRefused("FACEBOOK_PIXEL")},
		{typ: "ga4", objectMapping: objectRefused("GA4")},
		{typ: "gcs", objectMapping: objectRefused("GCS")},
		{typ: "hs"},
		{typ: "iterable"},
		{typ: "mp", objectMapping: objectRefused("MP")},
		{typ: "posthog", objectMapping: objectRefused("POSTHOG")},
		{typ: "s3", objectMapping: objectRefused("S3")},
		{typ: "tiktok_ads", objectMapping: objectRefused("TIKTOK_ADS")},
		{typ: "webhook", objectMapping: objectRefused("WEBHOOK")},
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			t.Parallel()

			config := map[string]any{"connection_mode": map[string]any{"warehouse": "cloud"}}
			if tt.requiredKey != "" {
				config[tt.requiredKey] = "example-key"
			}
			dest := destinationFixture{id: "my-destination", typ: tt.typ, enabled: true, config: config}
			graph := connectedGraph(postgresModel(), dest)

			jsonEntry := connectionTo(dest.id)
			assert.Equal(t, tt.jsonMapper, validateConnectionsSemantic(registry, specOf(jsonEntry), graph), "json mapper")

			objectEntry := connectionTo(dest.id)
			objectMappingEntry(&objectEntry)
			assert.Equal(t, tt.objectMapping, validateConnectionsSemantic(registry, specOf(objectEntry), graph), "object mapping")
		})
	}
}

func TestConnectionSemanticValid_Mappings(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	tests := []struct {
		name        string
		destination destinationFixture
		mutate      func(c *retlConnection.ConnectionSpec)
		expected    []rules.ValidationResult
	}{
		{
			name:        "a JSON identifier aimed outside the reserved user identities",
			destination: httpDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Identifiers = []retlConnection.MappingSpec{{From: "user_id", To: "traits.id"}}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/identifiers/0/to",
				Message:   "'to' is not valid: a JSON mapping identifier must target 'user_id' or 'anonymous_id'",
			}},
		},
		{
			name:        "a JSON mapper connection delivering nothing",
			destination: httpDestination(),
			mutate:      func(c *retlConnection.ConnectionSpec) { c.Config.Mappings = nil },
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/mappings",
				Message:   "'mappings' is required with JSON mapping",
			}},
		},
		{
			name:        "a JSON mapping aimed at user_id",
			destination: httpDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Mappings = []retlConnection.MappingSpec{{From: "legacy_id", To: "user_id"}}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/mappings/0/to",
				Message:   "'to' is not valid: mapping to 'user_id' is reserved for identifiers",
			}},
		},
		{
			name:        "a JSON mapping aimed at anonymous_id",
			destination: httpDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Mappings = []retlConnection.MappingSpec{{From: "device_id", To: "anonymous_id"}}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/mappings/0/to",
				Message:   "'to' is not valid: mapping to 'anonymous_id' is reserved for identifiers",
			}},
		},
		{
			name:        "the external id target is not reserved on the JSON mapper flow",
			destination: httpDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.Mappings = []retlConnection.MappingSpec{{From: "crm_id", To: "context.externalId[0].id"}}
			},
		},
		{
			name:        "object mapping carrying constants",
			destination: bingDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				objectMappingEntry(c)
				c.Config.Constants = []retlConnection.ConstantSpec{{Key: "source", Value: "warehouse"}}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/constants",
				Message:   "'constants' is not allowed with object mapping",
			}},
		},
		{
			name:        "object mapping carrying an event",
			destination: bingDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				objectMappingEntry(c)
				c.Config.Event = &retlConnection.EventSpec{Type: "track", Name: "Signed Up"}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/event",
				Message:   "'event' is not allowed with object mapping",
			}},
		},
		{
			name:        "object mapping carrying more than one identifier",
			destination: bingDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				objectMappingEntry(c)
				c.Config.Identifiers = []retlConnection.MappingSpec{
					{From: "user_id", To: "user_id"},
					{From: "crm_id", To: "context.externalId[0].id"},
				}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/identifiers",
				Message:   "object mapping supports a single identifier, found 2",
			}},
		},
		{
			name:        "an object mapping aimed at user_id",
			destination: bingDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				objectMappingEntry(c)
				c.Config.Mappings = []retlConnection.MappingSpec{{From: "legacy_id", To: "user_id"}}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/mappings/0/to",
				Message:   "'to' is not valid: mapping to 'user_id' is reserved for identifiers",
			}},
		},
		{
			name:        "an object mapping aimed at the external id",
			destination: bingDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				objectMappingEntry(c)
				c.Config.Mappings = []retlConnection.MappingSpec{{From: "crm_id", To: "context.externalId[0].id"}}
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/mappings/0/to",
				Message:   "'to' is not valid: mapping to 'context.externalId[0].id' is reserved for identifiers",
			}},
		},
		{
			name:        "the anonymous id target is not reserved on the object mapping flow",
			destination: bingDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				objectMappingEntry(c)
				c.Config.Mappings = []retlConnection.MappingSpec{{From: "device_id", To: "anonymous_id"}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := connectionTo(tt.destination.id)
			tt.mutate(&c)

			graph := connectedGraph(postgresModel(), tt.destination)
			assert.Equal(t, tt.expected, validateConnectionsSemantic(registry, specOf(c), graph))
		})
	}
}

// objectMappingEntry turns the base entry into the object mapping shape the
// visual-mapper destinations run: an object, no mappings, and the only sync
// behaviour those destinations accept.
func objectMappingEntry(c *retlConnection.ConnectionSpec) {
	c.Config.Object = lo.ToPtr("Audience")
	c.Config.Mappings = nil
	c.Config.SyncBehaviour = "mirror"
}

func TestConnectionSemanticValid_SourceCapability(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	tests := []struct {
		name        string
		model       modelFixture
		destination destinationFixture
		mutate      func(c *retlConnection.ConnectionSpec)
		expected    []rules.ValidationResult
	}{
		{
			name:        "a source definition this CLI version has no rETL metadata for",
			model:       modelFixture{id: "users-model", sourceDefinition: "oracle", primaryKey: "user_id", enabled: true},
			destination: httpDestination(),
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/source",
				Message:   "rETL source 'users-model' uses source definition 'oracle', whose rETL capabilities this CLI version does not know; upgrade the CLI to validate this connection",
			}},
		},
		{
			name:        "a source without a primary key",
			model:       modelFixture{id: "users-model", sourceDefinition: "postgres", enabled: true},
			destination: httpDestination(),
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/source",
				Message:   "rETL source 'users-model' declares no primary_key, which source definition 'postgres' requires to sync",
			}},
		},
		{
			name:        "an s3 source is exempt from the primary key but cannot back a SQL model",
			model:       modelFixture{id: "users-model", sourceDefinition: "s3", enabled: true},
			destination: httpDestination(),
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/source",
				Message:   "rETL source 'users-model' is a SQL model, which source definition 's3' does not support",
			}},
		},
		{
			name:        "an sftp source is exempt from the primary key too",
			model:       modelFixture{id: "users-model", sourceDefinition: "sftp", enabled: true},
			destination: httpDestination(),
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/source",
				Message:   "rETL source 'users-model' is a SQL model, which source definition 'sftp' does not support",
			}},
		},
		{
			name:        "sync settings a source definition cannot carry",
			model:       modelFixture{id: "users-model", sourceDefinition: "clickhouse", primaryKey: "user_id", enabled: true},
			destination: httpDestination(),
			mutate: func(c *retlConnection.ConnectionSpec) {
				c.Config.SyncSettings = &retlConnection.SyncSettingsSpec{
					FailedKeys: &retlConnection.FailedKeysSpec{Retry: lo.ToPtr(true)},
				}
			},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/source",
					Message:   "rETL source 'users-model' is a SQL model, which source definition 'clickhouse' does not support",
				},
				{
					Reference: "/connections/0/config/sync_settings",
					Message:   "'sync_settings' is not allowed: source definition 'clickhouse' does not support sync settings",
				},
			},
		},
		{
			name:        "the JSON mapper flow has no mirror to offer",
			model:       postgresModel(),
			destination: httpDestination(),
			mutate:      func(c *retlConnection.ConnectionSpec) { c.Config.SyncBehaviour = "mirror" },
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/sync_behaviour",
				Message:   "'sync_behaviour' must be one of [upsert full] for source definition 'postgres' and destination 'my-http-destination' on the json_mapper flow",
			}},
		},
		{
			name:  "endpoints with no sync behaviour in common",
			model: postgresModel(),
			destination: destinationFixture{
				id: "my-mirror-destination", typ: "mirroronly", enabled: true, config: map[string]any{},
			},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/sync_behaviour",
				Message:   "source definition 'postgres' and destination 'my-mirror-destination' share no sync behaviour for the json_mapper flow; the connection cannot sync",
			}},
		},
		{
			name:        "a sync behaviour both endpoints accept",
			model:       postgresModel(),
			destination: httpDestination(),
			mutate:      func(c *retlConnection.ConnectionSpec) { c.Config.SyncBehaviour = "full" },
		},

		// Source checks do not wait on the destination's prerequisites.
		{
			name:        "a source without a primary key aimed at a destination that takes no rETL sources",
			model:       modelFixture{id: "users-model", sourceDefinition: "postgres", enabled: true},
			destination: destinationFixture{id: "my-es-destination", typ: "eventstreamonly", enabled: true, config: map[string]any{}},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/destination",
					Message:   "destination 'my-es-destination' (type 'eventstreamonly') does not accept rETL sources: source type 'warehouse' is not among supported source types: web",
				},
				{
					Reference: "/connections/0/source",
					Message:   "rETL source 'users-model' declares no primary_key, which source definition 'postgres' requires to sync",
				},
			},
		},
		{
			name:        "an unknown source definition behind a destination-specific flow",
			model:       modelFixture{id: "users-model", sourceDefinition: "oracle", primaryKey: "user_id", enabled: true},
			destination: destinationFixture{id: "my-cio-destination", typ: "customerio_audience", enabled: true, config: map[string]any{}},
			expected: []rules.ValidationResult{
				{
					Reference: "/connections/0/destination",
					Message:   `destination api type "CUSTOMERIO_AUDIENCE" uses a destination-specific rETL flow, which is not supported`,
				},
				{
					Reference: "/connections/0/source",
					Message:   "rETL source 'users-model' uses source definition 'oracle', whose rETL capabilities this CLI version does not know; upgrade the CLI to validate this connection",
				},
			},
		},
		{
			name:        "a source without a primary key aimed at an unregistered destination type",
			model:       modelFixture{id: "users-model", sourceDefinition: "postgres", enabled: true},
			destination: destinationFixture{id: "my-unknown-destination", typ: "not-registered", enabled: true, config: map[string]any{}},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/source",
				Message:   "rETL source 'users-model' declares no primary_key, which source definition 'postgres' requires to sync",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := connectionTo(tt.destination.id)
			if tt.mutate != nil {
				tt.mutate(&c)
			}

			graph := connectedGraph(tt.model, tt.destination)
			assert.Equal(t, tt.expected, validateConnectionsSemantic(registry, specOf(c), graph))
		})
	}
}

func TestConnectionEnabledEndpoints(t *testing.T) {
	t.Parallel()

	disabledModel := postgresModel()
	disabledModel.enabled = false

	disabledDestination := httpDestination()
	disabledDestination.enabled = false

	tests := []struct {
		name     string
		graph    func() *resources.Graph
		mutate   func(c *retlConnection.ConnectionSpec)
		expected []rules.ValidationResult
	}{
		{
			name: "both endpoints enabled",
		},
		{
			name:  "an enabled connection from a disabled source",
			graph: func() *resources.Graph { return connectedGraph(disabledModel, httpDestination()) },
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/source",
				Message:   "connection 'users-to-webhook' is enabled but its source 'users-model' is disabled; the connection will not sync",
			}},
		},
		{
			name:  "an enabled connection to a disabled destination",
			graph: func() *resources.Graph { return connectedGraph(postgresModel(), disabledDestination) },
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/destination",
				Message:   "connection 'users-to-webhook' is enabled but its destination 'my-http-destination' is disabled; the connection will not sync",
			}},
		},
		{
			name:   "a disabled connection never warns",
			graph:  func() *resources.Graph { return connectedGraph(disabledModel, disabledDestination) },
			mutate: func(c *retlConnection.ConnectionSpec) { c.Enabled = lo.ToPtr(false) },
		},
		{
			name:  "missing endpoints are the semantic-valid rule's concern",
			graph: resources.NewGraph,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			graph := connectedGraph(postgresModel(), httpDestination())
			if tt.graph != nil {
				graph = tt.graph()
			}
			c := connectionTo("my-http-destination")
			if tt.mutate != nil {
				tt.mutate(&c)
			}

			assert.Equal(t, tt.expected, validateEnabledEndpoints("", "", nil, specOf(c), graph))
		})
	}
}
