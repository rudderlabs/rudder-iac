package rules

// SourceRule is an optional interface for rules that read specs as written,
// before variable substitution. Substitution replaces each {{ .VAR }}
// reference with its value, so a rule about the references themselves cannot
// read them from the parsed spec. Substitution also stops the load when any
// variable is undefined, and these rules still report then.
//
// Rules implementing both Rule and SourceRule are:
//   - Registered via RegisterSyntactic(), and documented like any other rule
//   - Called once per load, before substitution, with the specs whose
//     (kind, version) match AppliesTo(), keyed by file path. Each context
//     carries FilePath, FileName, Kind, Version and Source.
//   - Validate() on the Rule interface should return nil (not used for these rules)
//
// Result references resolve against the source, so positions point at the
// file as written.
type SourceRule interface {
	ValidateSources(specs map[string]*ValidationContext) map[string][]ValidationResult
}
