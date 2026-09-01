package migrate

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/migrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
	"github.com/spf13/cobra"
)

var substitutionTokenRegex = regexp.MustCompile(`\{\{\s*\.[^}\s|]+(?:\s*\|\s*(?:[^}]|}[^}])*?)?\s*\}\}`)

func NewCmdMigrate() *cobra.Command {
	var (
		deps                app.Deps
		err                 error
		location            string
		confirm             bool
		proj                project.Project
		varFiles            []string
		restorePlaceholders func(path string, data []byte) ([]byte, error)
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

			// Validate project before migration.
			proj = deps.NewProject(projectOpts...)
			if err := proj.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			restorePlaceholders, err = newPlaceholderRestorer(location, varFiles)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("migrate", err, migrateTelemetryExtras(location, confirm)...)
			}()

			m := migrator.New(proj, deps.CompositeProvider(), migrator.WithWritePostprocessor(restorePlaceholders))
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
	token       string
	key         string
	scalarForms []string
}

func newPlaceholderRestorer(location string, varFiles []string) (func(path string, data []byte) ([]byte, error), error) {
	rawSpecs, err := (&loader.Loader{}).Load(location)
	if err != nil {
		return nil, fmt.Errorf("loading source specs for placeholder preservation: %w", err)
	}

	sub, err := buildMigrateSubstitutor(varFiles)
	if err != nil {
		return nil, err
	}

	replacementsByPath := make(map[string][]placeholderReplacement, len(rawSpecs))
	for path, rawSpec := range rawSpecs {
		replacements := collectPlaceholderReplacements(rawSpec.Data, sub)
		if len(replacements) > 0 {
			replacementsByPath[path] = replacements
		}
	}

	return func(path string, data []byte) ([]byte, error) {
		replacements := replacementsByPath[path]
		if len(replacements) == 0 {
			return data, nil
		}
		return restorePlaceholderScalars(data, replacements), nil
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

func collectPlaceholderReplacements(data []byte, sub varsubst.Substitutor) []placeholderReplacement {
	matches := substitutionTokenRegex.FindAllIndex(data, -1)
	replacements := make([]placeholderReplacement, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		token := string(data[match[0]:match[1]])
		resolved, errs := sub.SubstituteBytes([]byte(token))
		if len(errs) > 0 {
			continue
		}
		value := string(resolved)
		fieldKey := placeholderLineKey(data, match[0])
		seenKey := fieldKey + "\x00" + value + "\x00" + token
		if _, ok := seen[seenKey]; ok {
			continue
		}
		seen[seenKey] = struct{}{}
		replacements = append(replacements, placeholderReplacement{
			token:       token,
			key:         fieldKey,
			scalarForms: []string{value, strconv.Quote(value)},
		})
	}

	sort.SliceStable(replacements, func(i, j int) bool {
		return len(replacements[i].scalarForms[0]) > len(replacements[j].scalarForms[0])
	})

	return replacements
}

func restorePlaceholderScalars(data []byte, replacements []placeholderReplacement) []byte {
	lines := strings.SplitAfter(string(data), "\n")
	for i, line := range lines {
		lines[i] = restorePlaceholderScalarLine(line, replacements)
	}
	return []byte(strings.Join(lines, ""))
}

func restorePlaceholderScalarLine(line string, replacements []placeholderReplacement) string {
	lineEnding := ""
	content := line
	if strings.HasSuffix(content, "\n") {
		lineEnding = "\n"
		content = strings.TrimSuffix(content, "\n")
	}

	valueStart := scalarValueStart(content)
	if valueStart == -1 {
		return line
	}

	value := strings.TrimSpace(content[valueStart:])
	lineKey := scalarLineKey(content, valueStart)
	for _, replacement := range replacements {
		if replacement.key != "" && replacement.key != lineKey {
			continue
		}
		for _, scalarForm := range replacement.scalarForms {
			if value == scalarForm {
				return content[:valueStart] + replacement.token + lineEnding
			}
		}
	}

	return line
}

func placeholderLineKey(data []byte, idx int) string {
	lineStart := idx
	for lineStart > 0 && data[lineStart-1] != '\n' {
		lineStart--
	}
	return rawLineKey(string(data[lineStart:idx]))
}

func scalarValueStart(line string) int {
	if idx := strings.Index(line, ": "); idx >= 0 {
		return idx + len(": ")
	}

	trimmed := strings.TrimLeft(line, " ")
	indent := len(line) - len(trimmed)
	if strings.HasPrefix(trimmed, "- ") {
		return indent + len("- ")
	}

	return -1
}

func scalarLineKey(line string, valueStart int) string {
	return rawLineKey(line[:valueStart])
}

func rawLineKey(prefix string) string {
	idx := strings.LastIndex(prefix, ":")
	if idx == -1 {
		return ""
	}
	key := strings.TrimSpace(prefix[:idx])
	key = strings.TrimPrefix(key, "- ")
	return key
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
