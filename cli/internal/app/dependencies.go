package app

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
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

	cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
		"datacatalog": p.DataCatalog,
		"retl":        p.RETL,
		"eventstream": p.EventStream,
		"transformations": p.Transformations,
	})
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

	return &Providers{
		DataCatalog: dcp,
		RETL:        retlp,
		EventStream: esp,
		Transformations: trp,
		Workspace:   wsp,
	}, nil
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

func GetVersion() string {
	return v
}
