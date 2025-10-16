package utils

import "sort"

// SortableResource defines an interface for resources that can be sorted.
// will be extended with more sortable fields in the future like name, displayName, etc.
type SortableResource interface {
	GetLocalID() string
}

func SortByLocalID[T SortableResource](items []T) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetLocalID() < items[j].GetLocalID()
	})
}

// SortLexicographically sorts a slice of any type by comparing their string values.
// It expects that all elements in items are either strings or can be type casted to strings.
func SortLexicographically(items []any) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].(string) < items[j].(string)
	})
}
