package app

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
)

var (
	v string
)

type Providers struct {
	DataCatalog project.Provider
	RETL        project.Provider
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

	cp, err := providers.NewCompositeProvider(p.DataCatalog, p.RETL)
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

	// If APIURL is not the default, ensure it includes /v2 for API versioning
	apiURL := cfg.APIURL
	if apiURL != client.BASE_URL_V2 {
		// Custom URL provided - append /v2 if not already present
		if !strings.HasSuffix(apiURL, "/v2") {
			apiURL = apiURL + "/v2"
		}
	}

	return client.New(
		cfg.Auth.AccessToken,
		client.WithBaseURL(apiURL),
		client.WithUserAgent("rudder-cli/"+version),
	)
}

func setupProviders(c *client.Client) *Providers {
	dcp := datacatalog.New(catalog.NewRudderDataCatalog(c))
	retlp := retl.New()

	return &Providers{
		DataCatalog: dcp,
		RETL:        retlp,
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
