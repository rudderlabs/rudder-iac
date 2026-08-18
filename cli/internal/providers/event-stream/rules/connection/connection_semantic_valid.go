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

// connectionEdgeReaders maps each resource type whose graph entries are
// project connections to the reader that reduces one such resource to its
// endpoint URNs, feeding the topology checks (V-C3, V-E1). Only event stream
// connections exist as project resources today; the retl-connections kind
// registers its own type and reader here when it lands, so its field layout
// stays its own concern. Known limitation: these checks see only
// project-managed connections — a connection that exists remotely but is not
// in the project is invisible at validate time.
var connectionEdgeReaders = map[string]func(*resources.Resource) (connectionEdge, bool){
	esConnection.EventStreamConnectionResourceType: eventStreamConnectionEdge,
}

// connectionEdge is one project connection reduced to its endpoint URNs.
type connectionEdge struct {
	sourceURN      string
	destinationURN string
}

// eventStreamConnectionEdge reads the endpoint URNs off an
// event-stream-connection graph resource, whose data map carries both
// endpoints as PropertyRefs.
func eventStreamConnectionEdge(res *resources.Resource) (connectionEdge, bool) {
	data := res.Data()
	src, ok := data[esConnection.SourceKey].(*resources.PropertyRef)
	if !ok {
		return connectionEdge{}, false
	}
	dst, ok := data[esConnection.DestinationKey].(*resources.PropertyRef)
	if !ok {
		return connectionEdge{}, false
	}
	return connectionEdge{sourceURN: src.URN, destinationURN: dst.URN}, true
}

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
	for i, c := range spec.Connections {
		results = append(results, validateConnectionSemantic(registry, graph, edges, i, c)...)
	}
	return results
}

func validateConnectionSemantic(
	registry *definitions.Registry,
	graph *resources.Graph,
	edges []connectionEdge,
	index int,
	c esConnection.ConnectionSpec,
) []rules.ValidationResult {
	var results []rules.ValidationResult

	sourceID, sourceRefOK := endpointID(c.Source, esSource.ResourceKind)
	destinationID, destinationRefOK := endpointID(c.Destination, destination.DestinationSpecKind)

	// V-C1: both endpoints must resolve to project resources.
	var sourceRes, destinationRes *resources.Resource
	if sourceRefOK {
		var found bool
		if sourceRes, found = graph.GetResource(resources.URN(sourceID, esSource.ResourceType)); !found {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/connections/%d/source", index),
				Message:   fmt.Sprintf("event stream source '%s' not found in the project", sourceID),
			})
		}
	}
	if destinationRefOK {
		var found bool
		if destinationRes, found = graph.GetResource(resources.URN(destinationID, destination.DestinationResourceType)); !found {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/connections/%d/destination", index),
				Message:   fmt.Sprintf("destination '%s' not found in the project", destinationID),
			})
		}
	}

	if sourceRefOK && destinationRefOK {
		results = append(results, validatePairUniqueness(edges, index, sourceID, destinationID)...)
	}
	// Sharing a destination is only meaningful for a destination that exists;
	// a dangling ref already carries the V-C1 error above.
	if destinationRes != nil {
		results = append(results, validateDestinationFamily(edges, index, destinationID)...)
	}
	if sourceRes != nil && destinationRes != nil {
		results = append(results, validateSourceTypeCompatibility(registry, index, sourceID, sourceRes, destinationID, destinationRes)...)
	}

	return results
}

// validatePairUniqueness (V-C3): the same source–destination pair can only be
// connected once in the project. The count runs over every project
// connection, so duplicates are flagged whether they sit in this spec or in
// another one.
func validatePairUniqueness(edges []connectionEdge, index int, sourceID, destinationID string) []rules.ValidationResult {
	pair := connectionEdge{
		sourceURN:      resources.URN(sourceID, esSource.ResourceType),
		destinationURN: resources.URN(destinationID, destination.DestinationResourceType),
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
		Reference: fmt.Sprintf("/connections/%d", index),
		Message: fmt.Sprintf(
			"source '%s' and destination '%s' are connected more than once in the project; a source-destination pair can only be connected once",
			sourceID, destinationID,
		),
	}}
}

