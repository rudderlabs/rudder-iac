package rules

// MultiSpecRule is an optional interface for rules that need more than one
// spec at once (e.g. cross-file uniqueness). Rules implementing both Rule and
// MultiSpecRule are:
//   - Registered via RegisterSyntactic(), which returns an error if AppliesTo() fails registry validation
//   - Skipped during per-spec validation (engine detects via type assertion)
//   - Called after per-spec rules pass, with only the specs whose (kind, version)
//     match the rule's AppliesTo() patterns (the engine pre-filters)
//
// Contract:
//   - Stateless: receives the matching contexts, returns results. No side effects.
//   - Input: map of filePaths → ValidationContexts, pre-filtered by the engine to
//     the specs this rule applies to.
//   - Output: map of filePaths → ValidationResults (violations per file)
//   - Validate() on the Rule interface should return nil (not used for these rules)
type MultiSpecRule interface {
	ValidateSpecs(specs map[string]*ValidationContext) map[string][]ValidationResult
}
