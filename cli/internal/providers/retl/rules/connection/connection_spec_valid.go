package connection

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// mappedToDestinationKey is the constant key the backend writes itself
// (config-backend src/modules/retl/api-gateway/connection-config/constants.ts),
// so a user constant claiming it is overwritten rather than delivered.
const mappedToDestinationKey = "context.mappedToDestination"

// validateRawConnectionsSpec reads the spec map strictly before checking it.
// The handler's mapstructure decode rejects unknown keys too, but only once
// loading starts — after validation has already passed the spec — so a misspelt
// field would surface as a generic load failure instead of a diagnostic
// pointing at the key.
func validateRawConnectionsSpec(
	_ string,
	_ string,
	_ map[string]any,
	raw map[string]any,
) []rules.ValidationResult {
	var (
		spec retlConnection.ConnectionsSpec
		md   mapstructure.Metadata
	)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &spec, Metadata: &md})
	if err != nil {
		return []rules.ValidationResult{{Message: fmt.Sprintf("creating spec decoder: %v", err)}}
	}

	// A field that failed to decode keeps its zero value, so the checks below
	// would report the decoder's spec rather than the author's: return the
	// decode verdicts alone.
	if err := decoder.Decode(raw); err != nil {
		return decodeErrorResults(err)
	}

	return append(unknownFieldResults(md.Unused), validateConnectionsSpec(spec)...)
}

// mapstructurePathIndex matches the "[0]" element suffixes mapstructure writes
// into the field paths it reports.
var mapstructurePathIndex = regexp.MustCompile(`\[(\d+)\]`)

// unknownFieldResults reports every key the spec type does not carry. The
// decoder collects them in map iteration order, so they are sorted to keep the
// diagnostics stable between runs.
func unknownFieldResults(unused []string) []rules.ValidationResult {
	slices.Sort(unused)

	results := make([]rules.ValidationResult, 0, len(unused))
	for _, path := range unused {
		results = append(results, result(
			jsonPointer(path),
			fmt.Sprintf("unknown field %q", fieldName(path)),
		))
	}
	return results
}

// decodeErrorResults turns a decode failure into one result per field that
// could not be read. mapstructure joins the field errors into a tree and wraps
// each one in a DecodeError carrying the field's path, so flattening the tree
// is what turns a single root error into per-field references.
func decodeErrorResults(err error) []rules.ValidationResult {
	decodeErrors := flattenDecodeErrors(err)
	if len(decodeErrors) == 0 {
		return []rules.ValidationResult{{Message: fmt.Sprintf("decoding spec: %v", err)}}
	}

	results := make([]rules.ValidationResult, 0, len(decodeErrors))
	for _, decodeErr := range decodeErrors {
		results = append(results, result(
			jsonPointer(decodeErr.Name()),
			fmt.Sprintf("'%s' is not valid: %v", fieldName(decodeErr.Name()), decodeErr.Unwrap()),
		))
	}
	return results
}

// flattenDecodeErrors walks the joined error tree and collects the outermost
// DecodeError on every branch: that is the node that knows the field path,
// while what it wraps is the bare reason.
func flattenDecodeErrors(err error) []*mapstructure.DecodeError {
	if decodeErr, ok := err.(*mapstructure.DecodeError); ok {
		return []*mapstructure.DecodeError{decodeErr}
	}

	switch wrapper := err.(type) {
	case interface{ Unwrap() []error }:
		var collected []*mapstructure.DecodeError
		for _, wrapped := range wrapper.Unwrap() {
			collected = append(collected, flattenDecodeErrors(wrapped)...)
		}
		return collected
	case interface{ Unwrap() error }:
		return flattenDecodeErrors(wrapper.Unwrap())
	}

	return nil
}

// jsonPointer converts a mapstructure field path ("connections[0].config.typo")
// into the JSON pointer the rule engine reports against
// ("/connections/0/config/typo").
func jsonPointer(path string) string {
	return "/" + strings.ReplaceAll(mapstructurePathIndex.ReplaceAllString(path, "/$1"), ".", "/")
}

// fieldName is the field a mapstructure path is about: its last named segment,
// so an element path ("connections[0]") names the field holding the element
// rather than the index.
func fieldName(path string) string {
	named := mapstructurePathIndex.ReplaceAllString(path, "")
	if index := strings.LastIndex(named, "."); index != -1 {
		return named[index+1:]
	}
	return named
}

