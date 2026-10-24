package rules

import (
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

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
//     comes from NewSourceContext.
//   - Validate() on the Rule interface should return nil (not used for these rules)
//
// Result references resolve against the source, so positions point at the
// file as written.
type SourceRule interface {
	ValidateSources(specs map[string]*ValidationContext) map[string][]ValidationResult
}

// NewSourceContext reads a spec file as written for the rules implementing
// SourceRule. It swaps each variable reference for a placeholder and parses
// that copy once, so the rules share one parse instead of each parsing the
// file again. A source with no reference is parsed through source.Parse, which
// keeps the result, so a load that leaves the source unchanged through
// substitution does not parse it again.
//
// A spec that does not parse strictly, such as one with an unknown top-level
// field, gets no Spec or Metadata, since its syntax error is reported once
// substitution has run. Its kind and version are still read, so the rules
// that apply to it still receive it.
func NewSourceContext(path string, source *specs.RawSpec) *ValidationContext {
	ctx := &ValidationContext{
		FilePath:   path,
		FileName:   filepath.Base(path),
		Source:     source.Data,
		References: varsubst.ReplaceReferences(source.Data),
	}

	parse := source.Parse
	if len(ctx.References.References) > 0 {
		parse = func() (*specs.Spec, error) { return specs.New(ctx.References.Data) }
	}
	parsed, err := parse()
	if err != nil {
		ctx.Kind, ctx.Version = kindVersion(ctx.References.Data)
		return ctx
	}
	ctx.Kind = parsed.Kind
	ctx.Version = parsed.Version
	ctx.Spec = parsed.Spec
	ctx.Metadata = parsed.Metadata
	return ctx
}

// kindVersion reads only kind and version, skipping fields that do not
// decode.
func kindVersion(data []byte) (kind, version string) {
	var header struct {
		Kind    string `yaml:"kind"`
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return "", ""
	}
	return header.Kind, header.Version
}
