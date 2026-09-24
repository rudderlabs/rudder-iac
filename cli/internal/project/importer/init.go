package importer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// specExtensions are the files that make a directory a project rather than an
// empty scratch space. Anything else — a README, a .gitignore, a var file — is
// no reason to refuse a clone.
var specExtensions = []string{".yaml", ".yml"}

// maxReportedSpecs caps how many existing spec files the refusal names — the
// walk stops there, so a large project does not get read end to end just to
// produce an error message.
const maxReportedSpecs = 3

// InitProvider is the slice of the provider surface a clone needs: list
// everything upstream, then render it. No state mapping and no matchers — with
// the whole workspace in one collection there is no local project to diff
// against or link to.
type InitProvider interface {
	provider.UnmanagedRemoteResourceLoader
	provider.Exporter
}

// WorkspaceInit writes every resource in the workspace — managed and unmanaged
// alike — into location as spec files.
//
// Unlike [WorkspaceImport] it does not load the local project, diff it against
// remote, or emit an import manifest. It does not need to: the target directory
// is required to hold no specs, so there is nothing to diverge from and nothing
// a written spec could collide with. Managed resources keep the ExternalID they
// carry upstream, which is what makes the first apply on a clone a no-op.
func WorkspaceInit(ctx context.Context, location string, p InitProvider) error {
	if err := checkNoSpecs(location); err != nil {
		return err
	}

	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())

	importable, err := p.LoadImportable(ctx, idNamer, resources.ImportableFilter{IncludeManaged: true})
	if err != nil {
		return fmt.Errorf("loading workspace resources: %w", err)
	}

	if importable.Len() == 0 {
		fmt.Println("No resources found in the workspace")
		return nil
	}

	// Every resource is in the importable collection, so every cross-spec
	// reference resolves off it — the Remote/Graph arms of the resolver, which
	// exist to point at specs a local project already has, are never reached.
	refResolver := &resolver.ImportRefResolver{
		Remote:     resources.NewRemoteResources(),
		Graph:      resources.NewGraph(),
		Importable: importable,
	}

	entities, _, err := p.FormatForExport(importable, idNamer, refResolver)
	if err != nil {
		return fmt.Errorf("normalizing workspace resources for export: %w", err)
	}

	formatters := formatter.Setup(formatter.DefaultYAML, formatter.DefaultText)
	if err := writer.Write(ctx, location, formatters, entities); err != nil {
		return fmt.Errorf("writing files for formattable entities: %w", err)
	}

	varFile, err := scaffoldSecretsVarFile(ctx, location, entities)
	if err != nil {
		return fmt.Errorf("scaffolding secrets var file: %w", err)
	}
	if varFile != "" {
		ui.PrintInfo(fmt.Sprintf("The specs reference variables for secret values.\n"+
			"Fill in the placeholders in %s (keep it out of version control) and pass it to apply via --var-file.", varFile))
	}

	return nil
}

// checkNoSpecs refuses a location that already holds spec files, anywhere under
// it — a project keeps its specs in per-provider subdirectories, which is the
// layout init itself writes, so a top-level-only check would miss the case it
// exists to catch.
//
// A clone is the whole project: writing it over an existing one would duplicate
// every resource the workspace and the project have in common, and the next
// apply would fail on the unique-name collisions. `import workspace` is the
// command for adding to a project that already exists.
func checkNoSpecs(location string) error {
	var found []string
	err := filepath.WalkDir(location, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if slices.Contains(specExtensions, strings.ToLower(filepath.Ext(d.Name()))) {
			rel, relErr := filepath.Rel(location, path)
			if relErr != nil {
				rel = path
			}
			found = append(found, rel)
			// One is enough to refuse; naming a couple is enough to explain why.
			if len(found) == maxReportedSpecs {
				return fs.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading directory %s: %w", location, err)
	}

	if len(found) > 0 {
		return fmt.Errorf("%w: %s already contains spec files (%s) — use 'rudder-cli import workspace' to add remote resources to an existing project",
			ErrProjectNotEmpty, location, strings.Join(found, ", "))
	}
	return nil
}
