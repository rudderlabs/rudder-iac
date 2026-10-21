package specdocs

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

// listIndex stands in for a list item's index in a secret's variable name,
// which is only known once the spec writes the list.
const listIndex = "<N>"

// DestinationDoc documents one destination type for one definition version.
type DestinationDoc struct {
	SchemaVersion     int    `json:"schemaVersion"`
	CLIVersion        string `json:"cliVersion"`
	Type              string `json:"type"`
	DisplayName       string `json:"displayName"`
	APIType           string `json:"apiType"`
	DefinitionVersion int64  `json:"definitionVersion"`
	Verified          bool   `json:"verified"`
	// Enable lists the environment variables an unverified type needs.
	Enable map[string]string `json:"enable,omitempty"`
	// Source is the source the document was narrowed to.
	Source *SourceRef `json:"source,omitempty"`
	// ConnectionModes is keyed by source token.
	ConnectionModes  map[string]ConnectionModes `json:"connectionModes"`
	SecretKeys       []string                   `json:"secretKeys"`
	Fields           []Field                    `json:"fields"`
	Patterns         map[string]Pattern         `json:"patterns"`
	CustomValidators []string                   `json:"customValidators"`
	// SpecTemplate is a destination spec holding the keys every spec of this
	// type needs, present when the document names both a source and an ID. It
	// leaves out keys required only by a requiredIf or requiredUnless rule;
	// such a rule on a key the spec omits is checked against that key's
	// default, so a caller also adds the keys whose conditions the defaults
	// meet.
	SpecTemplate string `json:"specTemplate,omitempty"`
}

// SourceRef is a source spec type and the token it connects as.
type SourceRef struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// ConnectionModes are the modes a source token may connect in, and the keys
// each mode requires.
type ConnectionModes struct {
	Modes        []string            `json:"modes"`
	RequiredKeys map[string][]string `json:"requiredKeys"`
}

// Field is one config key. Required holds only for top-level keys; a nested
// key's own required rule applies once its parent block or list item is
// written, which RequiredWhenParentPresent says. Var is a secret's variable
// name and Slot the text a spec writes for it; in both, "<N>" stands for the
// item's zero-based index in a list on the key's path, so item 0 of
// headers[].to is SHOP_HEADERS_0_TO.
type Field struct {
	Key                       string      `json:"key"`
	Type                      string      `json:"type"`
	Required                  bool        `json:"required,omitempty"`
	RequiredWhenParentPresent bool        `json:"requiredWhenParentPresent,omitempty"`
	RequiredIf                []Condition `json:"requiredIf,omitempty"`
	RequiredUnless            []Condition `json:"requiredUnless,omitempty"`
	RequiredWith              []string    `json:"requiredWith,omitempty"`
	ExcludedWith              []string    `json:"excludedWith,omitempty"`
	ExcludedIf                []Condition `json:"excludedIf,omitempty"`
	CheckedBy                 []string    `json:"checkedBy,omitempty"`
	Enum                      []any       `json:"enum,omitempty"`
	OpenEnum                  bool        `json:"openEnum,omitempty"`
	Default                   any         `json:"default,omitempty"`
	DefaultRequiresParent     bool        `json:"defaultRequiresParent,omitempty"`
	Pattern                   string      `json:"pattern,omitempty"`
	AcceptsTemplate           bool        `json:"acceptsTemplate,omitempty"`
	Min                       *int        `json:"min,omitempty"`
	Max                       *int        `json:"max,omitempty"`
	OnlyForSourceTypes        []string    `json:"onlyForSourceTypes,omitempty"`
	Secret                    bool        `json:"secret,omitempty"`
	Var                       string      `json:"var,omitempty"`
	Slot                      string      `json:"slot,omitempty"`
}

// Condition is one key and value of a requiredIf, requiredUnless or excludedIf
// rule. requiredIf and excludedIf apply when every condition holds,
// requiredUnless unless any does.
type Condition struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type Pattern struct {
	Regex   string `json:"regex"`
	Reject  string `json:"reject,omitempty"`
	Message string `json:"message"`
}

