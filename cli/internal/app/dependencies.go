package app

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	esProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/workspace"
)

var (
	v string
)

type Providers struct {
	DataCatalog project.Provider
	RETL        project.Provider
	EventStream project.Provider
	Workspace   *workspace.Provider
}

type deps struct {
	client            *client.Client
	providers         *Providers
	compositeProvider project.Provider
}

type Deps interface {
	Client() *client.Client
	Providers() *Providers
	CompositeProvider() project.Provider
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

	p := setupProviders(c)

	cp, err := providers.NewCompositeProvider(p.DataCatalog, p.RETL, p.EventStream)
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

func setupProviders(c *client.Client) *Providers {
	cfg := config.GetConfig()
	dcp := datacatalog.New(catalog.NewRudderDataCatalog(c, cfg.Concurrency.CatalogClient))
	retlp := retl.New(retlClient.NewRudderRETLStore(c))
	esp := esProvider.New(esClient.NewRudderEventStreamStore(c))
	wsp := workspace.New(c)

	return &Providers{
		DataCatalog: dcp,
		RETL:        retlp,
		EventStream: esp,
		Workspace:   wsp,
	}
}

func (d *deps) Client() *client.Client {
	return d.client
}

func (d *deps) Providers() *Providers {
	return d.providers
}

func (d *deps) CompositeProvider() project.Provider {
	return d.compositeProvider
}

func GetVersion() string {
	return v
}
