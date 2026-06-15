package specs

// KindImportManifest is a project-level kind handled outside the resource
// provider tree, so the constant lives with the classifier rather than
// alongside resource kinds.
const KindImportManifest = "import-manifest"

// SpecLevel separates resource-level specs (sources, destinations — they flow
// through the CompositeProvider tree) from project-level specs (import
// manifests — handled by dedicated providers outside the tree).
type SpecLevel int

const (
	ResourceSpec SpecLevel = iota
	ProjectSpec
)

// Classify reports whether a spec is resource-level or project-level. The
// classification is internal — it is not visible in the YAML schema. It is a
// stateless function: classification depends only on the spec's kind.
func Classify(s *Spec) SpecLevel {
	if s.Kind == KindImportManifest {
		return ProjectSpec
	}
	return ResourceSpec
}
