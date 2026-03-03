package rules

// Registry manages validation rules and provides lookup by (kind, version) and phase.
// Rules are stored in flat lists per phase (syntactic/semantic) and matched
// against specs using their AppliesTo() patterns at query time.
type Registry interface {
	// RegisterSyntactic adds a rule that runs before resource graph construction.
	RegisterSyntactic(rule Rule)

	// RegisterSemantic adds a rule that runs after resource graph construction.
	RegisterSemantic(rule Rule)

	// SyntacticRulesFor returns syntactic rules whose AppliesTo() patterns
	// match the given (kind, version).
	SyntacticRulesFor(kind, version string) []Rule

	// SemanticRulesFor returns semantic rules whose AppliesTo() patterns
	// match the given (kind, version).
	SemanticRulesFor(kind, version string) []Rule
}

// defaultRegistry stores rules in flat slices per phase.
// Matching is done at query time by checking each rule's AppliesTo() patterns.
type defaultRegistry struct {
	syntactic []Rule
	semantic  []Rule
}

// NewRegistry creates a new empty rule registry.
func NewRegistry() Registry {
	return &defaultRegistry{}
}

func (r *defaultRegistry) RegisterSyntactic(rule Rule) {
	r.syntactic = append(r.syntactic, rule)
}

func (r *defaultRegistry) RegisterSemantic(rule Rule) {
	r.semantic = append(r.semantic, rule)
}

func (r *defaultRegistry) SyntacticRulesFor(kind, version string) []Rule {
	return matchRules(r.syntactic, kind, version)
}

func (r *defaultRegistry) SemanticRulesFor(kind, version string) []Rule {
	return matchRules(r.semantic, kind, version)
}

// matchRules returns rules that have at least one AppliesTo() pattern
// matching the given (kind, version).
func matchRules(rules []Rule, kind, version string) []Rule {
	var matched []Rule
	for _, rule := range rules {
		for _, p := range rule.AppliesTo() {
			if p.Matches(kind, version) {
				matched = append(matched, rule)
				break
			}
		}
	}
	return matched
}
