package docs

// Documented is an optional sibling interface for rules that author
// structured documentation data. The docs pipeline type-asserts each
// registered rule to Documented; rules that don't implement it produce
// no entry in the generated artifact.
//
// This is intentionally separate from the unrelated rules.Rule.Examples()
// method, which returns plain strings attached to diagnostics at runtime
// for the text renderer's end-user error output.
type Documented interface {
	DocExamples() []MatchBehaviorEntry
}
