package rules

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	datacatalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	esProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	retl "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	rrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/spf13/cobra"
)

var log = logger.New("docs-rules")

func NewCmdRules() *cobra.Command {
	var (
		outputDir    string
		strictVerify bool
	)

	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Generate validation rule documentation (JSON + YAML)",
		Long: heredoc.Doc(`
			Walk the registered validation rules, resolve documentation
			data for rules that implement the Documented interface,
			verify that each authored invalid example actually produces
			the expected diagnostics, and emit the consolidated artifact
			to the output directory.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli docs rules
			$ rudder-cli docs rules --output-dir ./docs/generated/
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strictVerify {
				return fmt.Errorf(
					"strict-verify mode is not implemented in the spike (tracked in DEX-216 follow-up); rerun without --strict-verify",
				)
			}

			registry, err := buildRegistry()
			if err != nil {
				return fmt.Errorf("building registry: %w", err)
			}

			engine, err := validation.NewValidationEngine(registry, log)
			if err != nil {
				return fmt.Errorf("constructing validation engine: %w", err)
			}

			gen := docs.NewGenerator(docs.ExamplesResolver{}, cliVersionString(cmd))
			doc, err := gen.Generate(registry)
			if err != nil {
				return fmt.Errorf("generating rules doc: %w", err)
			}

			verifier := docs.NewVerifier(func() validation.ValidationEngine { return engine })
			if err := verifier.Verify(doc); err != nil {
				return fmt.Errorf("verifying examples: %w", err)
			}

			if err := docs.Serialize(doc, outputDir); err != nil {
				return fmt.Errorf("serializing: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Wrote rules.yaml and rules.json to %s\n", outputDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&outputDir, "output-dir", "./docs/generated/", "Directory to write rules.yaml and rules.json")
	cmd.Flags().BoolVar(&strictVerify, "strict-verify", false,
		"Switch verifier to exact-match mode (not implemented in the spike; tracked in DEX-216 follow-up)")
	return cmd
}

// buildRegistry constructs a rules.Registry directly from each provider
// without needing an authenticated API client. Provider rule constructors
// are pure (they don't reach into the client at registration time), so a
// nil client is safe for docs generation.
func buildRegistry() (rrules.Registry, error) {
	reg := rrules.NewRegistry()

	providerMap := map[string]provider.Provider{
		"datacatalog":     datacatalog.New(nil),
		"retl":            retl.New(nil),
		"eventstream":     esProvider.New(nil),
		"transformations": transformations.NewProvider(nil),
	}

	cp, err := provider.NewCompositeProvider(providerMap)
	if err != nil {
		return nil, fmt.Errorf("composite provider: %w", err)
	}

	for _, p := range providerMap {
		for _, r := range p.SyntacticRules() {
			reg.RegisterSyntactic(r)
		}
		for _, r := range p.SemanticRules() {
			reg.RegisterSemantic(r)
		}
	}

	validVersions := []string{
		specs.SpecVersionV0_1,
		specs.SpecVersionV0_1Variant,
		specs.SpecVersionV1,
	}
	reg.RegisterSyntactic(prules.NewSpecSyntaxValidRule(cp.SupportedKinds(), validVersions))
	reg.RegisterSyntactic(prules.NewMetadataSyntaxValidRule(cp.ParseSpec, validVersions))
	reg.RegisterSyntactic(prules.NewDuplicateURNRule(cp.ParseSpec))

	return reg, nil
}

func cliVersionString(cmd *cobra.Command) string {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Version != "" {
			return c.Version
		}
	}
	return "dev"
}
