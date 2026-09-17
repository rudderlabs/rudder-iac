package connection

import (
	"fmt"
	"slices"
	"strings"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	esRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/rules/connection"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// primaryKeyExemptDefinition is the one source definition a connection may
// leave without a primary key — a source kind is not exempt just because an
// exemption exists for its warehouse.
const primaryKeyExemptDefinition = "s3"

// NewConnectionSemanticValidRule validates the cross-resource concerns of a
// rETL connection: that both endpoints exist, that the topology they form is
// allowed, and that the connection agrees with the flow the destination runs.
// retl/docs/connection-semantic-valid.docs.yaml enumerates the checks.
func NewConnectionSemanticValidRule(registry *definitions.Registry) rules.Rule {
	return prules.NewTypedRule(
		"retl/connection/semantic-valid",
		rules.Error,
		"retl connection endpoints must exist in the project and form a valid, compatible topology",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(retlConnection.ResourceKind),
			func(_ string, _ string, _ map[string]any, spec retlConnection.ConnectionsSpec, graph *resources.Graph) []rules.ValidationResult {
				return validateConnectionsSemantic(registry, spec, graph)
			},
		),
	)
}

// NewConnectionEnabledEndpointsRule warns when an enabled connection points at
// a disabled endpoint (V-C7): the API accepts it, but the connection will not
// sync until both endpoints are enabled.
func NewConnectionEnabledEndpointsRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/connection/enabled-endpoints-valid",
		rules.Warning,
		"an enabled retl connection needs both its endpoints enabled to sync",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(retlConnection.ResourceKind),
			validateEnabledEndpoints,
		),
	)
}

func validateConnectionsSemantic(
	registry *definitions.Registry,
	spec retlConnection.ConnectionsSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	edges := esRules.ProjectConnectionEdges(graph)

	var results []rules.ValidationResult
	for index, c := range spec.Connections {
		endpoints := resolveEndpoints(graph, c)

		results = append(results, validateEndpointsExist(index, endpoints)...)

		// Every check below is about the topology two existing endpoints form.
		// Run on an entry missing one of them they report on a shape the author
		// has not written yet — a pair that is only duplicated because the
		// dangling edge names the same URNs, or a destination that looks shared
		// because this connection's own source is absent — burying the one
		// error that has to be fixed first.
		if !endpoints.resolved() {
			continue
		}

		pair := esRules.ConnectionEdge{SourceURN: endpoints.sourceURN, DestinationURN: endpoints.destinationURN}
		results = append(results, esRules.ValidatePairUniqueness(edges, pair, connectionRef(index))...)
		results = append(results, validateDestinationSources(edges, index, endpoints)...)

		entry := connectionEntry{index: index, endpoints: endpoints, config: c.Config}
		results = append(results, validateDestinationCompatibility(registry, entry)...)
		results = append(results, validateSourceCapability(entry)...)
	}
	return results
}

// validateEnabledEndpoints (V-C7): a connection marked enabled should have both
// its source and destination enabled — otherwise it will never sync. Endpoints
// missing from the graph are the semantic-valid rule's concern and are skipped
// here.
var validateEnabledEndpoints = func(
	_ string,
	_ string,
	_ map[string]any,
	spec retlConnection.ConnectionsSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	var results []rules.ValidationResult
	for index, c := range spec.Connections {
		// enabled defaults to true when omitted, mirroring the handler.
		if c.Enabled != nil && !*c.Enabled {
			continue
		}

		endpoints := resolveEndpoints(graph, c)

		if res := endpoints.source; res != nil {
			// Every rETL source handler resolves the spec's default before the
			// graph is built, so the flag is always present.
			if enabled, ok := res.Data()[retlConnection.SourceEnabledKey].(bool); ok && !enabled {
				results = append(results, result(sourceRef(index), fmt.Sprintf(
					"connection '%s' is enabled but its source '%s' is disabled; the connection will not sync",
					c.LocalID, endpoints.sourceID,
				)))
			}
		}

		if res := endpoints.destination; res != nil {
			if data, ok := res.RawData().(*destination.DestinationResource); ok && !data.Enabled {
				results = append(results, result(destinationRef(index), fmt.Sprintf(
					"connection '%s' is enabled but its destination '%s' is disabled; the connection will not sync",
					c.LocalID, endpoints.destinationID,
				)))
			}
		}
	}
	return results
}

