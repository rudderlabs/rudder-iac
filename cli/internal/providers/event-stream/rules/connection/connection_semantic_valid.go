package connection

import (
	"fmt"
	"slices"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// NewConnectionSemanticValidRule validates cross-resource concerns for event
// stream connections: both endpoints exist in the project (V-C1), a
// source–destination pair is connected only once (V-C3), the destination
// definition supports the source's mapped type (V-C4) and its config carries
// the fields that type requires to connect (V-C5), and the destination is not
// shared with a rETL source (V-E1).
func NewConnectionSemanticValidRule(registry *definitions.Registry) rules.Rule {
	return prules.NewTypedRule(
		"event-stream/connection/semantic-valid",
		rules.Error,
		"event stream connection endpoints must exist in the project and form a valid, compatible topology",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(esConnection.EventStreamConnectionResourceKind),
			func(_ string, _ string, _ map[string]any, spec esConnection.ConnectionsSpec, graph *resources.Graph) []rules.ValidationResult {
				return validateConnectionsSemantic(registry, spec, graph)
			},
		),
	)
}

// NewConnectionEnabledEndpointsRule warns when an enabled connection points
// at a disabled endpoint (V-C7): the API accepts it, but the connection will
// not deliver anything until both endpoints are enabled.
func NewConnectionEnabledEndpointsRule() rules.Rule {
	return prules.NewTypedRule(
		"event-stream/connection/enabled-endpoints-valid",
		rules.Warning,
		"an enabled event stream connection needs both its endpoints enabled to deliver events",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(esConnection.EventStreamConnectionResourceKind),
			validateEnabledEndpoints,
		),
	)
}

func validateConnectionsSemantic(
	registry *definitions.Registry,
	spec esConnection.ConnectionsSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	edges := projectConnectionEdges(graph)

	var results []rules.ValidationResult
	for index, c := range spec.Connections {
		endpoints := resolveEndpoints(graph, c)

		results = append(results, validateEndpointsExist(index, endpoints)...)
		if endpoints.sourceRefOK && endpoints.destinationRefOK {
			results = append(results, validatePairUniqueness(edges, index, endpoints)...)
		}
		// Sharing a destination is only meaningful for a destination that
		// exists; a dangling ref already carries the V-C1 error above.
		if endpoints.destination != nil {
			results = append(results, validateDestinationHasOnlyEventStreamSources(edges, index, endpoints.destinationID)...)
		}
		if endpoints.source != nil && endpoints.destination != nil {
			results = append(results, validateSourceTypeCompatibility(registry, index, endpoints)...)
		}
	}
	return results
}

// validateEnabledEndpoints (V-C7): a connection marked enabled should have
// both its source and destination enabled — otherwise it will not deliver
// anything. Endpoints that are missing from the graph are the semantic-valid
// rule's concern and are skipped here.
var validateEnabledEndpoints = func(
	_ string,
	_ string,
	_ map[string]any,
	spec esConnection.ConnectionsSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	var results []rules.ValidationResult
	for index, c := range spec.Connections {
		// enabled defaults to true when omitted, mirroring the handler.
		if c.Enabled != nil && !*c.Enabled {
			continue
		}

		endpoints := resolveEndpoints(graph, c)
		if endpoints.source != nil && sourceDisabled(endpoints.source) {
			results = append(results, rules.ValidationResult{
				Reference: sourceRef(index),
				Message: fmt.Sprintf(
					"connection '%s' is enabled but its source '%s' is disabled; the connection will not deliver events",
					c.LocalID, endpoints.sourceID,
				),
			})
		}
		if endpoints.destination != nil && destinationDisabled(endpoints.destination) {
			results = append(results, rules.ValidationResult{
				Reference: destinationRef(index),
				Message: fmt.Sprintf(
					"connection '%s' is enabled but its destination '%s' is disabled; the connection will not deliver events",
					c.LocalID, endpoints.destinationID,
				),
			})
		}
	}
	return results
}

// connectionEndpoints is one connection entry's endpoint resolution, shared
// by every per-entry check: the parsed local ids, whether each ref parsed as
// the right kind, and the graph resources (nil when absent from the project).
type connectionEndpoints struct {
	sourceID    string
	sourceRefOK bool
	source      *resources.Resource

	destinationID    string
	destinationRefOK bool
	destination      *resources.Resource
}