func validateConnectionsSpec(spec retlConnection.ConnectionsSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	// The root type resolves cross-field tag params to their json display names.
	results := funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))

	for index, c := range spec.Connections {
		results = append(results, validateEndpointRef(
			sourceRef(index), "source", c.Source,
			retlConnection.SourceKindRefForms(), "a rETL source",
			func(kind string) bool { _, ok := retlConnection.SourceKindByKind(kind); return ok },
		)...)
		results = append(results, validateEndpointRef(
			destinationRef(index), "destination", c.Destination,
			fmt.Sprintf("#%s:<id>", destination.DestinationSpecKind), "a destination",
			func(kind string) bool { return kind == destination.DestinationSpecKind },
		)...)
		results = append(results, validateCron(index, c.Config.Schedule)...)
		results = append(results, validateCursorColumn(index, c.Config)...)
		results = append(results, validateConstants(index, c.Config.Constants)...)
		results = append(results, validateObject(index, c.Config.Object)...)
	}

	return results
}

// validateEndpointRef checks one endpoint reference (V-C2, and on the source
// side V-C6). An empty ref is skipped — the struct validator's required tag
// already reports it. A malformed ref and one pointing at a kind the endpoint
// does not accept fail differently: only the second can name the kind the
// author actually wrote.
func validateEndpointRef(reference, field, ref, forms, label string, accepts func(string) bool) []rules.ValidationResult {
	if ref == "" {
		return nil
	}

	kind, _, ok := endpointRef(ref)
	if !ok {
		return []rules.ValidationResult{result(reference, fmt.Sprintf(
			"'%s' is invalid: must be of pattern %s", field, forms,
		))}
	}

	if !accepts(kind) {
		return []rules.ValidationResult{result(reference, fmt.Sprintf(
			"'%s' must reference %s (%s), got a '%s' reference", field, label, forms, kind,
		))}
	}

	return nil
}

// validateCron (V-R3) is the part of the schedule contract struct tags cannot
// express: a cron expression has to parse against the supported grammar and
// stay above the frequency floor. Only verdicts this severity owns are reported
// — an unsupported dialect is the cron-expression warning rule's.
func validateCron(index int, schedule retlConnection.ScheduleSpec) []rules.ValidationResult {
	if schedule.Type != "cron" || schedule.CronExpression == "" {
		return nil
	}

	check := CheckCron(schedule.CronExpression)
	if check.Status != CronInvalid && check.Status != CronTooFrequent {
		return nil
	}

	return []rules.ValidationResult{result(
		scheduleRef(index)+"/cron_expression",
		fmt.Sprintf("'cron_expression' is not valid: %s", check.Reason),
	)}
}

// validateCursorColumn (V-R7): only an upsert sync tracks a cursor, so any
// other behaviour would carry the column without ever reading it.
func validateCursorColumn(index int, config retlConnection.ConfigSpec) []rules.ValidationResult {
	if config.CursorColumn == "" || config.SyncBehaviour == "upsert" {
		return nil
	}
	return []rules.ValidationResult{result(
		configRef(index)+"/cursor_column",
		fmt.Sprintf("'cursor_column' is not allowed when 'sync_behaviour' is %s", config.SyncBehaviour),
	)}
}

// validateConstants (V-R9a): the backend owns one constant key, so a user
// constant claiming it is dropped rather than delivered.
func validateConstants(index int, constants []retlConnection.ConstantSpec) []rules.ValidationResult {
	var results []rules.ValidationResult
	for i, constant := range constants {
		if constant.Key != mappedToDestinationKey {
			continue
		}
		results = append(results, result(
			fmt.Sprintf("%s/constants/%d/key", configRef(index), i),
			fmt.Sprintf("'key' is not valid: %q is reserved by the backend", mappedToDestinationKey),
		))
	}
	return results
}

// validateObject (V-R10): an omitted object is a JSON mapper connection and is
// valid; a declared one names a destination object and has to be usable as
// written, since the backend matches it verbatim.
func validateObject(index int, object *string) []rules.ValidationResult {
	if object == nil {
		return nil
	}

	reference := configRef(index) + "/object"
	switch {
	case *object == "":
		return []rules.ValidationResult{result(reference, "'object' must not be empty")}
	case strings.TrimSpace(*object) != *object:
		return []rules.ValidationResult{result(reference, "'object' must not have leading or trailing whitespace")}
	}

	return nil
}

func result(reference, message string) rules.ValidationResult {
	return rules.ValidationResult{Reference: reference, Message: message}
}

// connectionRef and the refs below build the JSON-pointer references for the
// connection entry at index and its fields; the rule engine prefixes "/spec".
func connectionRef(index int) string {
	return fmt.Sprintf("/connections/%d", index)
}

func sourceRef(index int) string {
	return connectionRef(index) + "/source"
}

func destinationRef(index int) string {
	return connectionRef(index) + "/destination"
}

func configRef(index int) string {
	return connectionRef(index) + "/config"
}

func scheduleRef(index int) string {
	return configRef(index) + "/schedule"
}

func NewConnectionSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/connection/spec-syntax-valid",
		rules.Error,
		"retl connection spec syntax must be valid",
		rules.Examples{},
		prules.NewPatternValidator(
			prules.V1VersionPatterns(retlConnection.ResourceKind),
			validateRawConnectionsSpec,
		),
	)
}
