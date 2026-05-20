package project

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// appendImportManifestPattern adds the import-manifest pattern to provider
// patterns when providers have declared their own. When providers return nil
// (meaning "no pattern-based restriction"), the result stays nil to preserve
// that unrestricted behaviour for gatekeeper rules.
func appendImportManifestPattern(providerPatterns []rules.MatchPattern) []rules.MatchPattern {
	if len(providerPatterns) == 0 {
		return nil
	}
	result := make([]rules.MatchPattern, 0, len(providerPatterns)+1)
	result = append(result, providerPatterns...)
	result = append(result, rules.MatchKindVersion(specs.KindImportManifest, specs.SpecVersionV1))
	return result
}

func parseManifestSpec() (*specs.ParsedSpec, error) {
	return &specs.ParsedSpec{
		URNs:               nil,
		LegacyResourceType: "",
	}, nil
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
