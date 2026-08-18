package connection

import (
	"fmt"
	"regexp"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// scalarRefRegex matches any well-formed scalar reference "#<kind>:<id>" so a
// ref pointing at the wrong kind can be told apart from a malformed one. The
// id side deliberately accepts anything non-empty ((?s) lets it span newlines)
// — endpoint local ids carry no charset restriction and the handler's parser
// is equally lenient, so a stricter pattern here could reject a spec that
// would apply cleanly.
var scalarRefRegex = regexp.MustCompile(`(?s)^#([a-zA-Z0-9_-]+):(.+)$`)

var validateConnectionsSpec = func(
	_ string,
	_ string,
	_ map[string]any,
	spec esConnection.ConnectionsSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)

	for i, c := range spec.Connections {
		results = append(results, validateEndpointRef(
			i, "source", c.Source, esSource.ResourceKind, "an event stream source",
		)...)
		results = append(results, validateEndpointRef(
			i, "destination", c.Destination, destination.DestinationSpecKind, "a destination",
		)...)
	}

	return results
}

// validateEndpointRef checks one endpoint reference of the connection at index.
// An empty ref is skipped — the struct validator's required tag already reports
// it. The ref must point at the endpoint's kind (V-C2); on the source side this
// doubles as the family rule (V-C6), since event-stream-connections only accept
// event stream source references.
func validateEndpointRef(index int, field, ref, wantKind, wantLabel string) []rules.ValidationResult {
	if ref == "" {
		return nil
	}

	reference := fmt.Sprintf("/connections/%d/%s", index, field)

	matches := scalarRefRegex.FindStringSubmatch(strings.TrimSpace(ref))
	if matches == nil {
		return []rules.ValidationResult{{
			Reference: reference,
			Message:   fmt.Sprintf("'%s' is invalid: must be of pattern #%s:<id>", field, wantKind),
		}}
	}

	if matches[1] != wantKind {
		return []rules.ValidationResult{{
			Reference: reference,
			Message: fmt.Sprintf(
				"'%s' must reference %s (#%s:<id>), got a '%s' reference",
				field, wantLabel, wantKind, matches[1],
			),
		}}
	}

	return nil
}

func NewConnectionSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"event-stream/connection/spec-syntax-valid",
		rules.Error,
		"event stream connection spec syntax must be valid",
		rules.Examples{},
		prules.NewPatternValidator(
			prules.V1VersionPatterns(esConnection.EventStreamConnectionResourceKind),
			validateConnectionsSpec,
		),
	)
}