// DestinationList lists the destination types.
type DestinationList struct {
	SchemaVersion int                  `json:"schemaVersion"`
	CLIVersion    string               `json:"cliVersion"`
	Destinations  []DestinationSummary `json:"destinations"`
}

// DestinationSummary is one type in the listing. SourceTypes holds source
// tokens, the keys of connectionModes, not the source spec types --source
// takes; `docs source` maps one to the other.
type DestinationSummary struct {
	Type               string            `json:"type"`
	DisplayName        string            `json:"displayName"`
	APIType            string            `json:"apiType"`
	DefinitionVersions []int64           `json:"definitionVersions"`
	Verified           bool              `json:"verified"`
	SourceTypes        []string          `json:"sourceTypes"`
	Enable             map[string]string `json:"enable,omitempty"`
}

// Destinations documents the definitions of a registry. The registry should
// hold the unverified definitions too, so every compiled type can be read.
type Destinations struct {
	registry *definitions.Registry
	verified func(destType string, version int64) bool
}

// NewDestinations documents registry, marking each definition with verified.
func NewDestinations(registry *definitions.Registry, verified func(destType string, version int64) bool) *Destinations {
	return &Destinations{registry: registry, verified: verified}
}

// ListOptions narrow the listing.
type ListOptions struct {
	CLIVersion string
	// Source keeps only the types that accept this source spec type.
	Source string
}

// List lists every type, at its latest definition version.
func (d *Destinations) List(opts ListOptions) (DestinationList, error) {
	var token string
	if opts.Source != "" {
		ref, err := resolveSource(opts.Source)
		if err != nil {
			return DestinationList{}, err
		}
		token = ref.Token
	}

	list := DestinationList{
		SchemaVersion: SchemaVersion,
		CLIVersion:    opts.CLIVersion,
		Destinations:  []DestinationSummary{},
	}
	for _, destType := range d.registry.SupportedTypes() {
		versions, err := d.registry.Versions(destType)
		if err != nil {
			return DestinationList{}, fmt.Errorf("listing versions of %s: %w", destType, err)
		}
		latest, err := d.registry.Get(destType, versions[len(versions)-1])
		if err != nil {
			return DestinationList{}, fmt.Errorf("reading %s: %w", destType, err)
		}

		sourceTypes := latest.SupportedSourceTypes()
		if token != "" && !slices.Contains(sourceTypes, token) {
			continue
		}
		slices.Sort(sourceTypes)

		summary := DestinationSummary{
			Type:               latest.Type,
			DisplayName:        latest.DisplayName,
			APIType:            latest.APIType,
			DefinitionVersions: versions,
			Verified:           d.verified(latest.Type, latest.Version),
			SourceTypes:        sourceTypes,
		}
		if summary.SourceTypes == nil {
			summary.SourceTypes = []string{}
		}
		if !summary.Verified {
			summary.Enable = unverifiedEnable()
		}
		list.Destinations = append(list.Destinations, summary)
	}
	return list, nil
}

// DescribeOptions pick the definition and what to fill in.
type DescribeOptions struct {
	CLIVersion string
	Type       string
	// Version is the definition version; zero means the latest.
	Version int64
	// Source narrows the document to the keys and connection modes of one
	// source spec type.
	Source string
	// ID is the destination spec ID secret variable names derive from.
	ID string
}

