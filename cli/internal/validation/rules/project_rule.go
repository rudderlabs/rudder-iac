package rules

// ProjectRule is an optional interface for project-wide validation.
// Rules implementing both Rule and ProjectRule are:
//   - Registered via RegisterSyntactic() like any rule
//   - Skipped during per-spec validation (engine detects via type assertion)
//   - Called with ALL parsed specs after per-spec rules pass
//
// Contract:
//   - Stateless: receives complete context, returns results. No side effects.
//   - Input: map of ALL filePaths → ValidationContexts (entire project)
//   - Output: map of filePaths → ValidationResults (violations per file)
//   - Validate() on the Rule interface should return nil (not used for project rules)
type ProjectRule interface {
	ValidateProject(specs map[string]*ValidationContext) map[string][]ValidationResult
}
