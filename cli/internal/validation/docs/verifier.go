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

// MatchInvalid checks that each expected diagnostic in ex is satisfied by at least one
// entry in produced (subset semantics). In ModeStrict, any produced diagnostic that no
// expectation covers is also reported as a miss. Returns human-readable miss strings; an
// empty slice means the example verified cleanly.
func MatchInvalid(ex InvalidExample, produced []ProducedDiagnostic, mode VerifyMode) []string {
	// matchedProduced tracks which produced indices were claimed by an expectation;
	// only relevant in ModeStrict.
	matchedProduced := make([]bool, len(produced))

	var misses []string

	for _, exp := range ex.ExpectedDiagnostics {
		matched := false
		for i, p := range produced {
			if p.File != exp.File || p.Severity != exp.Severity {
				continue
			}
			if exp.MessageContains != "" && !strings.Contains(p.Message, exp.MessageContains) {
				continue
			}
			matchedProduced[i] = true
			matched = true
			break
		}
		if !matched {
			misses = append(misses, fmt.Sprintf(
				"%s: expected diagnostic not found (file=%q severity=%q contains=%q)",
				ex.ExampleID, exp.File, exp.Severity, exp.MessageContains,
			))
		}
	}

	if mode == ModeStrict {
		for i, p := range produced {
			if matchedProduced[i] {
				continue
			}
			misses = append(misses, fmt.Sprintf(
				"%s: unexpected diagnostic (file=%q severity=%q message=%q)",
				ex.ExampleID, p.File, p.Severity, p.Message,
			))
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
