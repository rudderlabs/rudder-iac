package project

import (
	"fmt"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
)

// manifestRegistry builds the validation registry exclusively for import-manifest specs.
// Manifest validation is separate from resource validation to avoid coupling
// manifest rules with provider-specific concerns like ParseSpec dispatch.
func manifestRegistry() (rules.Registry, error) {
	manifestPatterns := []rules.MatchPattern{
		rules.MatchKindVersion(specs.KindImportManifest, specs.SpecVersionV1),
	}
	registry := rules.NewRegistry(manifestPatterns)

	syntactic := []rules.Rule{
		prules.NewSpecSyntaxValidRule(manifestPatterns),
		prules.NewResourceKindVersionValidRule(manifestPatterns),
		prules.NewImportManifestSyntaxRule(),
		prules.NewImportManifestProjectRule(),
	}
	for _, rule := range syntactic {
		if err := registry.RegisterSyntactic(rule); err != nil {
			return nil, fmt.Errorf("registering manifest syntactic rule %s: %w", rule.ID(), err)
		}
	}

	semantic := []rules.Rule{
		prules.NewImportManifestSemanticRule(),
	}
	for _, rule := range semantic {
		if err := registry.RegisterSemantic(rule); err != nil {
			return nil, fmt.Errorf("registering manifest semantic rule %s: %w", rule.ID(), err)
		}
	}

	return registry, nil
}

// checkInlineManifestConflicts detects URNs that appear in both manifest files
// and inline metadata.import blocks in resource specs. This is a temporary
// migration concern — once inline metadata is removed, this check becomes a no-op.
func checkInlineManifestConflicts(
	resourceSpecs map[string]*specs.RawSpec,
	manifestSpecs map[string]*specs.RawSpec,
) validation.Diagnostics {
	if len(manifestSpecs) == 0 {
		return nil
	}

	// Collect all URNs from manifests with their source file
	type urnSource struct {
		filePath  string
		reference string
	}
	manifestURNs := make(map[string]urnSource)

	for path, rawSpec := range manifestSpecs {
		for _, entry := range prules.ExtractManifestURNs(rawSpec.Parsed().Spec) {
			manifestURNs[entry.URN] = urnSource{filePath: path, reference: entry.Reference}
		}
	}

	if len(manifestURNs) == 0 {
		return nil
	}

	// Check resource specs' inline metadata.import for conflicts
	var diags validation.Diagnostics
	for resourcePath, rawSpec := range resourceSpecs {
		inlineURNs := prules.ExtractInlineImportURNs(rawSpec.Parsed().Metadata)
		for _, urn := range inlineURNs {
			src, exists := manifestURNs[urn]
			if !exists {
				continue
			}

			pos := pathindex.StartingPosition
			if pi, err := rawSpec.PathIndexer(); err == nil {
				pos = *pi.NearestPosition("/metadata/import")
			}

			diags = append(diags, validation.Diagnostic{
				RuleID:   "project/import-manifest-inline-conflict",
				Severity: rules.Error,
				Message: fmt.Sprintf(
					"URN '%s' found in both manifest %s and inline metadata in %s",
					urn, filepath.Base(src.filePath), filepath.Base(resourcePath),
				),
				File:     resourcePath,
				Position: pos,
			})
		}
	}

	return diags
}

func separateManifests(
	rawSpecs map[string]*specs.RawSpec,
) (resourceSpecs, manifestSpecs map[string]*specs.RawSpec) {
	resourceSpecs = make(map[string]*specs.RawSpec, len(rawSpecs))
	manifestSpecs = make(map[string]*specs.RawSpec)
	for path, rawSpec := range rawSpecs {
		if rawSpec.Parsed().IsImportManifest() {
			manifestSpecs[path] = rawSpec
			continue
		}
		resourceSpecs[path] = rawSpec
	}
	return resourceSpecs, manifestSpecs
}

func parseManifests(
	manifestSpecs map[string]*specs.RawSpec,
) (*specs.WorkspacesImportMetadata, error) {
	if len(manifestSpecs) == 0 {
		return nil, nil
	}

	var allWorkspaces []specs.WorkspaceImportMetadata
	for path, rawSpec := range manifestSpecs {
		ws, err := decodeManifestWorkspaces(rawSpec.Parsed().Spec)
		if err != nil {
			return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
		}
		allWorkspaces = append(allWorkspaces, ws...)
	}

	return &specs.WorkspacesImportMetadata{Workspaces: allWorkspaces}, nil
}

func decodeManifestWorkspaces(specMap map[string]any) ([]specs.WorkspaceImportMetadata, error) {
	var payload struct {
		Workspaces []specs.WorkspaceImportMetadata `yaml:"workspaces"`
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "yaml",
		ErrorUnused: true,
		Result:      &payload,
	})
	if err != nil {
		return nil, fmt.Errorf("creating decoder: %w", err)
	}
	if err := decoder.Decode(specMap); err != nil {
		return nil, fmt.Errorf("decoding workspaces: %w", err)
	}
	return payload.Workspaces, nil
}

// broadcastImportManifest sends parsed manifest data to providers that opt in
// via the ImportManifestLoader interface. Providers that don't implement it are skipped.
func broadcastImportManifest(
	pp ProjectProvider,
	manifest *specs.WorkspacesImportMetadata,
) error {
	if manifest == nil {
		return nil
	}

	loader, ok := pp.(provider.ImportManifestLoader)
	if !ok {
		return nil
	}
	return loader.LoadImportManifest(manifest)
}