// connectionEndpoints is one connection entry's endpoint resolution, shared by
// every per-entry check: the parsed local ids and URNs, whether each ref parsed
// as the right family, and the graph resources (nil when absent from the
// project).
type connectionEndpoints struct {
	sourceID    string
	sourceURN   string
	sourceKind  retlConnection.SourceKind
	sourceRefOK bool
	source      *resources.Resource

	destinationID    string
	destinationURN   string
	destinationRefOK bool
	destination      *resources.Resource
}

// sourceDefinition is the warehouse definition the source resource names. The
// source's own spec rules require one before a graph is built, so it is empty
// only on a resource no check here can say anything about.
func (e connectionEndpoints) sourceDefinition() string {
	definition, _ := e.source.Data()[retlConnection.SourceDefinitionKey].(string)
	return definition
}

// connectionEntry is what the compatibility checks read about one connection
// entry whose endpoints both resolved. registered and flow are known only once
// the destination has passed its prerequisites, so only the checks behind those
// read them.
type connectionEntry struct {
	index      int
	endpoints  connectionEndpoints
	config     retlConnection.ConfigSpec
	registered *definitions.RegisteredDefinition
	flow       retlConnection.Flow
}

// resolved reports whether both endpoints are references this rule owns and
// resources the project actually holds — the precondition of every check that
// reasons about the topology rather than about one endpoint.
func (e connectionEndpoints) resolved() bool {
	return e.sourceRefOK && e.destinationRefOK && e.source != nil && e.destination != nil
}

// resolveEndpoints parses both endpoint refs of a connection entry and looks
// the endpoints up in the graph. Malformed or wrong-family refs stay unresolved
// — the spec-syntax rule already reports those, so semantic checks skip them
// quietly.
func resolveEndpoints(graph *resources.Graph, c retlConnection.ConnectionSpec) connectionEndpoints {
	var endpoints connectionEndpoints

	if kind, id, ok := retlConnection.RefID(c.Source); ok {
		if sourceKind, known := retlConnection.SourceKindByKind(kind); known {
			endpoints.sourceID = id
			endpoints.sourceKind = sourceKind
			endpoints.sourceURN = resources.URN(id, sourceKind.ResourceType)
			endpoints.sourceRefOK = true
			endpoints.source, _ = graph.GetResource(endpoints.sourceURN)
		}
	}
	if kind, id, ok := retlConnection.RefID(c.Destination); ok && kind == destination.DestinationSpecKind {
		endpoints.destinationID = id
		endpoints.destinationURN = resources.URN(id, destination.DestinationResourceType)
		endpoints.destinationRefOK = true
		endpoints.destination, _ = graph.GetResource(endpoints.destinationURN)
	}
	return endpoints
}

// validateEndpointsExist implements V-C1.
func validateEndpointsExist(index int, endpoints connectionEndpoints) []rules.ValidationResult {
	var results []rules.ValidationResult
	if endpoints.sourceRefOK && endpoints.source == nil {
		results = append(results, result(sourceRef(index), fmt.Sprintf(
			"rETL source '%s' not found in the project", endpoints.sourceID,
		)))
	}
	if endpoints.destinationRefOK && endpoints.destination == nil {
		results = append(results, result(destinationRef(index), fmt.Sprintf(
			"destination '%s' not found in the project", endpoints.destinationID,
		)))
	}
	return results
}

