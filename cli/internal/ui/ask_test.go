package ui

import "testing"

// Select must refuse an empty option set rather than returning "" with no error,
// because callers filter options (by entitlement, by detected stack) and an empty
// result means the filter removed everything.
func TestSelectRejectsEmptyOptions(t *testing.T) {
	got, err := Select("Where should your data go?", nil)
	if err == nil {
		t.Fatalf("expected an error for an empty option set, got value %q", got)
	}
	if got != "" {
		t.Fatalf("expected no value alongside the error, got %q", got)
	}
}
