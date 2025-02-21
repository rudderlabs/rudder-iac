package app

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
)

var (
	v string
)

type Deps struct {
	p syncer.Provider
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

func NewDeps() (*Deps, error) {
	if err := validateDependencies(); err != nil {
		return nil, err
	}

	p, err := newProvider(v)
	if err != nil {
		return nil, fmt.Errorf("creating provider: %w", err)
	}

	return &Deps{
		p: p,
	}, nil
}

func newProvider(version string) (syncer.Provider, error) {
	cfg := config.GetConfig()
	rawClient, err := client.New(
		cfg.Auth.AccessToken,
		client.WithBaseURL(cfg.APIURL),
		client.WithUserAgent("rudder-cli/"+version),
	)

	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return provider.NewCatalogProvider(catalog.NewRudderDataCatalog(rawClient)), nil
}

func (d *Deps) Provider() syncer.Provider {
	return d.p
}
