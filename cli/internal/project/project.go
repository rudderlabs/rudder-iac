package project

import (
	"context"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

var log = logger.New("project")

// Loader defines the interface for loading project specifications.
type Loader interface {
	// Load loads specifications from the specified location.
	Load(location string) (map[string]*specs.RawSpec, error)
}

type ProjectProvider interface {
	provider.SpecLoader
	provider.RuleProvider
	provider.TypeProvider
}

// ImportManifestProvider is a ProjectProvider that also exposes the aggregated
// import metadata for every loaded manifest, for broadcast into the resource
// provider tree. It sits parallel to the resource providers; the
// CompositeProvider never sees the import-manifest kind.
type ImportManifestProvider interface {
	ProjectProvider
	ImportManifest() []specs.WorkspaceImportMetadata
}

type Project interface {
	Location() string
	Load(location string) error
	ResourceGraph() (*resources.Graph, error)
	Specs() map[string]*specs.Spec
}

type project struct {
	location               string
	provider               provider.Provider
	importManifestProvider ImportManifestProvider
	loader                 Loader
	workspaceID            string
	specs                  map[string]*specs.Spec
	validationEngine       validation.ValidationEngine
	renderer               renderer.Renderer
	substitutor            varsubst.Substitutor
	ignoreUnknownKinds     bool
}

// ProjectOption defines a functional option for configuring a Project.
type ProjectOption func(*project)

// WithSpecLoader allows providing a custom SpecLoader.
func WithLoader(l Loader) ProjectOption {
	return func(p *project) {
		if l != nil {
			p.loader = l
		}
	}
}

// WithRenderer allows providing a custom validation Renderer.
// Defaults to a stdout text renderer when unset; tests use this to
// capture rendered diagnostics into a buffer for assertions.
func WithRenderer(r renderer.Renderer) ProjectOption {
	return func(p *project) {
		if r != nil {
			p.renderer = r
		}
	}
}

// WithSubstitutor sets an optional variable substitutor that runs on raw spec
// bytes before parsing. When nil (the default), no substitution happens.
func WithSubstitutor(s varsubst.Substitutor) ProjectOption {
	return func(p *project) {
		p.substitutor = s
	}
}

// WithIgnoreUnknownKinds makes Load skip specs whose kind no registered provider
// owns, instead of failing syntax validation on them.
//
// Only for read-only consumers that need a subset of the project: the local
// typer reads the data catalog and tracking plan, but a project directory may
// also hold specs owned by providers it does not register (data-graphs,
// transformations), which are irrelevant to code generation. The apply path must
// never set this — silently ignoring a spec there would mean not applying it.
func WithIgnoreUnknownKinds() ProjectOption {
	return func(p *project) {
		p.ignoreUnknownKinds = true
	}
}

// WithWorkspaceID sets the resolved workspace ID for the project. This scopes
// workspace-aware operations (e.g. import-manifest broadcast) to only the
// entries belonging to the active workspace.
// When empty, workspace-aware operations fall back to unscoped behavior.
func WithWorkspaceID(id string) ProjectOption {
	return func(p *project) {
		p.workspaceID = id
	}
}

// New creates a new Project instance.
// By default, it uses a loader.Loader.
func New(provider provider.Provider, opts ...ProjectOption) Project {
	p := &project{
		provider:               provider,
		importManifestProvider: importmanifest.New(),
		specs:                  make(map[string]*specs.Spec),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.loader == nil {
		p.loader = &loader.Loader{}
	}

	if p.renderer == nil {
		// fallback to default text renderer if
		// no renderer is provided
		p.renderer = renderer.NewTextRenderer(os.Stdout)
	}

	return p
}

// GetLocation returns the root directory configured for this project, containing all specs.
func (p *project) Location() string {
	return p.location
}

func (p *project) Specs() map[string]*specs.Spec {
	return p.specs
}

func (p *project) loadSpec(path string, spec *specs.Spec) error {
	// Project-level specs (e.g. import-manifest) are handled outside the resource
	// provider tree by their dedicated provider; resource-level specs flow through
	// the version-based dispatch below.
	if classify(spec) == ProjectSpec {
		return p.importManifestProvider.LoadSpec(path, spec)
	}

	switch {
	case spec.IsLegacyVersion():
		return p.provider.LoadLegacySpec(path, spec)
	case spec.Version == specs.SpecVersionV1:
		return p.provider.LoadSpec(path, spec)
	default:
		return fmt.Errorf("unsupported spec version: %s", spec.Version)
	}
}

// Load loads the project specifications from the given location using the
// configured SpecLoader, runs variable substitution if a substitutor is
// configured, then runs the specs through the validation engine (syntax,
// then semantic rules from RuleProvider).
func (p *project) Load(location string) error {
	p.location = location

	rawSpecs, err := p.loader.Load(p.location)
	if err != nil {
		return fmt.Errorf("failed to load specs using specLoader: %w", err)
	}

	if p.substitutor != nil {
		substituted, subDiags := p.substituteSpecs(rawSpecs)
		if subDiags.HasErrors() {
			if err := p.renderer.Render(subDiags); err != nil {
				return fmt.Errorf("rendering diagnostics: %w", err)
			}
			return fmt.Errorf("variable substitution failed")
		}
		rawSpecs = substituted
	}

	return p.handleValidation(rawSpecs)
}

// handleValidation orchestrates the two-phase validation workflow:
// syntactic validation runs first to catch structural issues, and only if that passes,
// we proceed to build the resource graph and run semantic validation.
// This approach avoids expensive graph building when specs have basic syntax errors.
func (p *project) handleValidation(rawSpecs map[string]*specs.RawSpec) error {
	ctx := context.Background()

	// Parse the raw specs into structured form before syntactic validation.
	parsedRawSpecs, specDiags := p.parseSpecs(rawSpecs)

	registry, err := p.registry()
	if err != nil {
		return fmt.Errorf("setting up registry: %w", err)
	}

	engine, err := validation.NewValidationEngine(registry, log)
	if err != nil {
		return fmt.Errorf("initialising validation engine: %w", err)
	}

	// At this point, rawspecs are successfully parsed as well and information
	// parsed gets augmented to the base struct
	syntaxDiags, err := engine.ValidateSyntax(ctx, parsedRawSpecs)
	if err != nil {
		return fmt.Errorf("syntactic validation: %w", err)
	}

	// If any spec or syntax diagnostic errors exist, render the diagnostics and return
	// Both of them are part of the syntax validation although done at different places.
	if specDiags.HasErrors() || syntaxDiags.HasErrors() {
		if err := p.renderer.Render(append(
			specDiags,
			syntaxDiags...,
		)); err != nil {
			return fmt.Errorf("rendering diagnostics: %w", err)
		}
		return fmt.Errorf("syntax validation failed")
	}

	for path, rawSpec := range parsedRawSpecs {
		if err := p.loadSpec(
			path,
			rawSpec.Parsed(),
		); err != nil {
			return fmt.Errorf("loading spec %s: %w", path, err)
		}
	}

	// Broadcast the active workspace's import-manifest into the resource provider
	// tree before graph construction, so handlers attach ImportMetadata from the
	// same maps that inline metadata.import populates. Inline metadata is loaded
	// during the loadSpec loop above; this runs after, so the manifest wins on URN
	// overlap (last writer).
	//
	// Scoping to a single workspace is done here (not in the provider) so a URN
	// maps to at most one remote ID per handler.
	for _, ws := range p.importManifestProvider.ImportManifest() {
		if ws.WorkspaceID != p.workspaceID {
			continue
		}
		if err := p.provider.LoadImportManifest(&ws); err != nil {
			return fmt.Errorf("broadcasting import manifest: %w", err)
		}
		break
	}

	// Graph is built once here - single source of truth for all resource relationships.
	graph, err := p.provider.ResourceGraph()
	if err != nil {
		return fmt.Errorf("building resource graph: %w", err)
	}

	// Cycles make the graph unusable,
	// so detect them before semantic validation
	if _, err := graph.DetectCycles(); err != nil {
		return fmt.Errorf("cycle detected in resource graph: %w", err)
	}

	// Specs which were parsed will now be validated against semantic rules.
	semanticDiags, err := engine.ValidateSemantic(ctx, parsedRawSpecs, graph, p.workspaceID)
	if err != nil {
		return fmt.Errorf("semantic validation: %w", err)
	}

	if err := p.renderer.Render(semanticDiags); err != nil {
		return fmt.Errorf("rendering diagnostics: %w", err)
	}

	if semanticDiags.HasErrors() {
		return fmt.Errorf("semantic validation failed")
	}

	return nil
}

// substituteSpecs runs variable substitution over each raw spec, returning a
// map of fully-substituted specs ready for parsing. Callers must ensure a
// substitutor is configured before calling.
//
// All-or-nothing: if any spec fails substitution, the returned map is nil and
// the diagnostics carry the substitution errors. This stops downstream
// parsing and validation from surfacing cascading false errors (e.g. missing
// references) for resources whose definitions failed substitution.
func (p *project) substituteSpecs(raw map[string]*specs.RawSpec) (map[string]*specs.RawSpec, validation.Diagnostics) {
	var (
		diags       = make(validation.Diagnostics, 0)
		substituted = make(map[string]*specs.RawSpec, len(raw))
	)
	for path, rawSpec := range raw {
		data, subErrs := p.substitutor.SubstituteBytes(rawSpec.Data)
		if len(subErrs) > 0 {
			diags = append(diags, substitutionDiagnostics(path, subErrs)...)
			continue
		}
		substituted[path] = &specs.RawSpec{Data: data}
	}
	if diags.HasErrors() {
		diags.Sort()
		return nil, diags
	}
	return substituted, nil
}

// parseSpecs converts raw spec bytes into parsed specs, collecting any
// per-spec parse errors as diagnostics. Specs that fail to parse are dropped
// from the returned map so downstream validation only sees usable specs.
//
// Ideally this would live in the validation engine, but the engine only
// operates on already-parsed specs.
func (p *project) parseSpecs(raw map[string]*specs.RawSpec) (map[string]*specs.RawSpec, validation.Diagnostics) {
	var (
		diags          = make(validation.Diagnostics, 0)
		parsedRawSpecs = make(map[string]*specs.RawSpec)
	)

	for path, rawSpec := range raw {
		parsed, err := rawSpec.Parse()
		if err != nil {
			diags = append(diags, validation.Diagnostic{
				RuleID:   "project/spec-syntax-parse-valid",
				Severity: rules.Error,
				Message:  fmt.Sprintf("failed to parse spec from path %s: %s", path, err.Error()),
				File:     path,
				Position: pathindex.StartingPosition,
			})
			// Continue to the next spec if the current one is not parsable
			// preventing addition to the input specs map as we can't create specs
			// for rules to validate on.
			continue
		}

		// Dropped before validation so the gatekeeper rules never see the spec at
		// all: the kind belongs to a provider this caller did not register, so
		// every rule keyed off it would be a false error.
		if p.ignoreUnknownKinds && !p.isKnownKind(parsed.Kind) {
			log.Debug("skipping spec of unregistered kind", "path", path, "kind", parsed.Kind)
			continue
		}

		// At this point, we have a valid parsed spec which
		// we need to add to the specs map on the project required by migrate command.
		p.specs[path] = parsed
		parsedRawSpecs[path] = rawSpec
	}

	diags.Sort()
	return parsedRawSpecs, diags
}

// isKnownKind reports whether any registered provider owns the kind. The pattern
// union mirrors BuildRegistry's activePatterns, so a kind is skipped only when
// the gatekeeper rules would have rejected it as unknown.
func (p *project) isKnownKind(kind string) bool {
	providers := []ProjectProvider{p.provider, p.importManifestProvider}

	for _, prov := range providers {
		for _, pattern := range prov.SupportedMatchPatterns() {
			if pattern.Kind == "*" || pattern.Kind == kind {
				return true
			}
		}
	}

	return false
}

func (p *project) registry() (rules.Registry, error) {
	return BuildRegistry(p.provider, p.importManifestProvider, config.GetConfig().ExperimentalFlags.ImportMerge)
}

// BuildRegistry constructs the validation rule registry from the resource
// provider and the project-level import-manifest provider. It is shared by
// project loading and the docs generator so both observe an identical rule set.
//
// The two providers' match patterns are unioned into the active set so the
// gatekeeper rules treat both resource kinds and import-manifest as known. The
// resource-scoped gatekeepers (metadata-syntax-valid, duplicate-urn) are scoped
// to resourcePatterns alone so the engine never hands them a manifest spec.
//
// Import-manifest patterns and dedicated rules are only included when
// importMergeEnabled is set — otherwise the kind is unknown and
// leftover/hand-written manifest files fail syntax validation. The flag is
// resolved by callers and passed in so this stays a pure function of its
// arguments, testable without touching global config.
func BuildRegistry(provider, manifestProvider ProjectProvider, importMergeEnabled bool) (rules.Registry, error) {
	resourcePatterns := provider.SupportedMatchPatterns()

	activePatterns := append([]rules.MatchPattern{}, resourcePatterns...)
	if importMergeEnabled {
		activePatterns = append(activePatterns, manifestProvider.SupportedMatchPatterns()...)
	}

	baseRegistry := rules.NewRegistry(activePatterns)

	syntactic := []rules.Rule{
		// Gatekeeper rules (spec-syntax-valid, resource-kind-version-valid) keep
		// MatchAll AppliesTo and use activePatterns as the source of truth for the
		// supported kinds and versions.
		prules.NewSpecSyntaxValidRule(activePatterns),
		prules.NewResourceKindVersionValidRule(activePatterns),

		// metadata-syntax-valid and duplicate-urn are scoped to resourcePatterns so
		// the engine only hands them resource specs — the import-manifest kind is
		// excluded. See MultiSpecRule filtering.
		prules.NewMetadataSyntaxValidRule(provider.ParseSpec, resourcePatterns),
		prules.NewDuplicateURNRule(provider.ParseSpec, resourcePatterns),
	}
	// Cross-source conflict between import-manifest and inline metadata.import
	// only applies when the import-manifest kind is recognized.
	if importMergeEnabled {
		syntactic = append(syntactic, prules.NewManifestInlineConflictRule(provider.ParseSpec, activePatterns))
	}
	syntactic = append(syntactic, provider.SyntacticRules()...)
	if importMergeEnabled {
		syntactic = append(syntactic, manifestProvider.SyntacticRules()...)
	}

	for _, rule := range syntactic {
		if err := baseRegistry.RegisterSyntactic(rule); err != nil {
			return nil, fmt.Errorf("registering syntactic rule %s: %w", rule.ID(), err)
		}
	}

	semantic := provider.SemanticRules()
	if importMergeEnabled {
		semantic = append(semantic, manifestProvider.SemanticRules()...)
	}
	for _, rule := range semantic {
		if err := baseRegistry.RegisterSemantic(rule); err != nil {
			return nil, fmt.Errorf("registering semantic rule %s: %w", rule.ID(), err)
		}
	}

	return baseRegistry, nil
}

func (p *project) ResourceGraph() (*resources.Graph, error) {
	return p.provider.ResourceGraph()
}
