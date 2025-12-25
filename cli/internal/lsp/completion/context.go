package completion

import (
	"bytes"
	"regexp"
	"strings"
)

// CompletionType represents the type of completion being requested
type CompletionType int

const (
	// CompletionTypeNone indicates no completion is applicable
	CompletionTypeNone CompletionType = iota
	// CompletionTypeReferenceStart indicates user typed "#/" - show all kinds
	CompletionTypeReferenceStart
	// CompletionTypeReferenceKind indicates user typed "#/properties/" - show groups
	CompletionTypeReferenceKind
	// CompletionTypeReferenceResource indicates user typed "#/properties/general/" - show resources
	CompletionTypeReferenceResource
)

// CompletionContext represents the context at the cursor position
type CompletionContext struct {
	completionType CompletionType
	kind           string // "properties", "events", etc.
	group          string // "general", "api_tracking", etc.
	partial        string // what user has typed so far
}

// Regular expression to match reference patterns
// Excludes quotes to handle YAML strings like "#/properties/"
var referencePattern = regexp.MustCompile(`#/([^/"]*/?[^/"]*/?[^/"]*)$`)

// extractContext determines what the user is trying to complete based on cursor position
func extractContext(content []byte, line int, character int) *CompletionContext {
	lines := bytes.Split(content, []byte("\n"))
	if line < 0 || line >= len(lines) {
		return &CompletionContext{completionType: CompletionTypeNone}
	}

	currentLine := string(lines[line])
	if character > len(currentLine) {
		character = len(currentLine)
	}

	// Get text before cursor
	textBeforeCursor := currentLine[:character]

	// Check if we're in a reference context
	matches := referencePattern.FindStringSubmatch(textBeforeCursor)
	if len(matches) == 0 {
		return &CompletionContext{completionType: CompletionTypeNone}
	}

	refPath := matches[1]
	parts := strings.Split(refPath, "/")

	switch len(parts) {
	case 0:
		// Just "#/" typed
		return &CompletionContext{
			completionType: CompletionTypeReferenceStart,
			partial:        "",
		}
	case 1:
		// "#/prop" - completing kind
		return &CompletionContext{
			completionType: CompletionTypeReferenceStart,
			partial:        parts[0],
		}
	case 2:
		// "#/properties/" or "#/properties/gen" - completing group
		return &CompletionContext{
			completionType: CompletionTypeReferenceKind,
			kind:           parts[0],
			partial:        parts[1],
		}
	default:
		// "#/properties/general/" or more - completing resource
		return &CompletionContext{
			completionType: CompletionTypeReferenceResource,
			kind:           parts[0],
			group:          parts[1],
			partial:        strings.Join(parts[2:], "/"),
		}
	}
}