// resolveEndpoints parses both endpoint refs of a connection entry and looks
// the endpoints up in the graph. Malformed or wrong-kind refs stay unresolved
// — the spec-syntax rule already reports those, so semantic checks skip them
// quietly.
func resolveEndpoints(graph *resources.Graph, c esConnection.ConnectionSpec) connectionEndpoints {
	var endpoints connectionEndpoints

	if id, ok := endpointID(c.Source, esSource.ResourceKind); ok {
		endpoints.sourceID = id
		endpoints.sourceRefOK = true
		endpoints.source, _ = graph.GetResource(resources.URN(id, esSource.ResourceType))
	}
	if id, ok := endpointID(c.Destination, destination.DestinationSpecKind); ok {
		endpoints.destinationID = id
		endpoints.destinationRefOK = true
		endpoints.destination, _ = graph.GetResource(resources.URN(id, destination.DestinationResourceType))
	}
	return endpoints
}

// validateEndpointsExist (V-C1): both endpoint refs must resolve to resources
// that exist in the project.
func validateEndpointsExist(index int, endpoints connectionEndpoints) []rules.ValidationResult {
	var results []rules.ValidationResult
	if endpoints.sourceRefOK && endpoints.source == nil {
		results = append(results, rules.ValidationResult{
			Reference: sourceRef(index),
			Message:   fmt.Sprintf("event stream source '%s' not found in the project", endpoints.sourceID),
		})
	}
	if endpoints.destinationRefOK && endpoints.destination == nil {
		results = append(results, rules.ValidationResult{
			Reference: destinationRef(index),
			Message:   fmt.Sprintf("destination '%s' not found in the project", endpoints.destinationID),
		})
	}
	return results
}

// validatePairUniqueness (V-C3): the same source–destination pair can only be
// connected once in the project. The count runs over every project
// connection, so duplicates are flagged whether they sit in this spec or in
// another one.
func validatePairUniqueness(edges []connectionEdge, index int, endpoints connectionEndpoints) []rules.ValidationResult {
	pair := connectionEdge{
		sourceURN:      resources.URN(endpoints.sourceID, esSource.ResourceType),
		destinationURN: resources.URN(endpoints.destinationID, destination.DestinationResourceType),
	}

	count := 0
	for _, e := range edges {
		if e == pair {
			count++
		}
	}
	if count <= 1 {
		return nil
	}

	return []rules.ValidationResult{{
		Reference: connectionRef(index),
		Message: fmt.Sprintf(
			"source '%s' and destination '%s' are connected more than once in the project; a source-destination pair can only be connected once",
			endpoints.sourceID, endpoints.destinationID,
		),
	}}
}

// validateDestinationHasOnlyEventStreamSources (V-E1): an event stream source
// cannot share a destination with a rETL source — every project connection to
// this destination must come from an event stream source. This is enforced
// only in the webapp today, so the CLI is the last line of defense.
func validateDestinationHasOnlyEventStreamSources(edges []connectionEdge, index int, destinationID string) []rules.ValidationResult {
	destinationURN := resources.URN(destinationID, destination.DestinationResourceType)
	sourcePrefix := esSource.ResourceType + ":"

	var results []rules.ValidationResult
	for _, e := range edges {
		if e.destinationURN != destinationURN || strings.HasPrefix(e.sourceURN, sourcePrefix) {
			continue
		}
		_, foreignID, _ := strings.Cut(e.sourceURN, ":")
		results = append(results, rules.ValidationResult{
			Reference: destinationRef(index),
			Message: fmt.Sprintf(
				"destination '%s' is also connected to rETL source '%s' in this project; a destination cannot receive from both event stream and rETL sources",
				destinationID, foreignID,
			),
		})
	}
	return results
}

// validateSourceTypeCompatibility checks V-C4 — the destination definition's
// supported source types must include the source's mapped type — and, only
// when the type is supported, V-C5 — the destination's config must carry
// every field the definition requires for that source type to connect.
func validateSourceTypeCompatibility(
	registry *definitions.Registry,
	index int,
	endpoints connectionEndpoints,
) []rules.ValidationResult {
	// A source resource without a type cannot reach a built graph — the
	// source's own spec rules require one — so there is nothing to report
	// here without it.
	sourceType, _ := endpoints.source.Data()[esSource.SourceDefinitionKey].(string)
	if sourceType == "" {
		return nil
	}

	// Destination resources always carry *destination.DestinationResource;
	// anything else is the destination provider's corruption, not this
	// rule's to report.
	destinationData, ok := endpoints.destination.RawData().(*destination.DestinationResource)
	if !ok {
		return nil
	}

	// An unregistered (type, version) pair is the destination spec-syntax
	// rule's concern; skip quietly here.
	registered, err := registry.Get(destinationData.Type, destinationData.DefinitionVersion)
	if err != nil {
		return nil
	}

	// Event stream SDK source definitions carry no source category upstream,
	// so the token resolves from the definition type alone.
	token := common.SourceTypeToken(sourceType, "")
	supported := registered.SupportedSourceTypes()
	if !slices.Contains(supported, token) {
		return []rules.ValidationResult{{
			Reference: destinationRef(index),
			Message: fmt.Sprintf(
				"destination '%s' (type '%s') does not support source '%s': source type '%s' is not among supported source types: %s",
				endpoints.destinationID, destinationData.Type, endpoints.sourceID, token, strings.Join(supported, ", "),
			),
		}}
	}

	missing := missingRequiredConfigKeys(registered, token, destinationData.Config)
	if len(missing) > 0 {
		return []rules.ValidationResult{{
			Reference: destinationRef(index),
			Message: fmt.Sprintf(
				"destination '%s' config is missing fields required to connect a '%s' source: %s",
				endpoints.destinationID, token, strings.Join(missing, ", "),
			),
		}}
	}

	return nil
}