// validateDestinationSources enforces the two rules about who else may feed a
// destination: an event stream source may not share it (V-E1), and neither may
// a second rETL source (V-R1). Both are enforced in the webapp only, so the CLI
// is the last line of defense. The Customer.io audience exception to V-R1 is
// out of this release: that flow is rejected outright.
func validateDestinationSources(edges []esRules.ConnectionEdge, index int, endpoints connectionEndpoints) []rules.ValidationResult {
	esSourcePrefix := esSource.ResourceType + ":"

	var results []rules.ValidationResult
	for _, e := range edges {
		if e.DestinationURN != endpoints.destinationURN {
			continue
		}
		_, foreignID, _ := strings.Cut(e.SourceURN, ":")

		switch {
		case strings.HasPrefix(e.SourceURN, esSourcePrefix):
			results = append(results, result(destinationRef(index), fmt.Sprintf(
				"destination '%s' is also connected to event stream source '%s' in this project; a destination cannot receive from both event stream and rETL sources",
				endpoints.destinationID, foreignID,
			)))
		case e.SourceURN != endpoints.sourceURN:
			results = append(results, result(destinationRef(index), fmt.Sprintf(
				"destination '%s' is also connected to rETL source '%s' in this project; a destination can only receive from one rETL source",
				endpoints.destinationID, foreignID,
			)))
		}
	}
	return results
}

// validateDestinationCompatibility runs the checks that need the destination
// definition and, through it, the flow the connection runs — V-R2 among them,
// since the sync behaviour has to suit both endpoints. A prerequisite failure
// stops the rest: a check written in terms of a flow that could not be
// derived, or of a destination that takes no warehouse source at all, would
// only bury the error the author has to fix first. Independent prerequisites
// are still all reported before stopping.
func validateDestinationCompatibility(registry *definitions.Registry, entry connectionEntry) []rules.ValidationResult {
	// Destination resources always carry *destination.DestinationResource;
	// anything else is the destination provider's corruption, not this rule's
	// to report.
	destinationData, ok := entry.endpoints.destination.RawData().(*destination.DestinationResource)
	if !ok {
		return nil
	}

	// An unregistered (type, version) pair is the destination spec-syntax
	// rule's concern; skip quietly here.
	registered, err := registry.Get(destinationData.Type, destinationData.DefinitionVersion)
	if err != nil {
		return nil
	}

	// ClassifyFlow is the same derivation the backend runs, and its two refusals
	// are not equally final. A destination-specific integration is rejected
	// outright — nothing else said about it would hold, since its config is
	// never read as a generic one. An object aimed at a destination without the
	// visual mapper (V-R5) is instead independent of whether that destination
	// takes rETL sources at all (V-C4), so it waits until V-C4 has had its say
	// rather than hiding it.
	var (
		supported        = registered.SupportedSourceTypes()
		flow, flowErr    = retlConnection.ClassifyFlow(registered.APIType, registered.SupportsVisualMapper(), entry.config.Object)
		acceptsWarehouse = slices.Contains(supported, common.SourceTypeWarehouse)
	)

	if flowErr != nil && retlConnection.IsDestinationSpecificAPIType(registered.APIType) {
		return []rules.ValidationResult{result(destinationRef(entry.index), flowErr.Error())}
	}

	// V-C4: rETL sources reach a destination as warehouse sources, so the
	// definition has to declare that source type.
	var blocking []rules.ValidationResult
	if !acceptsWarehouse {
		blocking = append(blocking, result(destinationRef(entry.index), fmt.Sprintf(
			"destination '%s' (type '%s') does not accept rETL sources: source type '%s' is not among supported source types: %s",
			entry.endpoints.destinationID, destinationData.Type, common.SourceTypeWarehouse, strings.Join(supported, ", "),
		)))
	}
	if flowErr != nil {
		blocking = append(blocking, result(configRef(entry.index)+"/object", flowErr.Error()))
	}
	// Everything below is written in terms of a flow and a warehouse-capable
	// destination, so neither can be checked past these.
	if len(blocking) > 0 {
		return blocking
	}

	entry.registered, entry.flow = registered, flow

	results := validateDestinationConfig(entry, destinationData.Config)
	if flow == retlConnection.FlowObjectMapping {
		results = append(results, validateObjectMappings(entry.index, entry.config)...)
	} else {
		results = append(results, validateJSONMappings(entry.index, entry.config)...)
	}
	return append(results, validateSyncBehaviour(entry)...)
}

