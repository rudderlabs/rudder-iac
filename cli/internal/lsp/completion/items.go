package completion

import (
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// getKindCompletions returns completions for reference kinds (properties, events, etc.)
func (p *CompletionProvider) getKindCompletions() []protocol.CompletionItem {
	kinds := []struct {
		label       string
		description string
		insertText  string
	}{
		{"properties", "Property definitions", "properties/"},
		{"events", "Event definitions", "events/"},
		{"custom-types", "Custom type definitions", "custom-types/"},
		{"categories", "Category definitions", "categories/"},
	}

	items := make([]protocol.CompletionItem, len(kinds))
	for i, k := range kinds {
		detail := k.description
		kindType := protocol.CompletionItemKindModule
		items[i] = protocol.CompletionItem{
			Label:         k.label,
			Kind:          &kindType,
			Detail:        &detail,
			InsertText:    &k.insertText,
			Documentation: k.description,
		}
	}
	return items
}

// getGroupCompletions returns completions for groups within a kind
func (p *CompletionProvider) getGroupCompletions(
	kind string,
	graph *resources.Graph,
) []protocol.CompletionItem {

	// Extract unique groups from resource graph
	groups := p.extractGroupsForKind(kind, graph)

	items := make([]protocol.CompletionItem, 0, len(groups))
	kindType := protocol.CompletionItemKindFolder
	for groupName, count := range groups {
		detail := fmt.Sprintf("%d items in %s", count, groupName)
		insertText := groupName + "/"
		items = append(items, protocol.CompletionItem{
			Label:      groupName,
			Kind:       &kindType,
			Detail:     &detail,
			InsertText: &insertText,
		})
	}

	return items
}

// getResourceCompletions returns completions for resources within a group
func (p *CompletionProvider) getResourceCompletions(
	kind string,
	group string,
	graph *resources.Graph,
) []protocol.CompletionItem {

	resourceList := p.extractResourcesForGroup(kind, group, graph)

	items := make([]protocol.CompletionItem, 0, len(resourceList))
	kindType := protocol.CompletionItemKindField
	for _, res := range resourceList {
		metadata := res.FileMetadata()
		if metadata == nil || metadata.MetadataRef == "" {
			continue
		}

		// Parse the reference: "#/properties/general/globalId"
		ref := metadata.MetadataRef
		parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
		if len(parts) < 3 {
			continue
		}

		resourceID := parts[2]

		// Extract description from resource data
		description := p.extractDescription(res)

		// Build completion item
		detail := fmt.Sprintf("%s: %s", kind, resourceID)
		items = append(items, protocol.CompletionItem{
			Label:         resourceID,
			Kind:          &kindType,
			Detail:        &detail,
			Documentation: description,
			InsertText:    &resourceID,
		})
	}

	return items
}

// extractGroupsForKind extracts unique groups for a specific kind from the resource graph
func (p *CompletionProvider) extractGroupsForKind(kind string, graph *resources.Graph) map[string]int {
	groups := make(map[string]int)

	for _, resource := range graph.Resources() {
		metadata := resource.FileMetadata()
		if metadata == nil || metadata.MetadataRef == "" {
			continue
		}

		// Parse "#/properties/general/globalId"
		ref := metadata.MetadataRef
		parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
		if len(parts) < 2 {
			continue
		}

		refKind := parts[0]
		refGroup := parts[1]

		if refKind == kind {
			groups[refGroup]++
		}
	}

	return groups
}

// extractResourcesForGroup extracts resources for a specific kind and group
func (p *CompletionProvider) extractResourcesForGroup(
	kind string,
	group string,
	graph *resources.Graph,
) []*resources.Resource {
	var resourceList []*resources.Resource

	for _, resource := range graph.Resources() {
		metadata := resource.FileMetadata()
		if metadata == nil || metadata.MetadataRef == "" {
			continue
		}

		// Parse "#/properties/general/globalId"
		ref := metadata.MetadataRef
		parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
		if len(parts) < 3 {
			continue
		}

		refKind := parts[0]
		refGroup := parts[1]

		if refKind == kind && refGroup == group {
			resourceList = append(resourceList, resource)
		}
	}

	return resourceList
}

// extractDescription extracts a description from resource data
func (p *CompletionProvider) extractDescription(res *resources.Resource) string {
	data := res.Data()
	if desc, ok := data["description"].(string); ok && desc != "" {
		return desc
	}
	if name, ok := data["name"].(string); ok && name != "" {
		return name
	}
	return ""
}
