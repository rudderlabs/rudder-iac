package app

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	accountsProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	dgProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	destProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/builtin"
	esProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/workspace"
	"github.com/rudderlabs/rudder-iac/cli/internal/ruledoc"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/reporters"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var (
	v string
)

// Providers holds instances of all providers used in the application
// Provider types are intentionally set to specific provider implementations
// instead of the generic provider.Provider interface to allow access to
// provider-specific methods if needed.
type Providers struct {
	DataCatalog     *datacatalog.Provider
	RETL            *retl.Provider
	EventStream     *esProvider.Provider
	Transformations *transformations.Provider
	Workspace       *workspace.Provider
	DataGraph       *dgProvider.Provider
	Destination     *destProvider.Provider
	Account         *accountsProvider.Provider
}

type deps struct {
	client            *client.Client
	providers         *Providers
	compositeProvider provider.Provider
}

// Deps defines the dependencies initialized globally for Rudder CLI
type Deps interface {
	// Client returns the RudderStack API client instance, configured with the base URL.
	// Deps from NewOfflineDeps carry a placeholder token, so calls through their
	// client fail authentication.
	Client() *client.Client

	// Providers returns the initialized Providers struct containing all provider instances
	// used in the application when individual provider access is needed.
	Providers() *Providers

	// CompositeProvider returns a composite provider aggregating all individual providers
	// used by components that operate across multiple providers.
	CompositeProvider() provider.Provider

	// Registry builds a validation rule registry from the composite provider so
	// the docs generator observes the same rule set as project validation.
	Registry() (rules.Registry, error)

	// NewProject creates a new project instance with the composite provider.
	NewProject(opts ...project.ProjectOption) project.Project

	// NewDataCatalogProject creates a new project instance with only the DataCatalog provider.
	// Used by trackingplan commands.
	NewDataCatalogProject() project.Project
}

func Initialise(version string) {
	v = version
}

func validateDependencies() error {
	cfg := config.GetConfig()
	if cfg.Auth.AccessToken == "" {
		return fmt.Errorf("access token is required, please run `rudder-cli auth login`, or set the access token via the RUDDERSTACK_ACCESS_TOKEN environment variable")
	}

	return nil
}

func NewDeps() (Deps, error) {
	if err := validateDependencies(); err != nil {
		return nil, err
	}

	return newDeps(config.GetConfig().Auth.AccessToken)
}

// NewOfflineDeps builds the same providers as NewDeps (so it observes the same
// registered rules) for commands that must never contact the API, such as
// validate, which is expected to run in CI without credentials. It skips the
// access-token check and gives the client a placeholder token because client.New
// rejects an empty one; any API call made through it fails auth, which surfaces
// an accidental network dependency instead of hiding it.
func NewOfflineDeps() (Deps, error) {
	return newDeps("offline")
}

// newDeps is the single construction path for NewDeps and NewOfflineDeps, so a
// client option or provider added here reaches both.
func newDeps(token string) (Deps, error) {
	c, err := setupClient(token, v)
	if err != nil {
		return nil, fmt.Errorf("setup client: %w", err)
	}

	cp, p, err := composeProviders(c)
	if err != nil {
		return nil, err
	}

	return &deps{
		client:            c,
		providers:         p,
		compositeProvider: cp,
	}, nil
}

// GenerateRuleCatalog composes the same providers project validation uses and
// hands them to ruledoc.Build, which joins the live rules with the authored
// *.docs.yaml fragments and returns the validated catalog.
//
// Composition is the only part that needs the app's machinery, so it lives
// here in the composition root; the provider-in, catalog-out assembly lives in
// package ruledoc where it can be tested without credentials. Generation makes
// no network calls, so it uses the offline deps.
//
// verrs carries catalog validation failures (e.g. a registered rule with no
// authored fragment); a non-nil error means the catalog could not be assembled
// at all. generatedAt is injected so callers own the timestamp.
func GenerateRuleCatalog(generatedAt string) (docs.DocumentedRules, []error, error) {
	d, err := NewOfflineDeps()
	if err != nil {
		return docs.DocumentedRules{}, nil, fmt.Errorf("building offline deps: %w", err)
	}

	return ruledoc.Build(d.CompositeProvider(), GetVersion(), generatedAt)
}

// composeProviders builds the provider set and aggregates it into a composite
// provider. Shared by NewDeps and NewOfflineDeps so every consumer
// observes the same providers (and therefore the same registered rules).
func composeProviders(c *client.Client) (provider.Provider, *Providers, error) {
	rawProviders, providerMap, err := setupProviders(c)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	cp, err := provider.NewCompositeProvider(providerMap, providerOrder...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize composite provider: %w", err)
	}

	return cp, rawProviders, nil
}

