package completion

import (
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// CompletionCache caches pre-built completion lists for performance
type CompletionCache struct {
	mu sync.RWMutex

	// Pre-built completions by kind and group
	kindCompletions  []protocol.CompletionItem
	groupsByKind     map[string][]protocol.CompletionItem
	resourcesByGroup map[string]map[string][]protocol.CompletionItem

	// Version tracking to know when to invalidate
	graphVersion int64
}

// NewCache creates a new completion cache
func NewCache() *CompletionCache {
	return &CompletionCache{
		groupsByKind:     make(map[string][]protocol.CompletionItem),
		resourcesByGroup: make(map[string]map[string][]protocol.CompletionItem),
	}
}

// Rebuild rebuilds the cache from the resource graph
func (c *CompletionCache) Rebuild(graph *resources.Graph, provider *CompletionProvider) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear existing cache
	c.groupsByKind = make(map[string][]protocol.CompletionItem)
	c.resourcesByGroup = make(map[string]map[string][]protocol.CompletionItem)

	// Build kind completions (these are static)
	c.kindCompletions = provider.getKindCompletions()

	// Build group and resource completions from graph
	kinds := []string{"properties", "events", "custom-types", "categories"}
	for _, kind := range kinds {
		// Build group completions for this kind
		c.groupsByKind[kind] = provider.getGroupCompletions(kind, graph)

		// Build resource completions for each group
		if _, exists := c.resourcesByGroup[kind]; !exists {
			c.resourcesByGroup[kind] = make(map[string][]protocol.CompletionItem)
		}

		// Extract groups for this kind
		groups := provider.extractGroupsForKind(kind, graph)
		for groupName := range groups {
			c.resourcesByGroup[kind][groupName] = provider.getResourceCompletions(kind, groupName, graph)
		}
	}

	c.graphVersion++
}

// GetKindCompletions returns cached kind completions
func (c *CompletionCache) GetKindCompletions() []protocol.CompletionItem {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.kindCompletions
}

// GetGroupsForKind returns cached group completions for a kind
func (c *CompletionCache) GetGroupsForKind(kind string) []protocol.CompletionItem {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if items, ok := c.groupsByKind[kind]; ok {
		return items
	}
	return nil
}

// GetResourcesForGroup returns cached resource completions for a group
func (c *CompletionCache) GetResourcesForGroup(kind, group string) []protocol.CompletionItem {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if groups, ok := c.resourcesByGroup[kind]; ok {
		if items, ok := groups[group]; ok {
			return items
		}
	}
	return nil
}
