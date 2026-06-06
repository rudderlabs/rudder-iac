package docs

import (
	"fmt"
	"strings"
)

// ProducedDiagnostic is a single diagnostic emitted by the validation engine
// for a given example file.
type ProducedDiagnostic struct {
	File     string
	Severity string
	Message  string
}

// VerifyMode controls how strictly MatchInvalid compares expectations to produced diagnostics.
type VerifyMode int

const (
	// ModeSubset requires every expected diagnostic to match at least one produced
	// diagnostic, but ignores unmatched produced diagnostics.
	ModeSubset VerifyMode = iota

	// ModeStrict extends ModeSubset: also flags every produced diagnostic that no
	// expectation matched, catching unexpected noise.
	ModeStrict
)

// anyMatch returns true if at least one produced diagnostic satisfies exp.
func anyMatch(exp ExpectedDiagnostic, produced []ProducedDiagnostic) bool {
	for _, p := range produced {
		if matchesExpected(exp, p) {
			return true
		}
	}
	return false
}

// matchesExpected reports whether produced satisfies the expectation.
func matchesExpected(exp ExpectedDiagnostic, p ProducedDiagnostic) bool {
	if p.File != exp.File || p.Severity != exp.Severity {
		return false
	}
	if exp.MessageContains != "" && !strings.Contains(p.Message, exp.MessageContains) {
		return false
	}
	return true
}

// MatchInvalid checks that each expected diagnostic in ex is satisfied by at least one
// entry in produced (subset semantics). In ModeStrict, any produced diagnostic that no
// expectation covers is also reported as a miss via greedy one-to-one assignment. Returns
// human-readable miss strings; an empty slice means the example verified cleanly.
func MatchInvalid(ex InvalidExample, produced []ProducedDiagnostic, mode VerifyMode) []string {
	var misses []string

	// Subset: each expected must match at least one produced (semantics unchanged).
	for _, exp := range ex.ExpectedDiagnostics {
		if !anyMatch(exp, produced) {
			misses = append(misses, fmt.Sprintf(
				"%s: expected diagnostic not found (file=%q severity=%q contains=%q)",
				ex.ExampleID, exp.File, exp.Severity, exp.MessageContains,
			))
		}
	}

	if mode == ModeStrict {
		// Greedy one-to-one assignment: each produced may only be claimed by one
		// expectation so that N identical expected entries consume N distinct produced
		// entries rather than all collapsing onto produced[0].
		matched := make([]bool, len(produced))
		for _, exp := range ex.ExpectedDiagnostics {
			for i, p := range produced {
				if !matched[i] && matchesExpected(exp, p) {
					matched[i] = true
					break
				}
			}
		}
		for i, p := range produced {
			if !matched[i] {
				misses = append(misses, fmt.Sprintf(
					"%s: unexpected diagnostic (file=%q severity=%q message=%q)",
					ex.ExampleID, p.File, p.Severity, p.Message,
				))
			}
		}
	}

	return misses
}

// MatchValid checks that no error or warning diagnostics were produced for a valid example.
// Any such diagnostic is a miss — the example should have been clean.
func MatchValid(ex ValidExample, produced []ProducedDiagnostic) []string {
	var misses []string
	for _, p := range produced {
		if p.Severity != "error" && p.Severity != "warning" {
			continue
		}
		misses = append(misses, fmt.Sprintf(
			"%s: unexpected %s diagnostic on valid example (file=%q message=%q)",
			ex.ExampleID, p.Severity, p.File, p.Message,
		))
	}
	return misses
}