func setupClient(token, version string) (*client.Client, error) {
	cfg := config.GetConfig()
	return client.New(
		token,
		client.WithBaseURL(cfg.APIURL),
		client.WithUserAgent("rudder-cli/"+version),
	)
}

// providerOrder is the order the composite aggregates import --merge matchers
// in. A matcher that consults another provider's matches only finds them if
// that provider's matchers ran first, so a provider comes after everything it
// depends on. providerEdges records those dependencies, and a test keeps the
// two in step: a new cross-provider reference is a one-line addition there.
var providerOrder = []string{
	"account",
	"transformations",
	"datacatalog",
	"destination",
	"datagraph",
	"eventstream",
	"retl",
}

// providerEdges maps each provider to the providers whose matches its
// matchers consult or whose resources its specs reference.
var providerEdges = map[string][]string{
	// transformation: "#transformation:<id>"
	"destination": {"transformations"},
	// connections resolve their destination endpoint (matcher);
	// sources reference a tracking plan
	"eventstream": {"destination", "datacatalog"},
	// connections resolve their destination endpoint (matcher);
	// sources reference an account: "#account:<id>"
	"retl": {"destination", "account"},
}

func setupProviders(c *client.Client) (*Providers, map[string]provider.Provider, error) {
	cfg := config.GetConfig()

	catalogClient, err := catalog.NewRudderDataCatalog(
		c,
		catalog.WithConcurrency(cfg.Concurrency.CatalogClient),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize data catalog client: %w", err)
	}

	// Built ahead of the event-stream provider, whose connection semantic
	// rules read destination definitions. Sharing one populated registry keeps
	// connection checks aligned with the destination provider's validation.
	destRegistry, err := newDestinationRegistry(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize destination registry: %w", err)
	}

	dcp := datacatalog.New(catalogClient)
	retlp := retl.New(retlClient.NewRudderRETLStore(c), retlOptions(destRegistry)...)

	esp := esProvider.New(esClient.NewRudderEventStreamStore(c), esProvider.WithDestinationRegistry(destRegistry))
	trp := transformations.NewProvider(c)
	wsp := workspace.New(c)
	dgp := dgProvider.NewProvider(dgClient.NewRudderDataGraphClient(c), c.Accounts)
	ap := accountsProvider.NewProvider(c.Accounts)
	dp := destProvider.NewProvider(c, destRegistry)

	providers := &Providers{
		DataCatalog:     dcp,
		RETL:            retlp,
		EventStream:     esp,
		Transformations: trp,
		Workspace:       wsp,
		DataGraph:       dgp,
		Account:         ap,
		Destination:     dp,
	}

	providerMap := map[string]provider.Provider{
		"datacatalog":     dcp,
		"retl":            retlp,
		"eventstream":     esp,
		"transformations": trp,
		"datagraph":       dgp,
		"account":         ap,
		"destination":     dp,
	}

	return providers, providerMap, nil
}

// retlOptions returns the RETL provider options for the kinds beyond SQL models.
func retlOptions(destRegistry *definitions.Registry) []retl.Option {
	return []retl.Option{
		retl.WithConnectionSupport(destRegistry),
		retl.WithTableSupport(),
	}
}

// newDestinationRegistry builds the destination definition registry from the
// builtin table. Verified definitions are always registered; unverified
// definitions still require UnverifiedDestinations.
func newDestinationRegistry(cfg config.Config) (*definitions.Registry, error) {
	return builtin.NewRegistry(cfg.ExperimentalFlags.UnverifiedDestinations)
}

func SyncReporter() syncer.SyncReporter {
	if ui.IsTerminal() {
		return &reporters.ProgressSyncReporter{}
	}

	return &reporters.PlainSyncReporter{}
}

func (d *deps) Client() *client.Client {
	return d.client
}

func (d *deps) Providers() *Providers {
	return d.providers
}

func (d *deps) CompositeProvider() provider.Provider {
	return d.compositeProvider
}

// Registry builds a validation rule registry from the composite provider,
// sharing the same construction as project validation so the docs generator
// observes an identical rule set.
func (d *deps) Registry() (rules.Registry, error) {
	return project.BuildRegistry(d.CompositeProvider(), importmanifest.New(), config.GetConfig().ExperimentalFlags.ImportMerge)
}

// NewProject creates a project with composite provider.
func (d *deps) NewProject(opts ...project.ProjectOption) project.Project {
	return project.New(d.CompositeProvider(), opts...)
}

// NewDataCatalogProject creates a project with only the DataCatalog provider.
// Used by trackingplan commands that only need data catalog functionality.
func (d *deps) NewDataCatalogProject() project.Project {
	return project.New(d.Providers().DataCatalog)
}

func GetVersion() string {
	return v
}
