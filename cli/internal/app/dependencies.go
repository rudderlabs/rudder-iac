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
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	dgProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	esProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/workspace"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/reporters"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
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
}

type deps struct {
	client            *client.Client
	providers         *Providers
	compositeProvider provider.Provider
}

// Deps defines the dependencies initialized globally for Rudder CLI
type Deps interface {
	// Client returns the RudderStack API client instance, configured with authentication and base URL
	Client() *client.Client

	// Providers returns the initialized Providers struct containing all provider instances
	// used in the application when individual provider access is needed.
	Providers() *Providers

	// CompositeProvider returns a composite provider aggregating all individual providers
	// used by components that operate across multiple providers.
	CompositeProvider() provider.Provider

	// NewProject creates a new project instance with the composite provider and wires
	// experimental flags from config. This centralizes project configuration.
	NewProject() project.Project

	// NewDataCatalogProject creates a new project instance with only the DataCatalog provider
	// and wires experimental flags. Used by trackingplan commands.
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

	c, err := setupClient(v)
	if err != nil {
		return nil, fmt.Errorf("setup client: %w", err)
	}

	p, err := setupProviders(c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	providers := map[string]provider.Provider{
		"datacatalog":     p.DataCatalog,
		"retl":            p.RETL,
		"eventstream":     p.EventStream,
		"transformations": p.Transformations,
	}

	// Add data graph provider if experimental flag is enabled
	cfg := config.GetConfig()
	if cfg.ExperimentalFlags.DataGraph {
		providers["datagraph"] = p.DataGraph
	}

	cp, err := provider.NewCompositeProvider(providers)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize composite provider: %w", err)
	}

	return &deps{
		client:            c,
		providers:         p,
		compositeProvider: cp,
	}, nil
}

func setupClient(version string) (*client.Client, error) {
	cfg := config.GetConfig()
	return client.New(
		cfg.Auth.AccessToken,
		client.WithBaseURL(cfg.APIURL),
		client.WithUserAgent("rudder-cli/"+version),
	)
}

func setupProviders(c *client.Client) (*Providers, error) {
	cfg := config.GetConfig()

	catalogClient, err := catalog.NewRudderDataCatalog(
		c,
		catalog.WithConcurrency(cfg.Concurrency.CatalogClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize data catalog client: %w", err)
	}

	dcp := datacatalog.New(catalogClient)
	retlp := retl.New(retlClient.NewRudderRETLStore(c))
	esp := esProvider.New(esClient.NewRudderEventStreamStore(c))
	trp := transformations.NewProvider(c)
	wsp := workspace.New(c)

	providers := &Providers{
		DataCatalog:     dcp,
		RETL:            retlp,
		EventStream:     esp,
		Transformations: trp,
		Workspace:       wsp,
	}

	// Initialize data graph provider if experimental flag is enabled
	if cfg.ExperimentalFlags.DataGraph {
		dgStore := dgClient.NewRudderDataGraphClient(c)
		providers.DataGraph = dgProvider.NewProvider(dgStore)
	}

	return providers, nil
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

// NewProject creates a project with composite provider and wires experimental flags.
// This centralizes project configuration so commands don't need to wire flags manually.
func (d *deps) NewProject() project.Project {
	cfg := config.GetConfig()
	opts := []project.ProjectOption{}

	if cfg.ExperimentalFlags.V1SpecSupport {
		opts = append(opts, project.WithV1SpecSupport())
	}

	if cfg.ExperimentalFlags.ValidationFramework {
		opts = append(opts, project.WithValidateUsingEngine())
	}

	return project.New(d.CompositeProvider(), opts...)
}

// NewDataCatalogProject creates a project with only the DataCatalog provider.
// Used by trackingplan commands that only need data catalog functionality.
func (d *deps) NewDataCatalogProject() project.Project {
	cfg := config.GetConfig()
	opts := []project.ProjectOption{}

	if cfg.ExperimentalFlags.V1SpecSupport {
		opts = append(opts, project.WithV1SpecSupport())
	}

	if cfg.ExperimentalFlags.ValidationFramework {
		opts = append(opts, project.WithValidateUsingEngine())
	}

	return project.New(d.Providers().DataCatalog, opts...)
}

func GetVersion() string {
	return v
}
