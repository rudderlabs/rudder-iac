package resources

// ImportableFilter selects which remote resources a provider offers up for
// import. It is threaded variadically through LoadImportable so the common
// case — unmanaged resources only, which is what `import workspace` wants —
// stays a two-argument call at every existing call site.
type ImportableFilter struct {
	// IncludeManaged also offers resources already under CLI management (those
	// carrying an ExternalID upstream), keeping that ExternalID as the local
	// identifier instead of generating one from the resource's name. Renaming
	// them would make a cloned project read as a rename of everything on the
	// next apply. `init` clones a whole workspace with this set.
	IncludeManaged bool
}

// ImportableFilterOf collapses the variadic filter argument to a single value.
// The zero filter — unmanaged only — is the default every implementation of
// LoadImportable falls back to.
func ImportableFilterOf(filters []ImportableFilter) ImportableFilter {
	if len(filters) == 0 {
		return ImportableFilter{}
	}
	return filters[0]
}

// KeepID returns the identifier a resource should keep: its upstream
// ExternalID while managed resources are in scope, and "" — generate one —
// otherwise. Gating it here preserves `import workspace`'s behaviour of
// treating everything it writes as new, even for a handler that hands back
// managed resources regardless of the filter.
func (f ImportableFilter) KeepID(externalID string) string {
	if !f.IncludeManaged {
		return ""
	}
	return externalID
}

// UnmanagedOnly is the value for an API-level "has external ID" filter: a
// pointer to false while only unmanaged resources are wanted, and nil — no
// filter at all — once managed resources are in scope too.
func (f ImportableFilter) UnmanagedOnly() *bool {
	if f.IncludeManaged {
		return nil
	}
	unmanaged := false
	return &unmanaged
}
