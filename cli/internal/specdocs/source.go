package specdocs

import (
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
)

// SourceList lists the event stream source spec types and the token each
// connects to a destination as.
type SourceList struct {
	SchemaVersion int         `json:"schemaVersion"`
	CLIVersion    string      `json:"cliVersion"`
	Sources       []SourceRef `json:"sources"`
}

// ListSources lists the types a source spec's `type` accepts.
func ListSources(cliVersion string) SourceList {
	list := SourceList{SchemaVersion: SchemaVersion, CLIVersion: cliVersion, Sources: []SourceRef{}}
	for _, sourceType := range sortedSourceTypes() {
		list.Sources = append(list.Sources, SourceRef{Type: sourceType, Token: common.SourceTypeToken(sourceType, "")})
	}
	return list
}

// IsSourceType reports whether sourceType is a value a source spec's `type`
// accepts.
func IsSourceType(sourceType string) bool {
	return slices.Contains(source.SourceTypes(), sourceType)
}

func sortedSourceTypes() []string {
	types := source.SourceTypes()
	slices.Sort(types)
	return types
}

// ValidateSource returns the unknown_source error for a --source value that
// is not a source spec type, or nil.
func ValidateSource(sourceType string) error {
	_, err := resolveSource(sourceType)
	return err
}

// resolveSource accepts only a source spec's own type values. A token such
// as "web" is rejected: SourceTypeToken maps every unknown value to "cloud",
// so accepting tokens would silently mean something else.
func resolveSource(sourceType string) (SourceRef, error) {
	if !IsSourceType(sourceType) {
		return SourceRef{}, &Error{
			Code:        CodeUnknownSource,
			Message:     fmt.Sprintf("source type %q is not an event stream source spec type", sourceType),
			Suggestions: suggestSources(sourceType),
			ValidTypes:  sortedSourceTypes(),
		}
	}
	return SourceRef{Type: sourceType, Token: common.SourceTypeToken(sourceType, "")}, nil
}

// suggestSources offers the source types that connect as input, so a token
// such as "web" leads to "javascript", plus near spellings.
func suggestSources(input string) []string {
	var candidates []candidate
	for _, sourceType := range sortedSourceTypes() {
		candidates = append(candidates, candidate{value: sourceType, aliases: []string{common.SourceTypeToken(sourceType, "")}})
	}
	return suggest(input, candidates)
}
