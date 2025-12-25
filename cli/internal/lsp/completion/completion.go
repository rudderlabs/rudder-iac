package completion

import (
	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// CompletionProvider handles completion requests
type CompletionProvider struct {
	cache *CompletionCache
}

// NewProvider creates a new completion provider
func NewProvider() *CompletionProvider {
	return &CompletionProvider{
		cache: NewCache(),
	}
}

// GetCompletions returns completion items for the given context
func (p *CompletionProvider) GetCompletions(
	content []byte,
	line int,
	character int,
	resourceGraph *resources.Graph,
) ([]protocol.CompletionItem, error) {

	// Validate inputs
	if resourceGraph == nil {
		return []protocol.CompletionItem{}, nil
	}

	// Extract context (what's being typed)
	ctx := extractContext(content, line, character)

	// Determine completion type based on context
	switch ctx.completionType {
	case CompletionTypeReferenceStart:
		// User typed "#/" - show all kinds
		return p.cache.GetKindCompletions(), nil

	case CompletionTypeReferenceKind:
		// User typed "#/properties/" - show all groups for this kind
		// Always generate from live graph to ensure fresh data
		return p.getGroupCompletions(ctx.kind, resourceGraph), nil

	case CompletionTypeReferenceResource:
		// User typed "#/properties/general/" - show all resources in group
		// Always generate from live graph to ensure fresh data
		return p.getResourceCompletions(ctx.kind, ctx.group, resourceGraph), nil

	default:
		return []protocol.CompletionItem{}, nil
	}
}

// RebuildCache rebuilds the completion cache from the resource graph
func (p *CompletionProvider) RebuildCache(graph *resources.Graph) {
	if graph == nil {
		return
	}
	p.cache.Rebuild(graph, p)
}