// validateDestinationFamily (V-E1): an event stream source cannot share a
// destination with a rETL source — every project connection to this
// destination must come from an event stream source. This is enforced only in
// the webapp today, so the CLI is the last line of defense.
func validateDestinationFamily(edges []connectionEdge, index int, destinationID string) []rules.ValidationResult {
	destinationURN := resources.URN(destinationID, destination.DestinationResourceType)
	sourcePrefix := esSource.ResourceType + ":"

	var results []rules.ValidationResult
	for _, e := range edges {
		if e.destinationURN != destinationURN || strings.HasPrefix(e.sourceURN, sourcePrefix) {
			continue
		}
		_, foreignID, _ := strings.Cut(e.sourceURN, ":")
		results = append(results, rules.ValidationResult{
			Reference: fmt.Sprintf("/connections/%d/destination", index),
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
	sourceID string,
	sourceRes *resources.Resource,
	destinationID string,
	destinationRes *resources.Resource,
) []rules.ValidationResult {
	sourceType, _ := sourceRes.Data()[esSource.SourceDefinitionKey].(string)
	destinationData, ok := destinationRes.RawData().(*destination.DestinationResource)
	if sourceType == "" || !ok {
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
			Reference: fmt.Sprintf("/connections/%d/destination", index),
			Message: fmt.Sprintf(
				"destination '%s' (type '%s') does not support source '%s': source type '%s' is not among supported source types: %s",
				destinationID, destinationData.Type, sourceID, token, strings.Join(supported, ", "),
			),
		}}
	}

	missing := missingRequiredConfigKeys(registered, token, destinationData.Config)
	if len(missing) > 0 {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("/connections/%d/destination", index),
			Message: fmt.Sprintf(
				"destination '%s' config is missing fields required to connect a '%s' source: %s",
				destinationID, token, strings.Join(missing, ", "),
			),
		}}
	}

	return nil
}

// missingRequiredConfigKeys returns the definition-required config keys for
// the given source type that the destination's config does not carry.
// Source-type-scoped keys (connection_mode, use_native_sdk) live under
// config.<key>.<source type> and are reported with that path.
func missingRequiredConfigKeys(
	registered *definitions.RegisteredDefinition,
	sourceType string,
	config map[string]any,
) []string {
	var missing []string
	for _, key := range registered.SupportedSourcesValidation(sourceType) {
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
	for i, c := range spec.Connections {
		// enabled defaults to true when omitted, mirroring the handler.
		if c.Enabled != nil && !*c.Enabled {
			continue
		}

		if sourceID, ok := endpointID(c.Source, esSource.ResourceKind); ok {
			if res, found := graph.GetResource(resources.URN(sourceID, esSource.ResourceType)); found {
				if enabled, ok := res.Data()[esSource.EnabledKey].(bool); ok && !enabled {
					results = append(results, rules.ValidationResult{
						Reference: fmt.Sprintf("/connections/%d/source", i),
						Message: fmt.Sprintf(
							"connection '%s' is enabled but its source '%s' is disabled; the connection will not deliver events",
							c.LocalID, sourceID,
						),
					})
				}
			}
		}

		if destinationID, ok := endpointID(c.Destination, destination.DestinationSpecKind); ok {
			if res, found := graph.GetResource(resources.URN(destinationID, destination.DestinationResourceType)); found {
				if destinationData, ok := res.RawData().(*destination.DestinationResource); ok && !destinationData.Enabled {
					results = append(results, rules.ValidationResult{
						Reference: fmt.Sprintf("/connections/%d/destination", i),
						Message: fmt.Sprintf(
							"connection '%s' is enabled but its destination '%s' is disabled; the connection will not deliver events",
							c.LocalID, destinationID,
						),
					})
				}
			}
		}
	}
	return results
}

// projectConnectionEdges reduces every project connection in the graph to its
// endpoint URN pair for the topology checks.
func projectConnectionEdges(graph *resources.Graph) []connectionEdge {
	var edges []connectionEdge
	for resourceType, edgeOf := range connectionEdgeReaders {
		for _, res := range graph.ResourcesByType(resourceType) {
			if edge, ok := edgeOf(res); ok {
				edges = append(edges, edge)
			}
		}
	}
	return edges
}

// endpointID extracts the local id from a "#<kind>:<id>" endpoint reference.
// Malformed or wrong-kind refs return ok=false — the spec-syntax rule already
// reports those, so semantic checks skip the entry quietly.
func endpointID(ref, wantKind string) (string, bool) {
	matches := scalarRefRegex.FindStringSubmatch(strings.TrimSpace(ref))
	if matches == nil || matches[1] != wantKind {
		return "", false
	}
	return matches[2], true
}
