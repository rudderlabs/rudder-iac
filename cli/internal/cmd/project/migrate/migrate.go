package migrate

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	projectpkg "github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/migrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewCmdMigrate() *cobra.Command {
	var (
		deps        app.Deps
		err         error
		location    string
		confirm     bool
		proj        projectpkg.Project
		postprocess migrator.WritePostprocessor
		varFiles    []string
	)

	cmd := &cobra.Command{
		Use:    "migrate",
		Short:  "Migrate project from spec rudder/0.1 to rudder/1",
		Hidden: true, // Hidden until ready for general availability
		Long: heredoc.Doc(`
			Migrates project configuration from spec rudder/0.1 to rudder/1.
			This command transforms your existing project files to the new spec version.
			
			⚠️  WARNING: This command modifies files in place. Commit or backup your
			changes before running this command.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli migrate --location </path/to/dir or file>
			$ rudder-cli migrate --location </path/to/dir or file> --confirm=false
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize dependencies
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			projectOpts, err := app.NewProjectOptions(varFiles)
			if err != nil {
				return err
			}

			postprocess, err = newPlaceholderRestorer(location, varFiles)
			if err != nil {
				return err
			}

			// Validate with substitution enabled and migrate the loaded project so
			// provider migrations can convert legacy references to v1 URNs. A write
			// postprocessor restores placeholder scalar values before writing files.
			proj = deps.NewProject(projectOpts...)
			if err := proj.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("migrate", err, migrateTelemetryExtras(location, confirm)...)
			}()

			opts := []migrator.Option{}
			if postprocess != nil {
				opts = append(opts, migrator.WithWritePostprocessor(postprocess))
			}

			m := migrator.New(proj, deps.CompositeProvider(), opts...)
			err = m.Migrate(confirm)
			return err
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm migration before proceeding")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")

	return cmd
}

type placeholderReplacement struct {
	key           string
	resolvedValue string
	rawScalar     string
}

func newPlaceholderRestorer(location string, varFiles []string) (migrator.WritePostprocessor, error) {
	rawSpecs, err := (&loader.Loader{}).Load(location)
	if err != nil {
		return nil, fmt.Errorf("loading source specs for placeholder preservation: %w", err)
	}

	sub, err := buildMigrateSubstitutor(varFiles)
	if err != nil {
		return nil, err
	}

	replacementsByPath := make(map[string][]placeholderReplacement)
	for path, rawSpec := range rawSpecs {
		replacements, err := collectPlaceholderReplacements(rawSpec.Data, sub)
		if err != nil {
			return nil, fmt.Errorf("collecting placeholder replacements for %s: %w", path, err)
		}
		if len(replacements) > 0 {
			replacementsByPath[path] = replacements
		}
	}

	if len(replacementsByPath) == 0 {
		return nil, nil
	}

	return func(path string, data []byte) ([]byte, error) {
		return restorePlaceholderScalars(data, replacementsByPath[path]), nil
	}, nil
}

func buildMigrateSubstitutor(varFiles []string) (varsubst.Substitutor, error) {
	envR, err := resolver.NewEnvResolver()
	if err != nil {
		return nil, fmt.Errorf("initialising env resolver: %w", err)
	}

	resolvers := []varsubst.Resolver{envR}
	for i := len(varFiles) - 1; i >= 0; i-- {
		r, err := resolver.NewFileResolver(varFiles[i])
		if err != nil {
			return nil, fmt.Errorf("initialising file resolver: %w", err)
		}
		resolvers = append(resolvers, r)
	}

	return varsubst.NewSubstitutor(resolvers...), nil
}

func collectPlaceholderReplacements(data []byte, sub varsubst.Substitutor) ([]placeholderReplacement, error) {
	lines := strings.Split(string(data), "\n")
	replacements := make([]placeholderReplacement, 0)
	for _, line := range lines {
		if !strings.Contains(line, "{{") {
			continue
		}

		key, _, rawScalar, ok := splitScalarMappingLine(line)
		if !ok {
			continue
		}

		resolvedLine, errs := sub.SubstituteBytes([]byte(line))
		if len(errs) > 0 {
			return nil, fmt.Errorf("%s", (&errs[0]).Error())
		}
		if string(resolvedLine) == line {
			continue
		}

		resolvedKey, _, resolvedScalar, ok := splitScalarMappingLine(string(resolvedLine))
		if !ok || resolvedKey != key {
			continue
		}

		resolvedValue, ok := parseYAMLStringScalar(resolvedScalar)
		if !ok {
			continue
		}

		replacements = append(replacements, placeholderReplacement{
			key:           key,
			resolvedValue: resolvedValue,
			rawScalar:     rawScalar,
		})
	}
	return replacements, nil
}

func restorePlaceholderScalars(data []byte, replacements []placeholderReplacement) []byte {
	if len(replacements) == 0 {
		return data
	}

	byKeyAndValue := make(map[string]string, len(replacements))
	byUniqueValue := uniquePlaceholderValues(replacements)
	for _, replacement := range replacements {
		byKeyAndValue[replacement.key+"\x00"+replacement.resolvedValue] = replacement.rawScalar
	}

	lines := strings.Split(string(data), "\n")
	valueCounts := scalarValueCounts(lines)
	for i, line := range lines {
		key, prefix, scalar, ok := splitScalarMappingLine(line)
		if !ok {
			continue
		}

		value, ok := parseYAMLStringScalar(scalar)
		if !ok {
			continue
		}

		rawScalar, ok := byKeyAndValue[key+"\x00"+value]
		if !ok {
			if valueCounts[value] != 1 {
				continue
			}
			rawScalar, ok = byUniqueValue[value]
		}
		if !ok {
			continue
		}
		lines[i] = prefix + rawScalar
	}

	return []byte(strings.Join(lines, "\n"))
}

func scalarValueCounts(lines []string) map[string]int {
	counts := make(map[string]int)
	for _, line := range lines {
		_, _, scalar, ok := splitScalarMappingLine(line)
		if !ok {
			continue
		}

		value, ok := parseYAMLStringScalar(scalar)
		if !ok {
			continue
		}
		counts[value]++
	}
	return counts
}

func uniquePlaceholderValues(replacements []placeholderReplacement) map[string]string {
	unique := make(map[string]string)
	ambiguous := make(map[string]struct{})
	for _, replacement := range replacements {
		if existing, ok := unique[replacement.resolvedValue]; ok && existing != replacement.rawScalar {
			delete(unique, replacement.resolvedValue)
			ambiguous[replacement.resolvedValue] = struct{}{}
			continue
		}
		if _, ok := ambiguous[replacement.resolvedValue]; !ok {
			unique[replacement.resolvedValue] = replacement.rawScalar
		}
	}
	return unique
}

func splitScalarMappingLine(line string) (key string, prefix string, scalar string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", "", false
	}

	colon := strings.Index(line, ":")
	if colon == -1 {
		return "", "", "", false
	}

	key = strings.TrimSpace(line[:colon])
	if key == "" {
		return "", "", "", false
	}

	valueStart := colon + 1
	for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
		valueStart++
	}
	if valueStart >= len(line) {
		return "", "", "", false
	}

	scalar = strings.TrimSpace(line[valueStart:])
	if scalar == "" || strings.HasPrefix(scalar, "#") {
		return "", "", "", false
	}

	return key, line[:valueStart], scalar, true
}

func parseYAMLStringScalar(scalar string) (string, bool) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(scalar), &node); err != nil {
		return "", false
	}
	if len(node.Content) != 1 {
		return "", false
	}
	child := node.Content[0]
	if child.Kind != yaml.ScalarNode || child.Tag != "!!str" {
		return "", false
	}
	return child.Value, true
}

// migrateTelemetryExtras returns TrackCommand key-values for migrate (fixed from/to spec versions for this path).
func migrateTelemetryExtras(location string, confirm bool) []telemetry.KV {
	return []telemetry.KV{
		{K: "location", V: location},
		{K: "confirm", V: confirm},
		{K: "from_version", V: specs.SpecVersionV0_1},
		{K: "to_version", V: specs.SpecVersionV1},
	}
}
