// Package ruledoc assembles the canonical validation rule documentation
// catalog from an already-composed provider.
//
// It is the glue layer above the provider set, the rule registry, and the docs
// generator: given a provider it builds the registry, joins the live rules
// with every authored *.docs.yaml fragment — provider-contributed plus the
// project-level gatekeeper fragments — and runs the generator.
//
// Keeping this provider-in, catalog-out step in its own package lets it be
// unit-tested with a stub provider, free of any client, config, or auth, while
// the composition root (package app) owns only the credentialled provider
// wiring. Sharing project.BuildRegistry with project validation guarantees the
// documented rule set cannot drift from what validation observes.
package ruledoc

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	projectdocs "github.com/rudderlabs/rudder-iac/cli/internal/project/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
)

// Build joins the rules registered for cp with the authored doc fragments and
// returns the validated catalog.
//
// verrs carries catalog validation failures (e.g. a registered rule with no
// authored fragment) and is returned rather than raised so the caller can
// surface every problem at once; a non-nil error means the catalog could not
// be assembled at all. generatedAt is injected so callers own the timestamp —
// the real clock in the command, a fixed value in tests.
func Build(cp provider.Provider, cliVersion, generatedAt string) (docs.DocumentedRules, []error, error) {
	reg, err := project.BuildRegistry(cp)
	if err != nil {
		return docs.DocumentedRules{}, nil, fmt.Errorf("building rule registry: %w", err)
	}

	// Project-level (gatekeeper) rules are registered outside any provider, so
	// their authored fragments are embedded here and appended to the
	// provider-contributed entries.
	entries := cp.RuleDocEntries()
	projectEntries, err := docs.LoadRuleDocEntries(projectdocs.FragmentsFS, ".")
	if err != nil {
		return docs.DocumentedRules{}, nil, fmt.Errorf("loading project rule docs: %w", err)
	}
	entries = append(entries, projectEntries...)

	doc, verrs := docs.Generate(
		reg.AllSyntacticRules(),
		reg.AllSemanticRules(),
		entries,
		cliVersion,
		generatedAt,
	)
	return doc, verrs, nil
}
