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