// validateDestinationConfig runs the destination-side config checks for a
// warehouse source that the event stream rules share (V-C5, V-C8), plus the
// object-mapping restriction on hyphenated destination names (V-R10).
func validateDestinationConfig(entry connectionEntry, config map[string]any) []rules.ValidationResult {
	results := esRules.ValidateDestinationConfig(
		entry.registered, destinationRef(entry.index), entry.endpoints.destinationID, common.SourceTypeWarehouse, config,
	)

	// The object mapping response mapper splits the destination name on "-", so
	// a hyphenated name cannot be put back together; the backend refuses the
	// combination rather than delivering to the wrong object.
	registered := entry.registered
	if entry.flow == retlConnection.FlowObjectMapping && (strings.Contains(registered.APIType, "-") || strings.Contains(registered.Type, "-")) {
		results = append(results, result(destinationRef(entry.index), fmt.Sprintf(
			"destination '%s' (type '%s', api type '%s') cannot be used with object mapping: its name contains a hyphen, which the backend cannot reconstruct",
			entry.endpoints.destinationID, registered.Type, registered.APIType,
		)))
	}

	return results
}

// validateObjectMappings covers the object mapping flow: constants and events
// belong to the JSON mapper alone (V-R5), the flow stores exactly one
// identifier (V-R4), its mappings may be absent (V-R9b), and a mapping aimed at
// an identifier target would not survive a round trip.
func validateObjectMappings(index int, config retlConnection.ConfigSpec) []rules.ValidationResult {
	var results []rules.ValidationResult

	if len(config.Constants) > 0 {
		results = append(results, result(configRef(index)+"/constants", "'constants' is not allowed with object mapping"))
	}
	if config.Event != nil {
		results = append(results, result(configRef(index)+"/event", "'event' is not allowed with object mapping"))
	}
	if len(config.Identifiers) != 1 {
		results = append(results, result(configRef(index)+"/identifiers", fmt.Sprintf(
			"object mapping supports a single identifier, found %d", len(config.Identifiers),
		)))
	}

	return append(results, reservedMappingTargets(index, config.Mappings, retlConnection.UserIDTarget, retlConnection.ExternalIDTarget)...)
}

// validateJSONMappings covers the JSON mapper flow: identifiers target the two
// reserved user identities (V-R4), mappings are what the connection delivers so
// they cannot be empty (V-R9b), and a mapping aimed at an identifier target
// would not survive a round trip.
func validateJSONMappings(index int, config retlConnection.ConfigSpec) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, identifier := range config.Identifiers {
		if identifier.To == retlConnection.UserIDTarget || identifier.To == retlConnection.AnonymousIDTarget {
			continue
		}
		results = append(results, result(fmt.Sprintf("%s/identifiers/%d/to", configRef(index), i), fmt.Sprintf(
			"'to' is not valid: a JSON mapping identifier must target '%s' or '%s'",
			retlConnection.UserIDTarget, retlConnection.AnonymousIDTarget,
		)))
	}

	if len(config.Mappings) == 0 {
		results = append(results, result(configRef(index)+"/mappings", "'mappings' is required with JSON mapping"))
	}

	return append(results, reservedMappingTargets(index, config.Mappings, retlConnection.UserIDTarget, retlConnection.AnonymousIDTarget)...)
}

// reservedMappingTargets is the round-trip guard: the backend folds a user
// mapping aimed at one of the flow's identifier targets into the identifiers,
// or consumes it outright, so it never comes back where it was written and the
// connection would diff on every apply.
func reservedMappingTargets(index int, mappings []retlConnection.MappingSpec, reserved ...string) []rules.ValidationResult {
	var results []rules.ValidationResult
	for i, mapping := range mappings {
		if !slices.Contains(reserved, mapping.To) {
			continue
		}
		results = append(results, result(fmt.Sprintf("%s/mappings/%d/to", configRef(index), i), fmt.Sprintf(
			"'to' is not valid: mapping to '%s' is reserved for identifiers", mapping.To,
		)))
	}
	return results
}

