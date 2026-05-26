package rules

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/spf13/cobra"
)

const (
	defaultFragmentsDir = "cli/internal/validation/docs/fragments"
	defaultOutputDir    = "docs/generated"
)

var log = logger.New("docs-rules")

// NewCmdRules wires the `rudder-cli docs rules` subcommand.
func NewCmdRules() *cobra.Command {
	var (
		fragmentsDir string
		outputDir    string
		strictVerify bool
	)

	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Generate the validation rules documentation artifact",
		Long: heredoc.Doc(`
			Generates rules.yaml and rules.json describing every registered
			validation rule with authored docs. Hugo consumes the YAML to render
			markdown for the public docs site; the JSON is intended for LLMs.

			Every authored invalid example is verified at generation time by
			running it through the validation engine and asserting that the
			authored expected diagnostics are produced (subset semantics).
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strictVerify {
				return fmt.Errorf("--strict-verify is not implemented in the spike (DEX-371); only subset mode is supported")
			}
			return run(cmd.Context(), fragmentsDir, outputDir)
		},
	}

	cmd.Flags().StringVar(&fragmentsDir, "fragments-dir", defaultFragmentsDir, "Directory containing rule doc YAML fragments")
	cmd.Flags().StringVar(&outputDir, "output-dir", defaultOutputDir, "Directory to write rules.yaml and rules.json into")
	cmd.Flags().BoolVar(&strictVerify, "strict-verify", false, "Use exact-match verification (not implemented in the spike)")

	return cmd
}

func run(ctx context.Context, fragmentsDir, outputDir string) error {
	deps, err := app.NewDeps()
	if err != nil {
		return fmt.Errorf("initialising dependencies: %w", err)
	}

	reg, err := buildRegistry(deps)
	if err != nil {
		return fmt.Errorf("building registry: %w", err)
	}

	resolver, err := docs.NewYAMLResolver(os.DirFS("."), fragmentsDir)
	if err != nil {
		return fmt.Errorf("creating YAML resolver: %w", err)
	}

	gen := docs.NewGenerator(reg, resolver, docs.GeneratorOptions{
		CLIVersion:    app.GetVersion(),
		SchemaVersion: 1,
	})
	doc, err := gen.Generate()
	if err != nil {
		return fmt.Errorf("generating rules doc: %w", err)
	}

	if errs := doc.Validate(nil); len(errs) > 0 {
		return fmt.Errorf("structural validation failed: %v", errs)
	}

	verifier, err := docs.NewVerifier(reg, log)
	if err != nil {
		return fmt.Errorf("creating verifier: %w", err)
	}
	if err := verifier.Verify(ctx, doc); err != nil {
		return fmt.Errorf("executable verification failed: %w", err)
	}

	yamlPath, err := docs.EmitYAML(doc, outputDir)
	if err != nil {
		return fmt.Errorf("emitting YAML: %w", err)
	}
	jsonPath, err := docs.EmitJSON(doc, outputDir)
	if err != nil {
		return fmt.Errorf("emitting JSON: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Wrote %s and %s", yamlPath, jsonPath))
	return nil
}

// buildRegistry mirrors project.registry() inline so the docs command can run
// without depending on a full project load. Duplication is intentional for the
// spike — see plan §6 commit-decision notes.
func buildRegistry(deps app.Deps) (vrules.Registry, error) {
	provider := deps.CompositeProvider()
	reg := vrules.NewRegistry()

	validVersions := []string{specs.SpecVersionV0_1, specs.SpecVersionV0_1Variant, specs.SpecVersionV1}

	syntactic := []vrules.Rule{
		prules.NewSpecSyntaxValidRule(provider.SupportedKinds(), validVersions),
		prules.NewMetadataSyntaxValidRule(provider.ParseSpec, validVersions),
		prules.NewDuplicateURNRule(provider.ParseSpec),
	}
	syntactic = append(syntactic, provider.SyntacticRules()...)
	for _, r := range syntactic {
		reg.RegisterSyntactic(r)
	}

	for _, r := range provider.SemanticRules() {
		reg.RegisterSemantic(r)
	}

	return reg, nil
}