// Describe documents one destination type.
func (d *Destinations) Describe(opts DescribeOptions) (DestinationDoc, error) {
	registered, err := d.lookup(opts.Type, opts.Version)
	if err != nil {
		return DestinationDoc{}, err
	}
	desc := registered.Describe()

	var source *SourceRef
	if opts.Source != "" {
		ref, err := resolveSource(opts.Source)
		if err != nil {
			return DestinationDoc{}, err
		}
		if !slices.Contains(desc.SourceTypes, ref.Token) {
			return DestinationDoc{}, unsupportedSource(desc, ref)
		}
		source = &ref
	}

	if opts.ID != "" {
		if err := validateID(opts.ID, desc.Fields); err != nil {
			return DestinationDoc{}, err
		}
	}

	doc := DestinationDoc{
		SchemaVersion:     SchemaVersion,
		CLIVersion:        opts.CLIVersion,
		Type:              desc.Type,
		DisplayName:       desc.DisplayName,
		APIType:           desc.APIType,
		DefinitionVersion: desc.Version,
		Verified:          d.verified(desc.Type, desc.Version),
		Source:            source,
		ConnectionModes:   connectionModes(desc, source),
		SecretKeys:        desc.SecretKeys,
		Fields:            []Field{},
		Patterns:          map[string]Pattern{},
		CustomValidators:  desc.CustomValidators,
	}
	if !doc.Verified {
		doc.Enable = unverifiedEnable()
	}
	// A definition may declare SecretKeys as an empty list, which copies to
	// nil; the contract prints every list as an array, never null.
	if doc.SecretKeys == nil {
		doc.SecretKeys = []string{}
	}

	for _, f := range desc.Fields {
		if source != nil && len(f.OnlyForSourceTypes) > 0 && !slices.Contains(f.OnlyForSourceTypes, source.Token) {
			continue
		}
		field := docField(f, opts.ID)
		if f.Pattern != "" {
			pattern := desc.Patterns[f.Pattern]
			doc.Patterns[f.Pattern] = Pattern{Regex: pattern.Regex, Reject: pattern.Reject, Message: pattern.Message}
		}
		doc.Fields = append(doc.Fields, field)
	}

	if source != nil && opts.ID != "" {
		template, err := specTemplate(doc, opts.ID)
		if err != nil {
			return DestinationDoc{}, err
		}
		doc.SpecTemplate = template
	}
	return doc, nil
}

func (d *Destinations) lookup(destType string, version int64) (*definitions.RegisteredDefinition, error) {
	versions, err := d.registry.Versions(destType)
	if err != nil {
		return nil, &Error{
			Code:        CodeUnknownType,
			Message:     fmt.Sprintf("destination type %q is not supported", destType),
			Suggestions: d.suggestTypes(destType),
			ValidTypes:  d.registry.SupportedTypes(),
		}
	}
	if version == 0 {
		version = versions[len(versions)-1]
	}
	registered, err := d.registry.Get(destType, version)
	if err != nil {
		return nil, &Error{
			Code:          CodeUnknownDefinitionVersion,
			Message:       fmt.Sprintf("destination type %q has no definition version %d", destType, version),
			Suggestions:   []string{},
			ValidVersions: versions,
		}
	}
	return registered, nil
}

func (d *Destinations) suggestTypes(input string) []string {
	var candidates []candidate
	for _, registered := range d.registry.Definitions() {
		candidates = append(candidates, candidate{value: registered.Type, aliases: []string{registered.DisplayName}})
	}
	return suggest(input, candidates)
}

func connectionModes(desc definitions.Description, source *SourceRef) map[string]ConnectionModes {
	out := map[string]ConnectionModes{}
	for _, token := range desc.SourceTypes {
		if source != nil && token != source.Token {
			continue
		}
		requiredKeys := desc.ConnectionRequiredKeys[token]
		if requiredKeys == nil {
			requiredKeys = map[string][]string{}
		}
		out[token] = ConnectionModes{Modes: desc.ConnectionModes[token], RequiredKeys: requiredKeys}
	}
	return out
}

