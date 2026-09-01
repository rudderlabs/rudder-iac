package migrate

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/migrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/spf13/cobra"
)

func NewCmdMigrate() *cobra.Command {
	var (
		deps     app.Deps
		err      error
		location string
		confirm  bool
		proj     migrator.Project
		varFiles []string
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

			// Validate with substitution enabled, then migrate the original parsed
			// specs so in-place writes keep placeholders instead of resolved values.
			validationProject := deps.NewProject(projectOpts...)
			if err := validationProject.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			proj, err = loadRawMigrationProject(location)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("migrate", err, migrateTelemetryExtras(location, confirm)...)
			}()

			m := migrator.New(proj, deps.CompositeProvider())
			err = m.Migrate(confirm)
			return err
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm migration before proceeding")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")

	return cmd
}

type rawMigrationProject struct {
	location string
	specs    map[string]*specs.Spec
}

func loadRawMigrationProject(location string) (migrator.Project, error) {
	rawSpecs, err := (&loader.Loader{}).Load(location)
	if err != nil {
		return nil, fmt.Errorf("loading source specs for migration: %w", err)
	}

	parsedSpecs := make(map[string]*specs.Spec, len(rawSpecs))
	for path, rawSpec := range rawSpecs {
		parseableData, mask := varsubst.MaskTokensForYAMLParse(rawSpec.Data)
		parseableRawSpec := &specs.RawSpec{Data: parseableData}
		parsed, err := parseableRawSpec.Parse()
		if err != nil {
			return nil, fmt.Errorf("parsing source spec %s: %w", path, err)
		}
		if err := restoreMaskedTokens(parsed, mask); err != nil {
			return nil, fmt.Errorf("restoring variable placeholders in source spec %s: %w", path, err)
		}
		parsedSpecs[path] = parsed
	}

	return &rawMigrationProject{location: location, specs: parsedSpecs}, nil
}

func restoreMaskedTokens(spec *specs.Spec, mask varsubst.TokenMask) error {
	var err error
	spec.Version, err = restoreMaskedString(spec.Version, mask)
	if err != nil {
		return fmt.Errorf("restoring version: %w", err)
	}
	spec.Kind, err = restoreMaskedString(spec.Kind, mask)
	if err != nil {
		return fmt.Errorf("restoring kind: %w", err)
	}

	metadata, err := restoreMaskedValue(spec.Metadata, mask)
	if err != nil {
		return err
	}
	if restored, ok := metadata.(map[string]any); ok {
		spec.Metadata = restored
	}

	specData, err := restoreMaskedValue(spec.Spec, mask)
	if err != nil {
		return err
	}
	if restored, ok := specData.(map[string]any); ok {
		spec.Spec = restored
	}

	return nil
}

func restoreMaskedValue(value any, mask varsubst.TokenMask) (any, error) {
	switch v := value.(type) {
	case string:
		restored, err := restoreMaskedString(v, mask)
		if err != nil {
			return nil, err
		}
		return restored, nil
	case []any:
		restored := make([]any, len(v))
		for i, item := range v {
			restoredItem, err := restoreMaskedValue(item, mask)
			if err != nil {
				return nil, fmt.Errorf("restoring list item %d: %w", i, err)
			}
			restored[i] = restoredItem
		}
		return restored, nil
	case map[string]any:
		restored := make(map[string]any, len(v))
		for key, item := range v {
			restoredKey := mask.RestoreString(key)
			if mask.ContainsSentinel(restoredKey) {
				return nil, fmt.Errorf("unrestored variable placeholder sentinel in key %q", restoredKey)
			}
			if _, exists := restored[restoredKey]; exists {
				return nil, fmt.Errorf("restoring variable placeholder key %q creates a duplicate key", restoredKey)
			}

			restoredItem, err := restoreMaskedValue(item, mask)
			if err != nil {
				return nil, fmt.Errorf("restoring key %q: %w", restoredKey, err)
			}
			restored[restoredKey] = restoredItem
		}
		return restored, nil
	default:
		return value, nil
	}
}

func restoreMaskedString(value string, mask varsubst.TokenMask) (string, error) {
	restored := mask.RestoreString(value)
	if mask.ContainsSentinel(restored) {
		return "", fmt.Errorf("unrestored variable placeholder sentinel in value %q", restored)
	}
	return restored, nil
}

func (p *rawMigrationProject) Location() string {
	return p.location
}

func (p *rawMigrationProject) Specs() map[string]*specs.Spec {
	return p.specs
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
