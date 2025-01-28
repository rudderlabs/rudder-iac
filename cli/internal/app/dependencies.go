package app

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
)

var (
	v string
)

type Deps struct {
	p  syncer.Provider
	sm syncer.StateManager
}

func Initialise(version string) {
	v = version
}

func validateDependencies() error {
	cfg := config.GetConfig()
	if cfg.Auth.AccessToken == "" {
		return fmt.Errorf("access token is required, please run `rudder-cli auth login`")
	}

	return nil
}

func NewDeps() (*Deps, error) {
	if err := validateDependencies(); err != nil {
		return nil, err
	}

	sm := newStateManager()

	p, err := newProvider(v)
	if err != nil {
		return nil, fmt.Errorf("creating provider: %w", err)
	}

	return &Deps{
		p:  p,
		sm: sm,
	}, nil
}

func newStateManager() syncer.StateManager {
	return &state.LocalManager{
		BaseDir: config.GetConfigDir(),
	}
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
	return provider.NewCatalogProvider(client.NewRudderDataCatalog(rawClient)), nil
}

func (d *Deps) StateManager() syncer.StateManager {
	return d.sm
}

func (d *Deps) Provider() syncer.Provider {
	return d.p
}