func docField(f definitions.Field, id string) Field {
	field := Field{
		Key:                f.Key,
		Type:               f.Type,
		RequiredIf:         docConditions(f.RequiredIf),
		RequiredUnless:     docConditions(f.RequiredUnless),
		RequiredWith:       f.RequiredWith,
		ExcludedWith:       f.ExcludedWith,
		ExcludedIf:         docConditions(f.ExcludedIf),
		CheckedBy:          f.CheckedBy,
		Enum:               f.Enum,
		OpenEnum:           f.OpenEnum,
		Pattern:            f.Pattern,
		AcceptsTemplate:    f.AcceptsTemplate,
		Min:                f.Min,
		Max:                f.Max,
		OnlyForSourceTypes: f.OnlyForSourceTypes,
		Secret:             f.Secret,
	}
	if f.Required {
		field.Required = !f.Nested()
		field.RequiredWhenParentPresent = f.Nested()
	}
	if f.HasDefault {
		field.Default = f.Default
		field.DefaultRequiresParent = f.Nested()
	}
	if f.Secret && id != "" {
		field.Var = secret.VariableName(id, varSegments(f.Key, listIndex)...)
		field.Slot = slot(field.Var)
	}
	return field
}

func docConditions(conditions []definitions.Condition) []Condition {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]Condition, 0, len(conditions))
	for _, c := range conditions {
		out = append(out, Condition{Key: c.Key, Value: c.Value})
	}
	return out
}

// varSegments splits a field key into the segments secret.VariableName takes,
// with index in place of each list item's position.
func varSegments(key, index string) []string {
	var segments []string
	for _, part := range strings.Split(key, ".") {
		if name, isList := strings.CutSuffix(part, "[]"); isList {
			segments = append(segments, name, index)
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

// slot is the text a spec writes for a secret. Single quotes match what
// import writes and keep a value holding a double quote intact.
func slot(varName string) string {
	return "'" + secret.Reference(varName) + "'"
}

// idAdvice is the rule both checks on an ID add up to.
const idAdvice = "use an ID of 2 to 100 letters, digits, underscores and hyphens that does not start with a digit"

// validateID checks what id becomes: the template's display_name, and the
// secret variable names. A list index is a number, so "0" stands in for it
// here.
func validateID(id string, fields []definitions.Field) error {
	if !funcs.MatchPattern(destination.DisplayNamePattern, id) {
		pattern, _ := funcs.LookupPattern(destination.DisplayNamePattern)
		return invalidID(id, fmt.Sprintf("--id %q is also the spec's display_name, which %s; %s", id, pattern.Message, idAdvice))
	}

	names := []string{secret.VariableName(id)}
	for _, f := range fields {
		if f.Secret {
			names = append(names, secret.VariableName(id, varSegments(f.Key, "0")...))
		}
	}
	for _, name := range names {
		if varsubst.IsValidVariableName(name) {
			continue
		}
		return invalidID(id, fmt.Sprintf("--id %q gives the secret variable name %q, which does not match ^[A-Za-z_][A-Za-z0-9_]*$; %s",
			id, name, idAdvice))
	}
	return nil
}

func invalidID(id, message string) error {
	e := &Error{Code: CodeInvalidID, Message: message, Suggestions: []string{}}
	if fixed := suggestID(id); fixed != "" {
		e.Suggestions = []string{fixed}
	}
	return e
}

// suggestID lower-cases id, turns every other character into a hyphen and
// drops a leading run that cannot start a name.
func suggestID(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	fixed := strings.TrimLeft(b.String(), "-0123456789")
	if !funcs.MatchPattern(destination.DisplayNamePattern, fixed) || !varsubst.IsValidVariableName(secret.VariableName(fixed)) {
		return ""
	}
	return fixed
}

func unsupportedSource(desc definitions.Description, ref SourceRef) error {
	var accepted []string
	for _, sourceType := range sortedSourceTypes() {
		if slices.Contains(desc.SourceTypes, common.SourceTypeToken(sourceType, "")) {
			accepted = append(accepted, sourceType)
		}
	}
	return &Error{
		Code: CodeUnsupportedSource,
		Message: fmt.Sprintf("destination type %q does not accept %s sources, which connect as %q",
			desc.Type, ref.Type, ref.Token),
		Suggestions: []string{},
		ValidTypes:  accepted,
	}
}
