package validate

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/spf13/cobra"
)

var (
	validateLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "validate",
	})
)

func NewCmdValidate() *cobra.Command {
	var (
		deps     app.Deps
		p        project.Project
		err      error
		location string
		genDocs  bool
		output   string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate project configuration",
		Long: heredoc.Doc(`
			Validates the project configuration files for correctness and consistency.
			This includes checking for valid syntax, required fields, and relationships
			between resources.

			Additionally, you can generate markdown documentation for all
			validation rules using the --gen-docs flag.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli validate --location </path/to/dir or file>
			$ rudder-cli validate --gen-docs
			$ rudder-cli validate --gen-docs --output ./docs/validation-rules.md
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			p = deps.NewProject()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			validateLog.Debug("validate", "location", location, "genDocs", genDocs)

			defer func() {
				telemetry.TrackCommand("validate", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "genDocs", V: genDocs},
				}...)
			}()

			// Handle documentation generation mode
			if genDocs {
				return runGenDocs(deps.CompositeProvider(), output)
			}

			// Load and validate the project
			// The Load method internally calls the provider's Validate method
			if err := p.Load(location); err != nil {
				return fmt.Errorf("validating project: %w", err)
			}

			validateLog.Info("Project configuration is valid")
			ui.PrintSuccess("Project configuration is valid")
			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&genDocs, "gen-docs", false, "Generate markdown documentation for all validation rules")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path for generated documentation (default: stdout)")

	return cmd
}

// runGenDocs generates markdown documentation for all validation rules.
func runGenDocs(prov provider.Provider, outputPath string) error {
	registry, err := buildRegistry(prov)
	if err != nil {
		return fmt.Errorf("building registry: %w", err)
	}

	generator := ui.NewRuleDocGenerator(registry)
	markdown := generator.Generate()

	if outputPath == "" {
		fmt.Println(markdown)
		return nil
	}

	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("writing documentation to %s: %w", outputPath, err)
	}

	ui.PrintSuccess(fmt.Sprintf("Documentation generated successfully at: %s", outputPath))
	return nil
}

// buildRegistry creates a registry populated with all validation rules.
func buildRegistry(prov provider.Provider) (rules.Registry, error) {
	registry := rules.NewRegistry()

	// Add project-level syntactic rules
	syntactic := []rules.Rule{
		prules.NewSpecSyntaxValidRule(),
		prules.NewMetadataSyntaxValidRule(),
		prules.NewSpecSemanticValidRule(
			prov.SupportedKinds(),
			[]string{
				specs.SpecVersionV0_1,
				specs.SpecVersionV0_1Variant,
			},
		),
	}

	// Add provider-specific syntactic rules
	syntactic = append(syntactic, prov.SyntacticRules()...)

	for _, rule := range syntactic {
		if err := registry.RegisterSyntactic(rule); err != nil {
			return nil, fmt.Errorf("registering syntactic rule %s: %w", rule.ID(), err)
		}
	}

	// Add provider-specific semantic rules
	semantic := prov.SemanticRules()
	for _, rule := range semantic {
		if err := registry.RegisterSemantic(rule); err != nil {
			return nil, fmt.Errorf("registering semantic rule %s: %w", rule.ID(), err)
		}
	}

	return registry, nil
}