// validateSourceCapability runs the checks that depend only on what the
// source's warehouse can do: a primary key unless its definition is exempt
// (V-R11), SQL model support for a model source (V-R13) and sync settings
// support (V-R14). None of them reads the destination, so they run whatever
// state it is in rather than surfacing one fix at a time.
func validateSourceCapability(entry connectionEntry) []rules.ValidationResult {
	var (
		index            = entry.index
		endpoints        = entry.endpoints
		sourceDefinition = endpoints.sourceDefinition()
	)
	if sourceDefinition == "" {
		return nil
	}

	var results []rules.ValidationResult

	// V-R11: a sync needs a key to identify rows by, except for the source
	// definitions that let the backend derive one.
	primaryKey, _ := endpoints.source.Data()[retlConnection.SourcePrimaryKeyKey].(string)
	if primaryKey == "" && sourceDefinition != primaryKeyExemptDefinition {
		results = append(results, result(sourceRef(index), fmt.Sprintf(
			"rETL source '%s' declares no primary_key, which source definition '%s' requires to sync",
			endpoints.sourceID, sourceDefinition,
		)))
	}

	// Unknown metadata is reported rather than guessed around: guessing turns a
	// backend rejection into a locally passing validation.
	warehouse, known := retlConnection.WarehouseDefinition(sourceDefinition)
	if !known {
		return append(results, result(sourceRef(index), fmt.Sprintf(
			"rETL source '%s' uses source definition '%s', whose rETL capabilities this CLI version does not know; upgrade the CLI to validate this connection",
			endpoints.sourceID, sourceDefinition,
		)))
	}

	if endpoints.sourceKind.SourceType == retlClient.ModelSourceType && !warehouse.SupportsSQLModel {
		results = append(results, result(sourceRef(index), fmt.Sprintf(
			"rETL source '%s' is a SQL model, which source definition '%s' does not support",
			endpoints.sourceID, sourceDefinition,
		)))
	}

	if entry.config.SyncSettings != nil && !warehouse.SupportsSyncSettings {
		results = append(results, result(configRef(index)+"/sync_settings", fmt.Sprintf(
			"'sync_settings' is not allowed: source definition '%s' does not support sync settings", sourceDefinition,
		)))
	}

	return results
}

// validateSyncBehaviour (V-R2): the behaviour has to be one both endpoints
// accept, and the JSON mapper flow drops mirror from whatever the two agree on.
// An empty intersection is reported as such rather than falling back to a
// permissive set — that would let a connection the backend refuses pass here.
// A source whose capabilities are unknown is validateSourceCapability's to
// report, so there is nothing to intersect.
func validateSyncBehaviour(entry connectionEntry) []rules.ValidationResult {
	sourceDefinition := entry.endpoints.sourceDefinition()
	warehouse, known := retlConnection.WarehouseDefinition(sourceDefinition)
	if !known {
		return nil
	}

	var (
		reference           = configRef(entry.index) + "/sync_behaviour"
		destinationAccepted = entry.registered.SyncBehaviours()
	)

	// WarehouseDefinition hands back a clone, so filtering in place is safe.
	// "mirror" is offered for object mapping only, so the JSON mapper flow drops
	// it however the two endpoints feel about it.
	accepted := slices.DeleteFunc(warehouse.SyncBehaviours, func(behaviour string) bool {
		if entry.flow == retlConnection.FlowJSONMapper && behaviour == "mirror" {
			return true
		}
		return !slices.Contains(destinationAccepted, behaviour)
	})

	if len(accepted) == 0 {
		return []rules.ValidationResult{result(reference, fmt.Sprintf(
			"source definition '%s' and destination '%s' share no sync behaviour for the %s flow; the connection cannot sync",
			sourceDefinition, entry.endpoints.destinationID, entry.flow,
		))}
	}
	if slices.Contains(accepted, entry.config.SyncBehaviour) {
		return nil
	}

	return []rules.ValidationResult{result(reference, fmt.Sprintf(
		"'sync_behaviour' must be one of [%s] for source definition '%s' and destination '%s' on the %s flow",
		strings.Join(accepted, " "), sourceDefinition, entry.endpoints.destinationID, entry.flow,
	))}
}