// missingRequiredConfigKeys returns the definition-required config keys for
// the given source type that the destination's config does not carry.
func missingRequiredConfigKeys(
	registered *definitions.RegisteredDefinition,
	sourceType string,
	config map[string]any,
) []string {
	var missing []string
	for _, key := range registered.SupportedSourcesValidation(sourceType) {
		// Source-type-scoped keys (connection_mode, use_native_sdk) are not
		// flat config fields: the destination spec carries them as maps keyed
		// by source type (e.g. config.use_native_sdk.web). For those,
		// "present" means the map has an entry for the connecting source's
		// type, and a miss is reported as <key>.<source type>.
		if slices.Contains(registered.SourceTypeConfigKeys(), key) {
			block, _ := config[key].(map[string]any)
			if _, present := block[sourceType]; !present {
				missing = append(missing, fmt.Sprintf("%s.%s", key, sourceType))
			}
			continue
		}
		if _, present := config[key]; !present {
			missing = append(missing, key)
		}
	}
	return missing
}

// connectionEdge is one project connection reduced to its endpoint URNs.
type connectionEdge struct {
	sourceURN      string
	destinationURN string
}

// projectConnectionEdges reduces every event stream connection in the graph
// to its endpoint URN pair for the topology checks (V-C3, V-E1). Event
// stream connections are the only project-managed connections today; when
// the retl-connections kind lands, its validations must fold its connections
// into these checks. Known limitation: only project-managed connections are
// visible — a connection that exists remotely but is not in the project is
// invisible at validate time.
func projectConnectionEdges(graph *resources.Graph) []connectionEdge {
	var edges []connectionEdge
	for _, res := range graph.ResourcesByType(esConnection.EventStreamConnectionResourceType) {
		data := res.Data()
		src, srcOK := data[esConnection.SourceKey].(*resources.PropertyRef)
		dst, dstOK := data[esConnection.DestinationKey].(*resources.PropertyRef)
		if !srcOK || !dstOK {
			continue
		}
		edges = append(edges, connectionEdge{sourceURN: src.URN, destinationURN: dst.URN})
	}
	return edges
}

// sourceDisabled reports whether a source graph resource is explicitly
// disabled; the handler resolves the spec's default before the graph is
// built, so the flag is always present.
func sourceDisabled(res *resources.Resource) bool {
	enabled, ok := res.Data()[esSource.EnabledKey].(bool)
	return ok && !enabled
}

// destinationDisabled reports whether a destination graph resource is
// disabled.
func destinationDisabled(res *resources.Resource) bool {
	destinationData, ok := res.RawData().(*destination.DestinationResource)
	return ok && !destinationData.Enabled
}

// connectionRef, sourceRef, and destinationRef build the JSON-pointer
// references for the connection entry at index and its endpoint fields.
func connectionRef(index int) string {
	return fmt.Sprintf("/connections/%d", index)
}

func sourceRef(index int) string {
	return connectionRef(index) + "/source"
}

func destinationRef(index int) string {
	return connectionRef(index) + "/destination"
}

// endpointID extracts the local id from a "#<kind>:<id>" endpoint reference.
// Malformed or wrong-kind refs return ok=false — the spec-syntax rule already
// reports those, so semantic checks skip the entry quietly.
func endpointID(ref, wantKind string) (string, bool) {
	// esConnection.ScalarRefRegex is the same pattern the handler parses with,
	// so semantic checks and parsing agree on what a reference looks like.
	matches := esConnection.ScalarRefRegex.FindStringSubmatch(strings.TrimSpace(ref))
	if matches == nil || matches[1] != wantKind {
		return "", false
	}
	return matches[2], true
}
